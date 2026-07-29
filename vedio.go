package main

import (
	"fmt"
	"reflect"
	"sync"
)

type (
	LifeCycle           int
	registrationOptions struct {
		LifeCycle LifeCycle
		Generator func() (any, error)
	}
	RegistrationOption func(*registrationOptions)
	Scope              interface {
		OnClose(func())
		Closed() bool
	}
)

const (
	SINGLETON LifeCycle = iota
	TRANSIENT
	SCOPED
)

var (
	container map[reflect.Type]func(Scope) (any, error)
)

func init() {
	container = make(map[reflect.Type]func(Scope) (any, error))
}

func Register[T any](opts ...RegistrationOption) error {
	typ := reflect.TypeFor[T]()
	rOpts := &registrationOptions{
		LifeCycle: SINGLETON,
		Generator: func() (any, error) {
			fn, ok := typ.MethodByName("Init")
			if !ok {
				return nil, fmt.Errorf("type %s does not implement an Init function", typ.Name())
			}
			return instantiate(fn)
		},
	}
	for _, opt := range opts {
		opt(rOpts)
	}

	switch rOpts.LifeCycle {
	default:
	case SINGLETON:
		{
			var once sync.Once
			container[typ] = func(_ Scope) (any, error) {
				var val any
				var err error
				once.Do(func() {
					val, err = rOpts.Generator()
				})
				return val, err
			}

		}
	case TRANSIENT:
		{
			container[typ] = func(_ Scope) (any, error) {
				return rOpts.Generator()
			}
		}
	case SCOPED:
		{
			instanceManager := make(map[Scope]func() (any, error))
			var mut sync.Mutex
			container[typ] = func(i Scope) (any, error) {
				if i.Closed() {
					return nil, fmt.Errorf("attempt to resolve type %s on a closed session", typ.Name())
				}
				mut.Lock()
				defer mut.Unlock()
				val, ok := instanceManager[i]
				if !ok {
					i.OnClose(func() {
						mut.Lock()
						defer mut.Unlock()
						delete(instanceManager, i)
					})
					val, err := rOpts.Generator()
					instanceManager[i] = func() (any, error) {
						return val, err
					}
				}
				return val()
			}
		}
	}

	return nil
}

func Resolve[T any](scope Scope) (T, error) {
	typ := reflect.TypeFor[T]()
	fn, ok := container[typ]
	if !ok {
		return reflect.New(typ).Elem().Interface().(T), fmt.Errorf("type %s could not be resolved", typ.Name())
	}
	val, err := fn(scope)
	if err != nil {
		return reflect.New(typ).Elem().Interface().(T), err
	}

	return val.(T), nil
}

func instantiate(method reflect.Method) (any, error) {
	typ := method.Type
	errN := -1
	iter := 0
	for i := range typ.Outs() {
		if i.AssignableTo(reflect.TypeFor[error]()) {
			errN = iter
			break
		}
		iter++
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
		args[i] = *val
	}

	out := method.Func.Call(args)

	if errN != -1 {
		if err := out[errN].Interface(); err != nil {
			return nil, err.(error)
		}
	}
	return val.Interface(), nil
}

func resolve(typ reflect.Type) (*reflect.Value, error) {
	val, ok := container[typ]
	if !ok {
		return nil, fmt.Errorf("type %s cannot be resolved", typ.Name())
	}
	rv := reflect.ValueOf(val)
	return &rv, nil
}

type Test struct{}

func (test *Test) Init() error {
	return nil
}

func main() {
	Register[Test]()
}
