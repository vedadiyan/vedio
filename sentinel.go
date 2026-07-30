package main

type (
	VedioError string
)

const (
	ErrTypeNotFound    VedioError = "type could not found"
	ErrTypeMismatch    VedioError = "type mismatch"
	ErrUnsupportedType VedioError = "type does not implement `Init` method"
	ErrClosedScope     VedioError = "attempt to resolve type on a closed scope"
)

func (err VedioError) Error() string {
	return string(err)
}
