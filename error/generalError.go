package error

import "fmt"

type PolvoErrGeneral int

const (
	ErrInvalidOperation PolvoErrGeneral = iota
	SystemError
	InputError
	InvalidOperationError
	InvalidArgumentError
	InvalidStateError
)

type PolvoGeneralError struct {
	Code   PolvoErrGeneral
	Origin error
	Msg    string
}

func (e PolvoGeneralError) Error() string {
	return fmt.Sprintf("Polvo General Error: %s\n\t: %s", e.Msg, e.Origin.Error())
}
