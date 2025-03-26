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
	"polvo/service/model"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

// # Service
//
// Service is an interface that corresponds to Service in the compose file.
// Service reflects the pipeline structure of the data flow from the sensor to the exporter.
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
	sensorPipeMap      map[string]sensorPipe.Pipe[model.CommonLogWrapper]
	exporterMap        map[string]exporter.Exporter[model.CommonLogWrapper]
	// sync pool
	logWrapperPool sync.Pool
	// wait group
	sensorGroup *errgroup.Group
	ctx         context.Context
	wg          sync.WaitGroup
}

// NewService creates a new service with the given compose file and logger.
// Initialize sequence is as follows:
// 1. Create service struct & maps to control workers.
// 2. Create exporter.
// 3. Create processor workers per pipeline.
// 4. Create filter workers & sensorPipe per sensor pipeline.
// Stop function operates in the opposite order.
func NewService(info *compose.Compose, loger plogger.PolvoLogger) (Service, error) {
	// create service
	svc := new(service)

	// dependency injection
	svc.logger = loger
	svc.info = info

	// init sync pool
	svc.logWrapperPool = sync.Pool{
		New: func() interface{} {
			new := new(model.CommonLogWrapper)
			new.RefCount = 0
			new.Tag = "NEW"
			return new
		},
	}

	// init error group
	svc.sensorGroup, svc.ctx = errgroup.WithContext(context.Background())

	// create worker maps
	svc.filterWorkerMap = make(map[string]*filterWorker)
	svc.processorWorkerMap = make(map[string]*processorWorker)
	svc.sensorPipeMap = make(map[string]sensorPipe.Pipe[model.CommonLogWrapper])
	svc.exporterMap = make(map[string]exporter.Exporter[model.CommonLogWrapper])

	// create exporters
	err := svc.createExporters(*info, loger)
	if err != nil {
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrExporterCreate,
			Origin: err,
			Msg:    "error while construct new pipeline",
		}
	}

	// create processor workers per pipeline
	err = svc.createProcessors(*info, loger)
	if err != nil {
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrProcessorCreate,
			Origin: err,
			Msg:    "error while construct new pipeline",
		}
	}

	// make pipemap to connect workers
	// add sensor & pipeline info to pipeMap
	// The relationship between the sensor and the pipeline is as follows:
	// multiple sensors -> single filterWorker per sensors -> processorWorker per pipelines -> exporter
	pipeMap := make(map[string][]chan<- *model.CommonLogWrapper)
	for _, sensorInfo := range info.Sensors {
		pipeMap[sensorInfo.Name] = make([]chan<- *model.CommonLogWrapper, 0)
	}
	for pipelineName, pipelineInfo := range info.Service.Pipeline {
		for _, sensorInfo := range pipelineInfo.Sensors {
			pipeMap[sensorInfo.Name] = append(pipeMap[sensorInfo.Name], svc.processorWorkerMap[pipelineName].LogChannel())
		}
	}

	// create filter workers & sensorPipe per sensor pipeline
	err = svc.createFilterAndSensors(*info, loger, pipeMap)
	if err != nil {
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrSensorCreate,
			Origin: err,
			Msg:    "error while construct new sensorPipe",
		}
	}
	return svc, nil
}

func (svc *service) createExporters(info compose.Compose, loger plogger.PolvoLogger) error {
	// create exporters
	// exporterName is predefined in
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
				svc.jsonMarshalFunc,
				loger,
				exporterInfo,
			)
			if err != nil {
				return perror.PolvoPipelineError{
					Code:   perror.ErrExporterCreate,
					Origin: err,
					Msg:    "error while construct new pipeline",
				}
			}
			// add to exporter map
			svc.exporterMap[exporterInfo.Name] = exporter
		default:
			return perror.PolvoPipelineError{
				Code:   perror.ErrInvalidExporterName,
				Origin: fmt.Errorf("invalid exporter name %s", exporterInfo.Name),
				Msg:    "error while construct new exporters",
			}
		}
		// print info
		loger.PrintInfo("exporter [%s] created", exporterName)
	}
	return nil
}

func (svc *service) createProcessors(info compose.Compose, loger plogger.PolvoLogger) error {
	// create processor workers per pipeline
	for pipelineName, pipelineInfo := range info.Service.Pipeline {
		// create worker per pipeline
		// get exporter channel from exporter map
		exporter, ok := svc.exporterMap[pipelineInfo.Exporter.Name]
		if !ok {
			return perror.PolvoPipelineError{
				Code:   perror.ErrInvalidPipelineCompose,
				Origin: fmt.Errorf("exporter %s not found", pipelineInfo.Exporter.Name),
				Msg:    "error while construct new processors",
			}
		}
		processorWorker := newProcessorWorker(pipelineName, &pipelineInfo, exporter.LogChannel())
		svc.processorWorkerMap[pipelineName] = processorWorker
		// print info
		loger.PrintInfo("processor [%s] created", pipelineName)
	}
	return nil
}

func (svc *service) createFilterAndSensors(info compose.Compose, loger plogger.PolvoLogger, pipeMap map[string][]chan<- *model.CommonLogWrapper) error {
	// create filter workers & sensorPipe per sensor pipeline
	for _, sensorInfo := range info.Sensors {
		// create filter workers
		filterWorker := newFilterWorker(sensorInfo, pipeMap[sensorInfo.Name]...)
		svc.filterWorkerMap[sensorInfo.Name] = filterWorker
		// create worker per sensor
		sensorPipe, err := sensorPipe.NewPipe(sensorInfo.Name, loger, filterWorker.LogChannel(), svc.jsonUnMarshalFunc)
		if err != nil {
			return perror.PolvoPipelineError{
				Code:   perror.ErrSensorCreate,
				Origin: err,
				Msg:    "error while construct new sensorPipe",
			}
		}
		svc.sensorPipeMap[sensorInfo.Name] = sensorPipe

		// print info
		loger.PrintInfo("filter [%s] created", sensorInfo.Name)
		loger.PrintInfo("sensor [%s] created", sensorInfo.Name)
	}
	return nil
}

/************************************************************************************************************
* Service public methods
************************************************************************************************************/

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

/************************************************************************************************************
* Service private methods
************************************************************************************************************/

func (s *service) jsonUnMarshalFunc(log string) (*model.CommonLogWrapper, error) {
	// Reason for control sync pool flow in pipeline is to prevent GC overhead in massive data processing
	// get from sync pool
	common := s.logWrapperPool.Get().(*model.CommonLogWrapper)
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

func (s *service) jsonMarshalFunc(logWrapper *model.CommonLogWrapper) (ret []byte, err error) {
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
