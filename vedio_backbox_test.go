package vedio_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vedadiyan/vedio"
)

type (
	iinterface interface {
		Test() string
	}
	ptrReceiver struct {
		id string
	}
	ptrReceiverComplex  struct{}
	valueReceiver       struct{}
	wrongImplementation struct{}
)

func (x *ptrReceiver) Init() error {
	x.id = uuid.New().String()
	return nil
}

func (x *ptrReceiver) Test() string {
	return x.id
}

func (x *ptrReceiverComplex) Init(str string) error {
	return nil
}

func (x *ptrReceiverComplex) Test() string {
	return "ok"
}

func (x valueReceiver) Init() error {
	return nil
}

func (x valueReceiver) Test() string {
	return "ok"
}

func TestSingletonPtr(t *testing.T) {
	if err := vedio.RegisterFor[iinterface, ptrReceiver](); err != nil {
		t.Error(err)
		return
	}
	v1, err := vedio.Resolve[iinterface]()
	if err != nil {
		t.Error(err)
		return
	}

	v2, err := vedio.Resolve[iinterface]()
	if err != nil {
		t.Error(err)
		return
	}
	if v1.Test() != v2.Test() {
		t.Error("expectation failed")
		return
	}
}

func TestSingletonDoublePtr(t *testing.T) {
	if err := vedio.RegisterFor[iinterface, *ptrReceiver](); err != nil {
		t.Error(err)
		return
	}
	v1, err := vedio.Resolve[iinterface]()
	if err != nil {
		t.Error(err)
		return
	}

	v2, err := vedio.Resolve[iinterface]()
	if err != nil {
		t.Error(err)
		return
	}
	if v1.Test() != v2.Test() {
		t.Error("expectation failed")
		return
	}
}

func TestSingletonNonPtr(t *testing.T) {
	if err := vedio.RegisterFor[iinterface, valueReceiver](); err != nil {
		t.Error(err)
		return
	}
	v1, err := vedio.Resolve[iinterface]()
	if err != nil {
		t.Error(err)
		return
	}

	v2, err := vedio.Resolve[iinterface]()
	if err != nil {
		t.Error(err)
		return
	}
	if v1.Test() != v2.Test() {
		t.Error("expectation failed")
		return
	}
}

func TestSingletonNonPtrAsPtr(t *testing.T) {
	if err := vedio.RegisterFor[iinterface, *valueReceiver](); err != nil {
		t.Error(err)
		return
	}
	v1, err := vedio.Resolve[iinterface]()
	if err != nil {
		t.Error(err)
		return
	}

	v2, err := vedio.Resolve[iinterface]()
	if err != nil {
		t.Error(err)
		return
	}
	if v1.Test() != v2.Test() {
		t.Error("expectation failed")
		return
	}
}
