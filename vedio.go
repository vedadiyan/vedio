package main

import (
	"reflect"
	"sync"
)

type (
	resolver            func(*resolutionContext) (any, error)
	registrationContext struct {
		typ       reflect.Type
		lifeCycle LifeCycle
		generator func() (any, error)
	}
	resolutionContext struct {
		Scope Scope
	}
	LifeCycle          int
	RegistrationOption func(*registrationContext)
	ResolutionOption   func(*resolutionContext)
	Scope              interface {
		ID() string
		OnClose(func())
		Closed() bool
		Close()
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
	ErrInvalidOperation  VedioError = "invalid operation"
	ErrTypeMismatch      VedioError = "type mismatch"
	ErrUnsupportedType   VedioError = "type does not implement `Init` method"
	ErrClosedScope       VedioError = "attempt to resolve type on a closed scope"
)

var (
	container map[reflect.Type]resolver
	mut       sync.Mutex
)

func init() {
	container = make(map[reflect.Type]resolver)
}

func (err VedioError) Error() string {
	return string(err)
}

func LifeCycleOpt(lifeCycle LifeCycle) RegistrationOption {
	return func(rc *registrationContext) {
		rc.lifeCycle = lifeCycle
	}
}

func GeneratorOpt[T any](generator func() (T, error)) RegistrationOption {
	return func(rc *registrationContext) {
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

func newRegistrationContext[I any, T any](opts ...RegistrationOption) (*registrationContext, error) {
	iType := reflect.TypeFor[I]()
	tType := reflect.TypeFor[T]()
	if err := assertTypeMatch(iType, tType); err != nil {
		return nil, err
	}

	out := &registrationContext{
		typ:       iType,
		lifeCycle: SINGLETON,
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
		out.generator = func() (any, error) {
			return instantiate(fn)
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
	return func(_ *resolutionContext) (any, error) {
		once.Do(func() {
			val, err = r.generator()
		})
		return val, err
	}
}

func (r *registrationContext) createTransient() resolver {
	return func(_ *resolutionContext) (any, error) {
		return r.generator()
	}
}

func (r *registrationContext) createScoped() resolver {
	instanceManager := make(map[string]func() (any, error))
	var instanceManagerMut sync.Mutex
	return func(rc *resolutionContext) (any, error) {
		if rc == nil || rc.Scope == nil {
			return nil, ErrInvalidOperation
		}
		if rc.Scope.Closed() {
			return nil, ErrClosedScope
		}
		instanceManagerMut.Lock()
		defer instanceManagerMut.Unlock()
		val, ok := instanceManager[rc.Scope.ID()]
		if !ok {
			defer rc.Scope.OnClose(func() {
				instanceManagerMut.Lock()
				defer instanceManagerMut.Unlock()
				delete(instanceManager, rc.Scope.ID())
			})
			val, err := r.generator()
			instanceManager[rc.Scope.ID()] = func() (any, error) {
				return val, err
			}
		}
		return val()
	}
}

func (r *registrationContext) register() {
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
	rOpts := &resolutionContext{}
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

func RegisterFor[T any, R any](opts ...RegistrationOption) error {
	r, err := newRegistrationContext[T, R](opts...)
	if err != nil {
		return err
	}
	r.register()
	return nil
}

func Register[T any](opts ...RegistrationOption) error {
	return RegisterFor[T, T](opts...)
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

func Zero[T any]() T {
	var zero T
	return zero
}
