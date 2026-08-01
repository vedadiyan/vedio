package vedio

import (
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

type (
	pkgReference        any
	resolver            func(*resolutionContext) (any, error)
	registrationContext struct {
		typ       reflect.Type
		name      string
		lifeCycle LifeCycle
		generator func(*resolutionContext) (any, error)
	}
	resolutionContext struct {
		scope Scoped
	}
	LifeCycle          int
	RegistrationOption func(*registrationContext)
	ResolutionOption   func(*resolutionContext)
	Scoped             interface {
		ID() string
		OnClose(func())
		Closed() bool
		Close()
	}
	Scope struct {
		id        string
		callBacks []func()
		closed    atomic.Bool
		mut       sync.Mutex
	}
	VedioError              string
	Named[T any, N ~string] struct {
		Value T
		alias N
	}
)

const (
	SINGLETON LifeCycle = iota
	TRANSIENT
	SCOPED

	ErrTypeNotFound      VedioError = "type could not be found"
	ErrNilType           VedioError = "nil type detected"
	ErrExpectationFailed VedioError = "expectation failed"
	ErrInvalidOperation  VedioError = "invalid operation"
	ErrDuplicateType     VedioError = "duplicate type registration"
	ErrTypeMismatch      VedioError = "type mismatch"
	ErrUnsupportedType   VedioError = "type does not implement `Init` method"
	ErrClosedScope       VedioError = "attempt to resolve type on a closed scope"

	Default string = "default"
)

var (
	AllowDuplicateRegistration = false
	AllowReadWithNoLock        = false

	container map[reflect.Type]map[string]resolver
	mut       sync.RWMutex

	pkgPath = reflect.TypeFor[pkgReference]().PkgPath()
)

func init() {
	container = make(map[reflect.Type]map[string]resolver)
}

func (err VedioError) Error() string {
	return string(err)
}

func WithLifeCycle(lifeCycle LifeCycle) RegistrationOption {
	return func(rc *registrationContext) {
		rc.lifeCycle = lifeCycle
	}
}

func WithGenerator[T any](generator func() (T, error)) RegistrationOption {
	return func(rc *registrationContext) {
		rc.generator = func(_ *resolutionContext) (any, error) {
			return generator()
		}
	}
}

func WithName(name string) RegistrationOption {
	return func(rc *registrationContext) {
		rc.name = name
	}
}

func WithNameFor[T any]() RegistrationOption {
	return func(rc *registrationContext) {
		rc.name = reflect.TypeFor[T]().Name()
	}
}

func WithScope(scope Scoped) ResolutionOption {
	return func(rc *resolutionContext) {
		rc.scope = scope
	}
}

func assertTypeMatch(interfaceType reflect.Type, implementationType reflect.Type) error {
	if interfaceType == nil || implementationType == nil {
		return ErrNilType
	}
	if implementationType.Kind() == reflect.Interface {
		return ErrExpectationFailed
	}
	if interfaceType.Kind() == reflect.Interface && !implementationType.Implements(interfaceType) {
		if !reflect.PointerTo(implementationType).Implements(interfaceType) {
			return ErrTypeMismatch
		}
	}
	if interfaceType.Kind() != reflect.Interface && interfaceType != implementationType {
		return ErrTypeMismatch
	}
	return nil
}

func newRegistrationContext[I any, T any](opts ...RegistrationOption) (*registrationContext, error) {
	iType := reflect.TypeFor[I]()
	tType := reflect.TypeFor[T]()
	if err := assertTypeMatch(iType, tType); err != nil {
		return nil, err
	}

	out := &registrationContext{
		typ:       iType,
		lifeCycle: SINGLETON,
		name:      Default,
	}

	for _, opt := range opts {
		opt(out)
	}

	if out.generator == nil {
		fn, ok := tType.MethodByName("Init")
		if !ok {
			fn, ok = reflect.PointerTo(tType).MethodByName("Init")
			if !ok {
				return nil, ErrUnsupportedType
			}
		}
		out.generator = func(rc *resolutionContext) (any, error) {
			return instantiate(fn, rc)
		}
	}

	return out, nil
}

func (r *registrationContext) createSingleton() resolver {
	var (
		once sync.Once
		val  any
		err  error
	)
	return func(rc *resolutionContext) (any, error) {
		once.Do(func() {
			val, err = r.generator(rc)
		})
		return val, err
	}
}

func (r *registrationContext) createTransient() resolver {
	return func(rc *resolutionContext) (any, error) {
		return r.generator(rc)
	}
}

func (r *registrationContext) createScoped() resolver {
	var instanceManager sync.Map
	return func(rc *resolutionContext) (any, error) {
		if rc == nil || rc.scope == nil {
			return nil, ErrInvalidOperation
		}
		if rc.scope.Closed() {
			return nil, ErrClosedScope
		}
		val, _ := instanceManager.LoadOrStore(rc.scope.ID(), sync.OnceValue(func() func() (any, error) {
			defer rc.scope.OnClose(func() {
				instanceManager.Delete(rc.scope.ID())
			})
			gen, err := r.generator(rc)
			return func() (any, error) {
				return gen, err
			}
		}))
		return val.(func() (any, error))()
	}
}

func (r *registrationContext) register() error {
	switch r.lifeCycle {
	case SINGLETON:
		{
			return set(r.typ, r.name, r.createSingleton())
		}
	case TRANSIENT:
		{
			return set(r.typ, r.name, r.createTransient())
		}
	case SCOPED:
		{
			return set(r.typ, r.name, r.createScoped())
		}
	}
	return nil
}

func NewScope() Scoped {
	return &Scope{
		id: uuid.New().String(),
	}
}

func (scope *Scope) ID() string {
	return scope.id
}

func (scope *Scope) OnClose(fn func()) {
	scope.mut.Lock()
	defer scope.mut.Unlock()
	scope.callBacks = append(scope.callBacks, fn)
}

func (scope *Scope) Close() {
	if !scope.closed.CompareAndSwap(false, true) {
		return
	}
	for _, cb := range scope.callBacks {
		cb()
	}
	scope.callBacks = nil
}

func (scope *Scope) Closed() bool {
	return scope.closed.Load()
}

func instantiate(method reflect.Method, rc *resolutionContext) (any, error) {
	typ := method.Type
	errN := -1
	if typ.NumOut() > 0 {
		pos := typ.NumOut() - 1
		if typ.Out(pos).AssignableTo(reflect.TypeFor[error]()) {
			errN = pos
		}
	}

	inN := typ.NumIn()
	args := make([]reflect.Value, inN)
	arg0 := typ.In(0)
	isPtr := arg0.Kind() == reflect.Pointer
	if isPtr {
		arg0 = arg0.Elem()
	}
	val := reflect.New(arg0)
	if !isPtr {
		val = val.Elem()
	}
	args[0] = val
	for i := 1; i < inN; i++ {
		in := typ.In(i)
		if isNamedParam(in) {
			val, err := resolveNamedParam(in, rc)
			if err != nil {
				return nil, err
			}
			args[i] = *val
			continue
		}
		val, err := resolveDefault(in, rc)
		if err != nil {
			return nil, err
		}
		args[i] = reflect.ValueOf(val)
	}

	out := method.Func.Call(args)

	if errN != -1 {
		if err := out[errN].Interface(); err != nil {
			return nil, err.(error)
		}
	}
	return val.Interface(), nil
}

func resolve(typ reflect.Type, name string, rc *resolutionContext) (any, error) {
	fn, ok := get(typ, name)
	if !ok {
		return nil, ErrTypeNotFound
	}
	val, err := fn(rc)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func resolveDefault(typ reflect.Type, rc *resolutionContext) (any, error) {
	return resolve(typ, Default, rc)
}

func resolveNamedParam(typp reflect.Type, rc *resolutionContext) (*reflect.Value, error) {
	const valueFieldName = "Value"
	const aliasFieldName = "alias"
	typField, ok := typp.FieldByName(valueFieldName)
	if !ok {
		return nil, ErrExpectationFailed
	}
	typ := typField.Type
	aliasField, ok := typp.FieldByName(aliasFieldName)
	if !ok {
		return nil, ErrExpectationFailed
	}
	val, err := resolve(typ, aliasField.Type.Name(), rc)
	if err != nil {
		return nil, err
	}
	out := reflect.New(typp)
	out.Elem().FieldByName(valueFieldName).Set(reflect.ValueOf(val))
	elem := out.Elem()
	return &elem, nil
}

func RegisterFor[T any, R any](opts ...RegistrationOption) error {
	r, err := newRegistrationContext[T, R](opts...)
	if err != nil {
		return err
	}
	return r.register()
}

func Register[T any](opts ...RegistrationOption) error {
	return RegisterFor[T, T](opts...)
}

func Resolve[T any](opts ...ResolutionOption) (T, error) {
	return ResolveNamed[T](Default, opts...)
}

func ResolveNamed[T any](name string, opts ...ResolutionOption) (T, error) {
	rc := &resolutionContext{}
	for _, opt := range opts {
		opt(rc)
	}
	typ := reflect.TypeFor[T]()
	val, err := resolve(typ, name, rc)
	if err != nil {
		return Zero[T](), err
	}
	if out, ok := val.(T); ok {
		return out, nil
	}
	return Zero[T](), ErrTypeMismatch
}

func get(typ reflect.Type, name string) (resolver, bool) {
	if !AllowReadWithNoLock {
		mut.RLock()
		defer mut.RUnlock()
	}
	base, ok := container[typ]
	if !ok {
		return nil, false
	}

	out, ok := base[name]
	return out, ok
}

func set(typ reflect.Type, name string, rslvr resolver) error {
	mut.Lock()
	defer mut.Unlock()
	val, ok := container[typ]
	if !ok {
		val = make(map[string]resolver)
		container[typ] = val
	}
	if !AllowDuplicateRegistration {
		if _, ok := val[name]; ok {
			return ErrDuplicateType
		}
	}
	val[name] = rslvr
	return nil
}

func isNamedParam(typ reflect.Type) bool {
	return typ.PkgPath() == pkgPath && strings.HasPrefix(typ.Name(), "Named[")
}

func Zero[T any]() T {
	var zero T
	return zero
}
