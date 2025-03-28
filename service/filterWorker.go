package service

import (
	"context"
	"polvo/compose"
	"polvo/service/filter"
	"polvo/service/model"
	"sync"
	"sync/atomic"
	"time"
)

// #filterWorker
//
// filterWorker is a worker that filter logs from sensorPipe.
// It receives logs from sensorPipe by using sync.Pool, and send them to the next processor worker.
// It use sync.Pool to reuse log objects.
type filterWorker struct {
	// status variables & context
	isRunning int32
	ctx       context.Context
	cancel    context.CancelFunc
	// dependency injection
	info        *compose.SensorInfo // TODO: processor info will be added.
	eventHeader map[string][]string
	// sensor pipe
	inboundChannel chan *model.CommonLogWrapper
	// outbound pipes
	outboundChannel []chan<- *model.CommonLogWrapper
	// filter Operator
	filterOperator        filter.FilterOperator
	returnLogObjectToPool func(*model.CommonLogWrapper)
	// wait group for filter thread
	waitForEndRemainTasks sync.WaitGroup
}

func (fw *filterWorker) LogChannel() chan<- *model.CommonLogWrapper {
	return fw.inboundChannel
}

func newFilterWorker(filterOperator filter.FilterOperator,
	returnLogObjectToPool func(*model.CommonLogWrapper),
	info *compose.SensorInfo,
	outboundChan ...chan<- *model.CommonLogWrapper) *filterWorker {

	nw := new(filterWorker)

	// dependency injection
	nw.info = info
	nw.eventHeader = info.EventsHeader
	// init filter operator
	// The relationship between filterWorker and filterOperator is has-a relationship.
	// There are multiple filterWorkers and each filterWorker uses a singleton filterOperator.
	// However, I judged it to be threadsafe because no write operation occurs in filteroperator.
	nw.filterOperator = filterOperator
	nw.returnLogObjectToPool = returnLogObjectToPool
	// context
	nw.ctx, nw.cancel = context.WithCancel(context.Background())
	// set channels
	nw.inboundChannel = make(chan *model.CommonLogWrapper)
	nw.outboundChannel = make([]chan<- *model.CommonLogWrapper, 0)
	nw.outboundChannel = append(nw.outboundChannel, outboundChan...)
	return nw
}

func (fw *filterWorker) Start() {
	atomic.StoreInt32(&fw.isRunning, 1)
	go fw.filterThread()
}

func (fw *filterWorker) Kill() error {
	// wait until all logs are processed
	// This is useless live lock. Because in golang, channel can pop data when it is closed.
	// So, this is just for safety.
	for {
		if len(fw.inboundChannel) == 0 {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	// cancel context to kill worker loop
	fw.cancel()
	fw.waitForEndRemainTasks.Wait()
	atomic.StoreInt32(&fw.isRunning, 0)
	// close channel
	close(fw.inboundChannel)
	return nil
}

func (fw *filterWorker) filterThread() {
	var (
		log             *model.CommonLogWrapper
		outboundChannel chan<- *model.CommonLogWrapper
	)

	for {
		select {
		case <-fw.ctx.Done():
			return
		case log = <-fw.inboundChannel:
			fw.waitForEndRemainTasks.Add(1)
			// filter log
			if fw.filterOperator.Operation(log) {
				// drop log if it is filtered
				// put log to sync.Pool
				// fmt.Fprintf(os.Stderr, "log is filtered: %v\n", log)
				fw.returnLogObjectToPool(log)
				fw.waitForEndRemainTasks.Done()
				continue
			}

			// send to outbound channels
			for _, outboundChannel = range fw.outboundChannel {
				// add ref count if log is sent to another worker
				atomic.AddInt32(&log.RefCount, 1)
				outboundChannel <- log
			}
			fw.waitForEndRemainTasks.Done()
		}
	}
}
