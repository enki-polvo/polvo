package pipeline

import (
	"fmt"
	perror "polvo/error"

	"golang.org/x/sync/errgroup"
)

type Pipeline interface {
	// Getter & Setter
	Name() string
	// methods
	Start()
	Stop()
}

type pipeline struct {
	eg         *errgroup.Group
	sensorName string
}

func NewPipeline(sensorName string) (Pipeline, error) {
	// param check
	if sensorName == "" {
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrInvalidSensorName,
			Origin: fmt.Errorf("Invalid sensor name %s", sensorName),
			Msg:    "error while construct new pipeline",
		}
	}

	newPipe := new(pipeline)

	newPipe.sensorName = sensorName
	newPipe.eg = new(errgroup.Group)
	return newPipe, nil
}

func (p *pipeline) Name() string {
	return p.sensorName
}

func (p *pipeline) Start() {

}

func (p *pipeline) Stop() {

}
