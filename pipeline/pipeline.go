package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"os"
	perror "polvo/error"
	plogger "polvo/logger"
	"sync"
	"sync/atomic"
)

type Pipeline[logWrapper any] interface {
	// Getter & Setter
	Name() string
	Pipeline() chan logWrapper
	// methods
	Start(string, ...string)
	Stop() error
}

type pipeline[logWrapper any] struct {
	sensorName string
	pipeline   chan logWrapper
	// stream
	readStream  *os.File
	writeStream *os.File
	scanner     *bufio.Scanner
	// thread control
	ctx    context.Context
	cancel context.CancelFunc
	// wait group for scanner thread print all logs before close
	waitScanner sync.WaitGroup
	wrapFunc    func(string) (logWrapper, error)
	promise     Promise
	// conditional variable
	isClosed int32
	// dependency
	logger plogger.PolvoLogger
}

func NewPipeline[logWrapper any](
	sensorName string,
	maxSize uint,
	logger plogger.PolvoLogger,
	wrapFunc func(string) (logWrapper, error)) (Pipeline[logWrapper], error) {

	var err error

	// param check
	if sensorName == "" {
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrInvalidSensorName,
			Origin: fmt.Errorf("invalid sensor name %s", sensorName),
			Msg:    "error while construct new pipeline",
		}
	}
	newPipe := new(pipeline[logWrapper])

	// open stream
	err = newPipe.openStream(maxSize)
	if err != nil {
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrSensorExecute,
			Origin: err,
			Msg:    "error while construct new pipeline",
		}
	}
	newPipe.sensorName = sensorName
	newPipe.scanner = bufio.NewScanner(newPipe.readStream)
	newPipe.ctx, newPipe.cancel = context.WithCancel(context.Background())
	// init waitGroup
	newPipe.waitScanner = sync.WaitGroup{}
	newPipe.promise = nil
	newPipe.wrapFunc = wrapFunc
	// set conditional variable to 0
	atomic.StoreInt32(&newPipe.isClosed, 0)
	// set dependencies
	newPipe.logger = logger
	return newPipe, nil
}

func (p *pipeline[logWrapper]) openStream(maxSize uint) (err error) {
	if atomic.LoadInt32(&p.isClosed) > 0 || p.pipeline != nil {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is already opened"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Open()", p.sensorName),
		}
	}

	// init pipeline
	if maxSize == 0 {
		p.pipeline = make(chan logWrapper)
	} else {
		p.pipeline = make(chan logWrapper, maxSize)
	}

	// init stream
	p.readStream, p.writeStream, err = os.Pipe()
	if err != nil {
		p.logger.PrintError("error while create console %s", err.Error())
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorExecute,
			Origin: err,
			Msg:    "error while construct new pipeline",
		}
	}
	return nil
}

/****************************************************
* Getter & Setter
****************************************************/

func (p *pipeline[logWrapper]) Name() string {
	return p.sensorName
}

func (p *pipeline[logWrapper]) Pipeline() chan logWrapper {
	return p.pipeline
}

/****************************************************
* Pipeline methods
****************************************************/

func (p *pipeline[logWrapper]) Start(arg0 string, arg1 ...string) {
	// start scanner thread
	go p.scannerThread()
	// start sensor thread
	go p.sensorThread(arg0, arg1...)
	// set conditional variable to 0
	atomic.AddInt32(&p.isClosed, 0)
}

func (p *pipeline[logWrapper]) Stop() (err error) {
	// stop sensor thread

	// prevent Call Stop() before Start()
	if p.promise != nil {
		err = p.promise.Cancel()
		if err != nil {
			return perror.PolvoGeneralError{
				Code:   perror.InvalidOperationError,
				Origin: err,
				Msg:    fmt.Sprintf("error while execute pipeline[%s].Stop()", p.sensorName),
			}
		}
	}
	err = p.promise.Cancel()
	if err != nil {
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorPanic,
			Origin: err,
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Stop()", p.sensorName),
		}
	}
	// check if console is already closed
	if atomic.LoadInt32(&p.isClosed) > 0 {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is already closed"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Close()", p.sensorName),
		}
	}
	// send EOF to writeStream
	err = p.writeStream.Close()
	if err != nil {
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorPanic,
			Origin: err,
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Close()", p.sensorName),
		}
	}
	// wait for scanner thread to Flush
	p.waitScanner.Wait()
	// close readStream
	err = p.readStream.Close()
	if err != nil {
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorPanic,
			Origin: err,
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Close()", p.sensorName),
		}
	}
	// close pipeline
	close(p.pipeline)
	p.pipeline = nil
	// set conditional variable to 1
	atomic.AddInt32(&p.isClosed, 1)
	p.promise = nil
	return nil
}

func (p *pipeline[logWrapper]) scannerThread() {
	var (
		log logWrapper
		err error
	)

	p.logger.PrintInfo("pipeline [%s]: scanner thread is started", p.sensorName)
	// set waitGroup
	p.waitScanner.Add(1)
	defer p.waitScanner.Done()
	// read from readStream
	// write to logger
	for p.scanner.Scan() {
		// select {
		// case <-c.ctx.Done():
		// 	c.logger.PrintInfo("console: scanner thread is canceled")
		// 	return
		// default:
		// 	c.logger.PrintInfo("console: %s", c.scanner.Text())
		// }
		log, err = p.wrapFunc(p.scanner.Text())
		if err != nil {
			p.logger.PrintError("pipeline [%s] sensor: error while wrap log. %s", p.sensorName, err.Error())
		}
		// send log to pipeline
		p.pipeline <- log
	}
	if err := p.scanner.Err(); err != nil {
		p.logger.PrintError("pipeline [%s] sensor: error while read from sensor. %s", p.sensorName, err.Error())
	}
	p.logger.PrintInfo("pipeline [%s]: scanner thread is closed", p.sensorName)
}

func (p *pipeline[logWrapper]) sensorThread(argv0 string, argv1 ...string) {
	// execute sensor

	// prevent duplicated sensor thread
	if p.promise != nil {
		return
	}
	// run sensor
	p.promise = Run(os.Stdin, p.writeStream, argv0, argv1...)
	p.logger.PrintInfo("pipeline [%s]: sensor thread is started", p.sensorName)
	// blocked until sensor thread is finished
	if err := p.promise.Wait(); err != nil {
		p.logger.PrintError("pipeline [%s]: error while execute sensor. %s", p.sensorName, err.Error())
	}
	p.logger.PrintInfo("pipeline [%s]: sensor thread is closed", p.sensorName)
}
