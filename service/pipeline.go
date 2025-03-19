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
	"time"
)

type filterWorker struct {
	// status variables & context
	isRunning int32
	ctx       context.Context
	cancel    context.CancelFunc
	// dependency injection
	info        *compose.SensorInfo // TODO: processor info will be added.
	eventHeader map[string][]string
	// sensor pipe
	inboundChannel <-chan *CommonLogWrapper
	// outbound pipes
	outboundChannel []chan<- *CommonLogWrapper
	// wait group for filter thread
	waitForEndRemainTasks sync.WaitGroup
}

func newFilterWorker(info *compose.SensorInfo, logChannel <-chan *CommonLogWrapper, outboundChan ...chan<- *CommonLogWrapper) *filterWorker {
	nw := new(filterWorker)

	// dependency injection
	nw.info = info
	nw.eventHeader = info.EventsHeader
	// context
	nw.ctx, nw.cancel = context.WithCancel(context.Background())
	// set channels
	nw.inboundChannel = logChannel
	nw.outboundChannel = make([]chan<- *CommonLogWrapper, 0)
	nw.outboundChannel = append(nw.outboundChannel, outboundChan...)
	// init sync pool
	return nw
}

func (w *filterWorker) Start() {
	atomic.StoreInt32(&w.isRunning, 1)
	go w.filterThread()
}

func (w *filterWorker) Kill() error {
	// wait until all logs are processed
	for {
		if len(w.inboundChannel) == 0 {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	// cancel context to kill worker loop
	w.cancel()
	w.waitForEndRemainTasks.Wait()
	atomic.StoreInt32(&w.isRunning, 0)
	return nil
}

func (w *filterWorker) filterThread() {
	var (
		log             *CommonLogWrapper
		outboundChannel chan<- *CommonLogWrapper
	)

	for {
		select {
		case <-w.ctx.Done():
			return
		case log = <-w.inboundChannel:
			w.waitForEndRemainTasks.Add(1)
			// process log
			// TODO: filter log
			// send to outbound channels
			for _, outboundChannel = range w.outboundChannel {
				// add ref count if log is sent to another worker
				atomic.AddInt32(&log.RefCount, 1)
				outboundChannel <- log
			}
			w.waitForEndRemainTasks.Done()
		}
	}
}

type processorWorker struct {
	Name string
	// status variables & context
	isRunning int32
	ctx       context.Context
	cancel    context.CancelFunc
	// dependency injection
	info                 *compose.PipelineInfo // TODO: processor info will be added.
	eventHeaderPerSensor map[string]map[string][]string
	// sensor pipe
	inboundChannel chan *CommonLogWrapper
	// outbound pipes
	outboundChannel chan<- *CommonLogWrapper
	// wait group for processor thread
	waitForEndRemainTasks sync.WaitGroup
}

func newProcessorWorker(name string, info *compose.PipelineInfo, exporterChan chan<- *CommonLogWrapper) *processorWorker {
	nw := new(processorWorker)

	// set name
	nw.Name = name
	nw.eventHeaderPerSensor = make(map[string]map[string][]string)
	// dependency injection
	nw.info = info
	// context
	nw.ctx, nw.cancel = context.WithCancel(context.Background())
	// set event headers
	for _, sensorInfo := range nw.info.Sensors {
		// set event headers
		nw.eventHeaderPerSensor[sensorInfo.Name] = sensorInfo.EventsHeader
	}
	// set channel
	nw.inboundChannel = make(chan *CommonLogWrapper)
	nw.outboundChannel = exporterChan
	// init sync pool
	return nw
}

func (p *processorWorker) Start() {
	atomic.StoreInt32(&p.isRunning, 1)
	go p.processorThread()
}

func (p *processorWorker) Kill() error {
	// wait until all logs are processed
	for {
		if len(p.inboundChannel) == 0 {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	// cancel context to kill worker loop
	p.cancel()
	p.waitForEndRemainTasks.Wait()
	atomic.StoreInt32(&p.isRunning, 0)
	// close channel
	close(p.inboundChannel)
	return nil
}

func (p *processorWorker) processorThread() {
	var (
		log *CommonLogWrapper
	)

	for {
		select {
		case <-p.ctx.Done():
			return
		case log = <-p.inboundChannel:
			p.waitForEndRemainTasks.Add(1)
			// fmt.Printf("Processor %s\tRecv: %v\n", p.Name, log)
			// TODO: process log
			// send to outbound channels

			// decrease ref count
			atomic.AddInt32(&log.RefCount, -1)
			p.outboundChannel <- log
			p.waitForEndRemainTasks.Done()
		}
	}
}

func (p *processorWorker) LogChannel() chan<- *CommonLogWrapper {
	return p.inboundChannel
}

type Service interface {
	Start()
	Stop() error
	Wait()
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
	wg sync.WaitGroup
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

	// create sensor pipes
	for _, sensorInfo := range info.Sensors {
		// create worker per sensor
		sensorPipe, err := sensorPipe.NewPipe(sensorInfo.Name, 0, loger, sv.jsonUnMarshalFunc)
		if err != nil {
			return nil, perror.PolvoPipelineError{
				Code:   perror.ErrSensorCreate,
				Origin: err,
				Msg:    "error while construct new pipeline",
			}
		}
		sv.sensorPipeMap[sensorInfo.Name] = sensorPipe

		// add to pipe map
		pipeMap[sensorInfo.Name] = make([]chan<- *CommonLogWrapper, 0)
	}

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

	// create filter workers per sensor pipeline
	for _, sensorInfo := range info.Sensors {
		// create worker per sensor
		sensorPipe, ok := sv.sensorPipeMap[sensorInfo.Name]
		if !ok {
			return nil, perror.PolvoPipelineError{
				Code:   perror.ErrSensorCreate,
				Origin: fmt.Errorf("sensor %s not found", sensorInfo.Name),
				Msg:    "error while construct new pipeline",
			}
		}
		// create filter workers
		sv.filterWorkerMap[sensorInfo.Name] = newFilterWorker(sensorInfo, sensorPipe.LogChannel(), pipeMap[sensorInfo.Name]...)
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
		fmt.Printf("Finally Recv: %v\n", logWrapper)
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

func (s *service) Wait() {
	s.wg.Wait()
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
	}
	return joinedErr
}
