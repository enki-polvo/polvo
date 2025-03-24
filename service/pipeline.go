package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"polvo/compose"
	perror "polvo/error"
	"polvo/exporter"
	plogger "polvo/logger"
	"polvo/sensorPipe"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type Service interface {
	Start()
	Stop() error
	Wait() error
}

type service struct {
	// dependency injection
	logger plogger.PolvoLogger
	info   *compose.Compose
	// maps for workers
	filterWorkerMap    map[string]*filterWorker
	processorWorkerMap map[string]*processorWorker
	sensorPipeMap      map[string]sensorPipe.Pipe[CommonLogWrapper]
	exporterMap        map[string]exporter.Exporter[CommonLogWrapper]
	// sync pool
	logWrapperPool sync.Pool
	// wait group
	sensorGroup *errgroup.Group
	ctx         context.Context
	wg          sync.WaitGroup
}

func NewService(info *compose.Compose, loger plogger.PolvoLogger) (Service, error) {
	pipeMap := make(map[string][]chan<- *CommonLogWrapper)
	// create service
	sv := new(service)

	// dependency injection
	sv.logger = loger
	sv.info = info
	// create maps
	sv.filterWorkerMap = make(map[string]*filterWorker)
	sv.processorWorkerMap = make(map[string]*processorWorker)
	sv.sensorPipeMap = make(map[string]sensorPipe.Pipe[CommonLogWrapper])
	sv.exporterMap = make(map[string]exporter.Exporter[CommonLogWrapper])
	// init sync pool
	sv.logWrapperPool = sync.Pool{
		New: func() interface{} {
			new := new(CommonLogWrapper)
			new.RefCount = 0
			new.Tag = "NEW"
			return new
		},
	}
	// init error group
	sv.sensorGroup, sv.ctx = errgroup.WithContext(context.Background())

	// create exporter pipes
	for exporterName, exporterInfo := range info.Exporters {
		// create exporter
		switch exporterName {
		case "kafka":
			// TODO: create kafka exporter
			fallthrough
		case "otel":
			// TODO: create otel exporter
			fallthrough
		case "file":
			// create file exporter
			exporter, err := exporter.NewFileExporter(
				exporterInfo.Name,
				0,
				sv.jsonMarshalFunc,
				loger,
				exporterInfo,
			)
			if err != nil {
				return nil, perror.PolvoPipelineError{
					Code:   perror.ErrExporterCreate,
					Origin: err,
					Msg:    "error while construct new pipeline",
				}
			}
			// add to exporter map
			sv.exporterMap[exporterInfo.Name] = exporter
		default:
			return nil, perror.PolvoPipelineError{
				Code:   perror.ErrInvalidExporterName,
				Origin: fmt.Errorf("invalid exporter name %s", exporterInfo.Name),
				Msg:    "error while construct new pipeline",
			}
		}
	}
	// make pipemap
	for _, sensorInfo := range info.Sensors {
		pipeMap[sensorInfo.Name] = make([]chan<- *CommonLogWrapper, 0)
	}

	// create processor workers per pipeline
	for pipelineName, pipelineInfo := range info.Service.Pipeline {
		// create worker per pipeline
		// get exporter channel from exporter map
		exporter, ok := sv.exporterMap[pipelineInfo.Exporter.Name]
		if !ok {
			return nil, perror.PolvoPipelineError{
				Code:   perror.ErrInvalidPipelineCompose,
				Origin: fmt.Errorf("exporter %s not found", pipelineInfo.Exporter.Name),
				Msg:    "error while construct new pipeline",
			}
		}
		processorWorker := newProcessorWorker(pipelineName, &pipelineInfo, exporter.LogChannel())
		sv.processorWorkerMap[pipelineName] = processorWorker

		// add sensor & pipeline info to pipeMap
		for _, sensorInfo := range pipelineInfo.Sensors {
			pipeMap[sensorInfo.Name] = append(pipeMap[sensorInfo.Name], processorWorker.LogChannel())
		}
	}

	// create filter workers & sensorPipe per sensor pipeline
	for _, sensorInfo := range info.Sensors {
		// create filter workers
		sv.filterWorkerMap[sensorInfo.Name] = newFilterWorker(sensorInfo, pipeMap[sensorInfo.Name]...)
		// create worker per sensor
		sensorPipe, err := sensorPipe.NewPipe(sensorInfo.Name, loger, sv.filterWorkerMap[sensorInfo.Name].LogChannel(), sv.jsonUnMarshalFunc)
		if err != nil {
			return nil, perror.PolvoPipelineError{
				Code:   perror.ErrSensorCreate,
				Origin: err,
				Msg:    "error while construct new pipeline",
			}
		}
		sv.sensorPipeMap[sensorInfo.Name] = sensorPipe
	}
	return sv, nil
}

func (s *service) jsonUnMarshalFunc(log string) (*CommonLogWrapper, error) {
	// Reason for control sync pool flow in pipeline is to prevent GC overhead in massive data processing
	// get from sync pool
	common := s.logWrapperPool.Get().(*CommonLogWrapper)
	// set ref count to 0
	atomic.StoreInt32(&common.RefCount, 0)
	// unmarshal json
	err := json.Unmarshal([]byte(log), common)
	if err != nil {
		// if error occurs, return log wrapper to pool
		s.logWrapperPool.Put(common)
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrPipelineUnmarshal,
			Origin: err,
			Msg:    "error while unmarshal pipeline",
		}
	}
	return common, nil
}

func (s *service) jsonMarshalFunc(logWrapper *CommonLogWrapper) (ret []byte, err error) {
	// Reason for control sync pool flow in pipeline is to prevent GC overhead in massive data processing

	ret, err = json.Marshal(logWrapper)
	if err != nil {
		// if ref count is 0, it means this wrapper is unused. return to pool
		if atomic.LoadInt32(&logWrapper.RefCount) == 0 {
			s.logWrapperPool.Put(logWrapper)
		}
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrPipelineMarshal,
			Origin: err,
			Msg:    "error while marshal pipeline",
		}
	}
	// return to sync pool
	// if ref count is 0, it means this wrapper is unused. return to pool
	if atomic.LoadInt32(&logWrapper.RefCount) == 0 {
		// fmt.Printf("Finally Recv: %v\n", logWrapper)
		logWrapper.Tag = "USED"
		s.logWrapperPool.Put(logWrapper)
	}
	return ret, nil
}

func (s *service) Start() {
	// start exporters
	for _, exporter := range s.exporterMap {
		exporter.Start()
		s.wg.Add(1)
	}
	// start processors
	for _, processorWorker := range s.processorWorkerMap {
		processorWorker.Start()
	}
	// start sensors & filter workers
	for _, sensorInfo := range s.info.Sensors {
		// start sensor worker
		s.filterWorkerMap[sensorInfo.Name].Start()
		// run sensor
		s.sensorPipeMap[sensorInfo.Name].Start(sensorInfo.ExecPath, strings.Split(sensorInfo.Param, " ")...)
	}
}

func (s *service) Wait() error {
	var err error

	// register sensorWait to sensorGroup
	for _, sensorInfo := range s.info.Sensors {
		s.sensorGroup.Go(func() error {
			return s.sensorPipeMap[sensorInfo.Name].Wait()
		})
	}

	err = s.sensorGroup.Wait()
	if err != nil {
		s.logger.PrintError("error while wait sensor", err)
		return perror.PolvoPipelineError{
			Code:   perror.ErrPipelineWait,
			Origin: err,
			Msg:    "error while wait sensor",
		}
	}
	// wait for all workers
	s.wg.Wait()

	return nil
}

func (s *service) Stop() error {
	var (
		err       error
		joinedErr error
	)

	joinedErr = nil

	// stop sensors & filter workers
	for _, sensorInfo := range s.info.Sensors {
		// stop sensor
		err = s.sensorPipeMap[sensorInfo.Name].Stop()
		if err != nil {
			if joinedErr != nil {
				joinedErr = errors.Join(joinedErr, perror.PolvoPipelineError{
					Code:   perror.ErrPipelineKill,
					Origin: err,
					Msg:    "error while kill pipeline",
				})
			} else {
				joinedErr = perror.PolvoPipelineError{
					Code:   perror.ErrPipelineKill,
					Origin: err,
					Msg:    "error while kill pipeline",
				}
			}
		}
	}
	// stop filter workers
	for _, filterWorker := range s.filterWorkerMap {
		err = filterWorker.Kill()
		if err != nil {
			if joinedErr != nil {
				joinedErr = errors.Join(joinedErr, perror.PolvoPipelineError{
					Code:   perror.ErrPipelineKill,
					Origin: err,
					Msg:    "error while kill pipeline",
				})
			} else {
				joinedErr = perror.PolvoPipelineError{
					Code:   perror.ErrPipelineKill,
					Origin: err,
					Msg:    "error while kill pipeline",
				}
			}
		}
	}
	// stop processors
	for _, processorWorker := range s.processorWorkerMap {
		err = processorWorker.Kill()
		if err != nil {
			if joinedErr != nil {
				joinedErr = errors.Join(joinedErr, perror.PolvoPipelineError{
					Code:   perror.ErrPipelineKill,
					Origin: err,
					Msg:    "error while kill pipeline",
				})
			} else {
				joinedErr = perror.PolvoPipelineError{
					Code:   perror.ErrPipelineKill,
					Origin: err,
					Msg:    "error while kill pipeline",
				}
			}
		}
	}
	// stop exporters
	for _, exporter := range s.exporterMap {
		err = exporter.Stop()
		if err != nil {
			if joinedErr != nil {
				joinedErr = errors.Join(joinedErr, perror.PolvoPipelineError{
					Code:   perror.ErrPipelineKill,
					Origin: err,
					Msg:    "error while kill pipeline",
				})
			} else {
				joinedErr = perror.PolvoPipelineError{
					Code:   perror.ErrPipelineKill,
					Origin: err,
					Msg:    "error while kill pipeline",
				}
			}
		}
		s.wg.Done()
	}
	return joinedErr
}
