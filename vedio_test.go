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

	mockInterface interface {
		Test() string
	}
	mockImplementation  struct{}
	wrongImplementation struct{}
)

func (x *mockImplementation) Init() {

}

func (x *mockImplementation) Test() string {
	return "ok"
}

func Test_assertTypeMatch(t *testing.T) {
	tests := []testCase{
		{"correct implementation - not pointer", reflect.TypeFor[mockInterface](), reflect.TypeFor[mockImplementation](), false},
		{"correct implementation - pointer", reflect.TypeFor[mockInterface](), reflect.TypeFor[*mockImplementation](), false},
		{"correct implementation - struct to struct not pointer", reflect.TypeFor[mockImplementation](), reflect.TypeFor[mockImplementation](), false},
		{"correct implementation - struct to struct pointer", reflect.TypeFor[*mockImplementation](), reflect.TypeFor[*mockImplementation](), false},
		{"wrong implementation - not pointer", reflect.TypeFor[mockInterface](), reflect.TypeFor[wrongImplementation](), true},
		{"wrong implementation - pointer", reflect.TypeFor[mockInterface](), reflect.TypeFor[*wrongImplementation](), true},
		{"wrong usage - interface to interface", reflect.TypeFor[mockInterface](), reflect.TypeFor[mockInterface](), true},
		{"wrong usage - pointer to interface", reflect.TypeFor[*mockInterface](), reflect.TypeFor[mockInterface](), true},
		{"wrong usage - interface to pointer", reflect.TypeFor[mockInterface](), reflect.TypeFor[*mockInterface](), true},
		{"wrong usage - struct to pointer", reflect.TypeFor[mockImplementation](), reflect.TypeFor[*mockImplementation](), true},
		{"wrong usage - pointer to struct", reflect.TypeFor[*mockImplementation](), reflect.TypeFor[mockImplementation](), true},
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
