package sensorPipe

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

type Pipe[logWrapper any] interface {
	// Getter & Setter
	Name() string
	LogChannel() <-chan logWrapper
	// methods
	Start(string, ...string)
	Stop() error
}

type pipe[logWrapper any] struct {
	sensorName string
	logChannel chan logWrapper
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

func NewPipe[logWrapper any](
	sensorName string,
	maxSize uint,
	logger plogger.PolvoLogger,
	wrapFunc func(string) (logWrapper, error)) (Pipe[logWrapper], error) {

	var err error

	// param check
	if sensorName == "" {
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrInvalidSensorName,
			Origin: fmt.Errorf("invalid sensor name %s", sensorName),
			Msg:    "error while construct new pipeline",
		}
	}
	newPipe := new(pipe[logWrapper])

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

func (p *pipe[logWrapper]) openStream(maxSize uint) (err error) {
	if atomic.LoadInt32(&p.isClosed) > 0 || p.logChannel != nil {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is already opened"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Open()", p.sensorName),
		}
	}

	// init pipeline
	if maxSize == 0 {
		p.logChannel = make(chan logWrapper)
	} else {
		p.logChannel = make(chan logWrapper, maxSize)
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

func (p *pipe[logWrapper]) Name() string {
	return p.sensorName
}

func (p *pipe[logWrapper]) LogChannel() <-chan logWrapper {
	return p.logChannel
}

/****************************************************
* Pipeline methods
****************************************************/

func (p *pipe[logWrapper]) Start(arg0 string, arg1 ...string) {
	// start scanner thread
	go p.scannerThread()
	// start sensor thread
	go p.sensorThread(arg0, arg1...)
	// set conditional variable to 0
	atomic.AddInt32(&p.isClosed, 0)
}

func (p *pipe[logWrapper]) Stop() (err error) {
	// stop sensor thread

	// prevent Call Stop() before Start()
	if p.promise == nil {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is not started"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Stop()", p.sensorName),
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
	// close logChannel
	close(p.logChannel)
	p.logChannel = nil
	// set conditional variable to 1
	atomic.AddInt32(&p.isClosed, 1)
	p.promise = nil
	return nil
}

func (p *pipe[logWrapper]) scannerThread() {
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
		log, err = p.wrapFunc(p.scanner.Text())
		if err != nil {
			p.logger.PrintError("pipeline [%s] sensor: error while wrap log. %s", p.sensorName, err.Error())
			continue
		}
		// send log to pipeline
		p.logChannel <- log
	}
	if err := p.scanner.Err(); err != nil {
		p.logger.PrintError("pipeline [%s] sensor: error while read from sensor. %s", p.sensorName, err.Error())
	}
	p.logger.PrintInfo("pipeline [%s]: scanner thread is closed", p.sensorName)
}

func (p *pipe[logWrapper]) sensorThread(argv0 string, argv1 ...string) {
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
