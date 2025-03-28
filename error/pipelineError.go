package error

import "fmt"

type PolvoErrPipeline int

const (
	ErrInvalidSensorName PolvoErrPipeline = iota
	ErrSensorCreate
	ErrSensorExecute
	ErrSensorPanic
	ErrInvalidPipelineCompose
	ErrPipelineKill
	ErrPipelineWait
	ErrPipelineUnmarshal
	ErrPipelineMarshal
	ErrInvalidExporterName
	ErrFilterWorkerCreate
	ErrExporterCreate
	ErrProcessorCreate
)

type PolvoPipelineError struct {
	Code   PolvoErrPipeline
	Origin error
	Msg    string
}

func (e PolvoPipelineError) Error() string {
	return fmt.Sprintf("Polvo Pipeline Error: %s\n\t: %s", e.Msg, e.Origin.Error())
}
