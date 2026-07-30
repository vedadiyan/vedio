package main

import (
	"reflect"
	"testing"
)

type (
	testCase struct {
		name string // description of this test case
		// Named input parameters for target function.
		interfaceType      reflect.Type
		implementationType reflect.Type
		wantErr            bool
	}

	iinterface interface {
		Test() string
	}
	ptrReceiver         struct{}
	valueReceiver       struct{}
	wrongImplementation struct{}
)

func (x *ptrReceiver) Init() {

}

func (x *ptrReceiver) Test() string {
	return "ok"
}

func (x valueReceiver) Test() string {
	return "ok"
}

func Test_assertTypeMatch(t *testing.T) {
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
