package main

import (
	"reflect"
	"sync"
)

type (
	LifeCycle           int
	Resolver            func(*ResolutionContext) (any, error)
	RegistrationContext struct {
		typ       reflect.Type
		lifeCycle LifeCycle
		generator func() (any, error)
	}
	ResolutionContext struct {
		Scope Scope
	}
	RegistrationOption func(*RegistrationContext)
	ResolutionOption   func(*ResolutionContext)
	Scope              interface {
		OnClose(func())
		Closed() bool
		Close()
		Open()
	}
	VedioError string
)

const (
	SINGLETON LifeCycle = iota
	TRANSIENT
	SCOPED

	ErrTypeNotFound      VedioError = "type could not found"
	ErrNilType           VedioError = "nil type detected"
	ErrExpectationFailed VedioError = "expectation failed"
	ErrTypeMismatch      VedioError = "type mismatch"
	ErrUnsupportedType   VedioError = "type does not implement `Init` method"
	ErrClosedScope       VedioError = "attempt to resolve type on a closed scope"
)

var (
	container map[reflect.Type]Resolver
	mut       sync.Mutex
)

func init() {
	container = make(map[reflect.Type]Resolver)
}

func (err VedioError) Error() string {
	return string(err)
}

func LifeCycleOpt(lifeCycle LifeCycle) RegistrationOption {
	return func(rc *RegistrationContext) {
		rc.lifeCycle = lifeCycle
	}
}

func GeneratorOpt[T any](generator func() (T, error)) RegistrationOption {
	return func(rc *RegistrationContext) {
		rc.generator = func() (any, error) {
			return generator()
		}
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

func NewRegistrationContext[I any, T any](opts ...RegistrationOption) (*RegistrationContext, error) {
	iType := reflect.TypeFor[I]()
	tType := reflect.TypeFor[T]()
	if err := assertTypeMatch(iType, tType); err != nil {
		return nil, err
	}

	out := &RegistrationContext{
		typ:       iType,
		lifeCycle: SINGLETON,
		generator: func() (any, error) {
			fn, ok := tType.MethodByName("Init")
			if !ok {
				return nil, ErrUnsupportedType
			}
			return instantiate(fn)
		},
	}
	for _, opt := range opts {
		opt(out)
	}

	return out, nil
}

func (r *RegistrationContext) createSingleton() Resolver {
	var once sync.Once
	var val any
	var err error
	return func(_ *ResolutionContext) (any, error) {
		once.Do(func() {
			val, err = r.generator()
		})
		return val, err
	}
}

func (r *RegistrationContext) createTransient() Resolver {
	return func(_ *ResolutionContext) (any, error) {
		return r.generator()
	}
}

func (r *RegistrationContext) createScoped() Resolver {
	instanceManager := make(map[Scope]func() (any, error))
	var instanceManagerMut sync.Mutex
	return func(rc *ResolutionContext) (any, error) {
		if rc.Scope.Closed() {
			return nil, ErrClosedScope
		}
		instanceManagerMut.Lock()
		defer instanceManagerMut.Unlock()
		val, ok := instanceManager[rc.Scope]
		if !ok {
			defer rc.Scope.OnClose(func() {
				instanceManagerMut.Lock()
				defer instanceManagerMut.Unlock()
				delete(instanceManager, rc.Scope)
			})
			val, err := r.generator()
			instanceManager[rc.Scope] = func() (any, error) {
				return val, err
			}
		}
		return val()
	}
}

func (r *RegistrationContext) Register() {
	mut.Lock()
	defer mut.Unlock()
	switch r.lifeCycle {
	case SINGLETON:
		{
			container[r.typ] = r.createSingleton()
		}
	case TRANSIENT:
		{
			container[r.typ] = r.createTransient()
		}
	case SCOPED:
		{
			container[r.typ] = r.createScoped()
		}
	}
}

func Resolve[T any](opts ...ResolutionOption) (T, error) {
	typ := reflect.TypeFor[T]()
	val, err := resolve(typ, opts...)
	if err != nil {
		return Zero[T](), err
	}
	if out, ok := val.(T); ok {
		return out, nil
	}
	return Zero[T](), ErrTypeMismatch
}

func instantiate(method reflect.Method) (any, error) {
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
	val := reflect.New(typ.In(0)).Elem()
	args[0] = val
	for i := 1; i < inN; i++ {
		val, err := resolve(typ.In(i))
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

func resolve(typ reflect.Type, opts ...ResolutionOption) (any, error) {
	rOpts := &ResolutionContext{}
	for _, opt := range opts {
		opt(rOpts)
	}
	fn, ok := container[typ]
	if !ok {
		return nil, ErrTypeNotFound
	}
	val, err := fn(rOpts)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func Zero[T any]() T {
	var zero T
	return zero
}
