package service

import (
	"context"
	"polvo/compose"
	"sync"
	"sync/atomic"
	"time"
)

// #processorWorker
//
// processorWorker is a worker that process logs from filterWorker.
// It receives logs from filterWorker and sends them to the next exporter.
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

func (p *processorWorker) LogChannel() chan<- *CommonLogWrapper {
	return p.inboundChannel
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
	// This is useless live lock. Because in golang, channel can pop data when it is closed.
	// So, this is just for safety.
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
			// In Processor Worker, we can process log with multiple sensors.
			// Process Function chain will be added here.
			// e.g. parser, signature detect, etc.
			// fmt.Fprintf(os.Stderr, "Processor Worker[%s] received log from Filter Worker[%s]\n", p.Name, log.Tag)

			// decrease ref count
			atomic.AddInt32(&log.RefCount, -1)
			p.outboundChannel <- log
			p.waitForEndRemainTasks.Done()
		}
	}
}
