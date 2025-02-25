package error

import "fmt"

type PolvoErrCompose int

const (
	ErrInvalidCompose PolvoErrCompose = iota
	UnsupportedOsError
	UnsupportedArchError
	SensorNotFoundError
)

type PolvoComposeError struct {
	Code   PolvoErrCompose
	Origin error
	Msg    string
}

func (e PolvoComposeError) Error() string {
	return fmt.Sprintf("Polvo Compose Error: %s\n\t: %s", e.Msg, e.Origin.Error())
}
