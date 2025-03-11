package service

import (
	"context"
	"encoding/json"
	"polvo/compose"
	perror "polvo/error"
	plogger "polvo/logger"
	"polvo/sensorPipe"
	"strings"
	"sync"
	"sync/atomic"
)

// DEPRECATED
// func CreateDynamicLogWrapperFromSensorInfo(fields []string) reflect.Type {
// 	// create struct fields
// 	structFields := make([]reflect.StructField, 0)
// 	// create struct field dinamically from param
// 	for _, field := range fields {
// 		structFields = append(structFields, reflect.StructField{
// 			Name:    field,
// 			Type:    reflect.TypeOf(""),
// 			Tag:     reflect.StructTag(`json:"` + field + `"`),
// 			PkgPath: "",
// 		})
// 	}
// 	// create struct type
// 	return reflect.StructOf(structFields)
// }

type worker struct {
	// status variables & context
	isRunning int32
	ctx       context.Context
	cancel    context.CancelFunc
	// dependency injection
	info *compose.SensorInfo // TODO: processor info will be added.
	// sensor pipe
	inboundChannel <-chan CommonHeaderWrapper
	// outbound pipes
	outboundChannels map[string]chan interface{}
}

func newWorker(info *compose.SensorInfo, logChannel <-chan CommonHeaderWrapper) *worker {
	nw := new(worker)

	// dependency injection
	nw.info = info
	// context
	nw.ctx, nw.cancel = context.WithCancel(context.Background())
	// set channels
	nw.inboundChannel = logChannel
	nw.outboundChannels = make(map[string]chan interface{})
	return nw
}

func (w *worker) Start() {
	atomic.StoreInt32(&w.isRunning, 1)
	go w.processorThread()
}

func (w *worker) Kill() error {
	// cancel context to kill worker loop
	w.cancel()
	atomic.StoreInt32(&w.isRunning, 0)
	return nil
}

func (w *worker) processorThread() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.inboundChannel:
			// process log
		}
	}
}

type Pipeline interface{}

type pipeline struct {
	// dependency injection
	logger plogger.PolvoLogger
	info   *compose.PipelineInfo
	// maps
	workerMap     map[string]*worker
	sensorPipeMap map[string]sensorPipe.Pipe[CommonHeaderWrapper]
	// sync pool
	marshalPool sync.Pool
}

func NewPipeline(info *compose.PipelineInfo, loger plogger.PolvoLogger) (Pipeline, error) {
	pl := new(pipeline)

	// dependency injection
	pl.logger = loger
	pl.info = info
	// create maps
	pl.workerMap = make(map[string]*worker)
	pl.sensorPipeMap = make(map[string]sensorPipe.Pipe[CommonHeaderWrapper])
	// init sync pool
	pl.marshalPool = sync.Pool{
		New: func() interface{} {
			return new(CommonHeaderWrapper)
		},
	}

	for _, sensorInfo := range info.Sensors {
		// create worker per sensor
		sensorPipe, err := sensorPipe.NewPipe(sensorInfo.Name, 0, loger, pl.jsonUnMarshalFunc)
		if err != nil {
			return nil, perror.PolvoPipelineError{
				Code:   perror.ErrSensorCreate,
				Origin: err,
				Msg:    "error while construct new pipeline",
			}
		}
		pl.sensorPipeMap[sensorInfo.Name] = sensorPipe
		pl.workerMap[sensorInfo.Name] = newWorker(sensorInfo, sensorPipe.LogChannel())
		// connect multiple exporter to worker

	}
	return pl, nil
}

func (p *pipeline) jsonUnMarshalFunc(log string) (CommonHeaderWrapper, error) {
	// get from sync pool
	common := p.marshalPool.Get().(*CommonHeaderWrapper)
	// put back to sync pool
	defer p.marshalPool.Put(common)
	// unmarshal json
	err := json.Unmarshal([]byte(log), common)
	if err != nil {
		return *common, err
	}
	return *common, nil
}

func (p *pipeline) Start() {
	for _, sensorInfo := range p.info.Sensors {
		// start sensor worker
		p.workerMap[sensorInfo.Name].Start()
		// run sensor
		p.sensorPipeMap[sensorInfo.Name].Start(sensorInfo.ExecPath, strings.Split(sensorInfo.Param, " ")...)
	}
}

func (p *pipeline) Stop() error {
	for _, sensorInfo := range p.info.Sensors {
		// stop sensor worker
		p.workerMap[sensorInfo.Name].Kill()
		// stop sensor
		err := p.sensorPipeMap[sensorInfo.Name].Stop()
		if err != nil {
			return perror.PolvoPipelineError{
				Code:   perror.ErrPipelineKill,
				Origin: err,
				Msg:    "error while kill pipeline",
			}
		}
	}
	return nil
}
