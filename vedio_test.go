package main

import (
	"reflect"
	"testing"
)

type (
	iinterface interface {
		Test() string
	}
	ptrReceiver         struct{}
	valueReceiver       struct{}
	wrongImplementation struct{}
)

func (x *ptrReceiver) Init() error {
	return nil
}

func (x *ptrReceiver) Test() string {
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
				t.Fatal("assertTypeMatch() succeeded unexpectedly")
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
			return newRegistrationContext[iinterface, ptrReceiver](LifeCycleOpt(TRANSIENT))
		}, func(rc *registrationContext) bool { return rc.lifeCycle == TRANSIENT && rc.generator != nil }, false},
		{"correct usage - generator option not pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, ptrReceiver](GeneratorOpt(generatorNonPointer))
		}, func(rc *registrationContext) bool { return rc.lifeCycle == SINGLETON && rc.generator != nil }, false},
		{"correct usage - no options pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, *ptrReceiver]()
		}, func(rc *registrationContext) bool { return rc.lifeCycle == SINGLETON && rc.generator != nil }, false},
		{"correct usage - lifecyle option pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, *ptrReceiver](LifeCycleOpt(TRANSIENT))
		}, func(rc *registrationContext) bool { return rc.lifeCycle == TRANSIENT && rc.generator != nil }, false},
		{"correct usage -  generator option pointer", func() (*registrationContext, error) {
			return newRegistrationContext[iinterface, *ptrReceiver](GeneratorOpt(generatorPointer))
		}, func(rc *registrationContext) bool { return rc.lifeCycle == SINGLETON && rc.generator != nil }, false},

		{"wronf usage -  no init method", func() (*registrationContext, error) {
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
				t.Fatal("newRegistrationContext succeeded unexpectedly")
			}
			if !tt.test(res) {
				t.Fatal("newRegistrationContext succeeded unexpectedly")
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

	method, _ := reflect.TypeFor[*ptrReceiver]().MethodByName("Init")

	tests := []testCase{
		{"correct usage", method, &ptrReceiver{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := instantiate(tt.method)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("instantiate() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("instantiate() succeeded unexpectedly")
			}
			if !reflect.ValueOf(got).Type().AssignableTo(reflect.ValueOf(tt.want).Type()) {
				t.Errorf("instantiate() = %v, want %v", got, tt.want)
			}
		})
	}
}
