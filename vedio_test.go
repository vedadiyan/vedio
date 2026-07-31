package vedio

import (
	"fmt"
	"reflect"
	"testing"
)

type (
	iinterface interface {
		Test() string
	}
	ptrReceiver         struct{}
	ptrReceiverComplex  struct{}
	valueReceiver       struct{}
	wrongImplementation struct{}
)

func (x *ptrReceiver) Init() error {
	return nil
}

func (x *ptrReceiver) Test() string {
	return "ok"
}

func (x *ptrReceiverComplex) Init(str string) error {
	return nil
}

func (x *ptrReceiverComplex) Test() string {
	return "ok"
}

func (x valueReceiver) Test() string {
	return "ok"
}

func Test_assertTypeMatch(t *testing.T) {
	type testCase struct {
		name               string
		interfaceType      reflect.Type
		implementationType reflect.Type
		wantErr            bool
	}
	tests := []testCase{
		{"correct usage - not pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[ptrReceiver](), false},
		{"correct usage - pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[*ptrReceiver](), false},
		{"correct usage - struct to struct not pointer", reflect.TypeFor[ptrReceiver](), reflect.TypeFor[ptrReceiver](), false},
		{"correct usage - struct to struct pointer", reflect.TypeFor[*ptrReceiver](), reflect.TypeFor[*ptrReceiver](), false},
		{"correct usage - value receiver not pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[valueReceiver](), false},
		{"correct usage - value receiver pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[*valueReceiver](), false},
		{"correct usage - none struct type", reflect.TypeFor[string](), reflect.TypeFor[string](), false},
		{"wrong usage - not pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[wrongImplementation](), true},
		{"wrong usage - pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[*wrongImplementation](), true},
		{"wrong usage - interface to interface", reflect.TypeFor[iinterface](), reflect.TypeFor[iinterface](), true},
		{"wrong usage - pointer to interface", reflect.TypeFor[*iinterface](), reflect.TypeFor[iinterface](), true},
		{"wrong usage - interface to pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[*iinterface](), true},
		{"wrong usage - struct to pointer", reflect.TypeFor[ptrReceiver](), reflect.TypeFor[*ptrReceiver](), true},
		{"wrong usage - pointer to struct", reflect.TypeFor[*ptrReceiver](), reflect.TypeFor[ptrReceiver](), true},
		{"wrong usage - nil", nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := assertTypeMatch(tt.interfaceType, tt.implementationType)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("assertTypeMatch() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Errorf("assertTypeMatch() succeeded unexpectedly")
			}
		})
	}
}

func TestNewRegistrationContext(t *testing.T) {
	type testCase struct {
		name    string
		fn      func() (*registrationContext, error)
		test    func(*registrationContext) bool
		wantErr bool
	}

	generatorNonPointer := func() (ptrReceiver, error) { return ptrReceiver{}, nil }

	generatorPointer := func() (valueReceiver, error) { return valueReceiver{}, nil }

	tests := []testCase{
		{"correct usage - no options not pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, ptrReceiver]()
		}, func(rc *registrationContext) bool { return rc.lifeCycle == SINGLETON && rc.generator != nil }, false},
		{"correct usage - lifecyle option not pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, ptrReceiver](WithLifeCycle(TRANSIENT))
		}, func(rc *registrationContext) bool { return rc.lifeCycle == TRANSIENT && rc.generator != nil }, false},
		{"correct usage - generator option not pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, ptrReceiver](WithGenerator(generatorNonPointer))
		}, func(rc *registrationContext) bool { return rc.lifeCycle == SINGLETON && rc.generator != nil }, false},
		{"correct usage - no options pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, *ptrReceiver]()
		}, func(rc *registrationContext) bool { return rc.lifeCycle == SINGLETON && rc.generator != nil }, false},
		{"correct usage - lifecyle option pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, *ptrReceiver](WithLifeCycle(TRANSIENT))
		}, func(rc *registrationContext) bool { return rc.lifeCycle == TRANSIENT && rc.generator != nil }, false},
		{"correct usage -  generator option pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, *ptrReceiver](WithGenerator(generatorPointer))
		}, func(rc *registrationContext) bool { return rc.lifeCycle == SINGLETON && rc.generator != nil }, false},

		{"wrong usage -  no init method", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, valueReceiver]()
		}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.fn()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("newRegistrationContext failed: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Errorf("newRegistrationContext succeeded unexpectedly")
				return
			}
			if !tt.test(res) {
				t.Errorf("newRegistrationContext succeeded unexpectedly")
			}
		})
	}
}

func Test_instantiate(t *testing.T) {

	type testCase struct {
		name    string
		method  reflect.Method
		want    any
		wantErr bool
	}

	simpleMethpd, _ := reflect.TypeFor[*ptrReceiver]().MethodByName("Init")
	complexMethpd, _ := reflect.TypeFor[*ptrReceiverComplex]().MethodByName("Init")
	key := reflect.TypeFor[string]()
	container[key] = map[string]resolver{Default: func(rc *resolutionContext) (any, error) {
		return "ok", nil
	}}
	defer delete(container, key)

	tests := []testCase{
		{"correct usage - simple", simpleMethpd, &ptrReceiver{}, false},
		{"correct usage - complex", complexMethpd, &ptrReceiverComplex{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := instantiate(tt.method, &resolutionContext{name: Default})
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("instantiate() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Errorf("instantiate() succeeded unexpectedly")
				return
			}
			if !reflect.ValueOf(got).Type().AssignableTo(reflect.ValueOf(tt.want).Type()) {
				t.Errorf("instantiate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_resolve(t *testing.T) {

	type testCase struct {
		name    string
		typ     reflect.Type
		opts    []ResolutionOption
		want    any
		wantErr bool
	}

	type CorrectKey bool
	type IncorrectKey bool
	type ErrorKey bool
	type ScopedKey bool

	correctKey := reflect.TypeFor[CorrectKey]()
	incorrectKey := reflect.TypeFor[IncorrectKey]()
	errorKey := reflect.TypeFor[ErrorKey]()
	scopedKey := reflect.TypeFor[ScopedKey]()

	container[correctKey] = map[string]resolver{Default: func(rc *resolutionContext) (any, error) {
		return true, nil
	}}
	container[errorKey] = map[string]resolver{Default: func(rc *resolutionContext) (any, error) {
		return nil, fmt.Errorf("expected error")
	}}
	container[scopedKey] = map[string]resolver{Default: func(rc *resolutionContext) (any, error) {
		if rc == nil || rc.scope == nil || rc.scope.ID() == "" {
			return nil, fmt.Errorf("scoped is null")
		}
		return true, nil
	}}
	defer delete(container, correctKey)
	defer delete(container, errorKey)

	tests := []testCase{
		{"correct usage - simple", correctKey, nil, true, false},
		{"correct usage - scoped", scopedKey, []ResolutionOption{WithScope(NewScope())}, true, false},
		{"correct usage - expected error", errorKey, nil, nil, true},
		{"wrong usage - unresolved type", incorrectKey, nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &resolutionContext{
				name: Default,
			}
			for _, opt := range tt.opts {
				opt(rc)
			}
			got, gotErr := resolve(tt.typ, rc)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("resolve() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("resolve() succeeded unexpectedly")
			}
			if !reflect.ValueOf(got).Type().AssignableTo(reflect.ValueOf(tt.want).Type()) {
				t.Errorf("resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}
