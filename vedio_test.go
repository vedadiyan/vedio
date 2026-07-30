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
		{"correct implementation - not pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[ptrReceiver](), false},
		{"correct implementation - pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[*ptrReceiver](), false},
		{"correct implementation - struct to struct not pointer", reflect.TypeFor[ptrReceiver](), reflect.TypeFor[ptrReceiver](), false},
		{"correct implementation - struct to struct pointer", reflect.TypeFor[*ptrReceiver](), reflect.TypeFor[*ptrReceiver](), false},
		{"correct implementation - value receiver not pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[valueReceiver](), false},
		{"correct implementation - value receiver pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[*valueReceiver](), false},
		{"wrong implementation - not pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[wrongImplementation](), true},
		{"wrong implementation - pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[*wrongImplementation](), true},
		{"wrong usage - interface to interface", reflect.TypeFor[iinterface](), reflect.TypeFor[iinterface](), true},
		{"wrong usage - pointer to interface", reflect.TypeFor[*iinterface](), reflect.TypeFor[iinterface](), true},
		{"wrong usage - interface to pointer", reflect.TypeFor[iinterface](), reflect.TypeFor[*iinterface](), true},
		{"wrong usage - struct to pointer", reflect.TypeFor[ptrReceiver](), reflect.TypeFor[*ptrReceiver](), true},
		{"wrong usage - pointer to struct", reflect.TypeFor[*ptrReceiver](), reflect.TypeFor[ptrReceiver](), true},
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
