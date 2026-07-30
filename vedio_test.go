package main

import (
	"reflect"
	"testing"
)

type (
	mockInterface interface {
		Test() string
	}
	mockImplementation struct{}
)

func (x *mockImplementation) Init() {

}

func (x *mockImplementation) Test() string {
	return "ok"
}

func Test_assertTypeMatch(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		interfaceType      reflect.Type
		implementationType reflect.Type
		wantErr            bool
	}{
		struct {
			name               string
			interfaceType      reflect.Type
			implementationType reflect.Type
			wantErr            bool
		}{"correct implementation - not pointer", reflect.TypeFor[mockInterface](), reflect.TypeFor[mockImplementation](), false},
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
