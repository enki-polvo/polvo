package error

import "fmt"

type PolvoErrFilter int

const (
	ErrDetectionField PolvoErrFilter = iota
	ErrRuleField
)

type PolvoFilterError struct {
	Code   PolvoErrFilter
	Origin error
	Msg    string
}

func (e PolvoFilterError) Error() string {
	return fmt.Sprintf("Polvo Filter Error: %s\n\t: %s", e.Msg, e.Origin.Error())
}
