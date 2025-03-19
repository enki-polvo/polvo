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
	"time"
)

type Pipe[log any] interface {
	// Getter & Setter
	Name() string
	LogChannel() <-chan *log
	// methods
	Start(string, ...string)
	Stop() error
}

type pipe[log any] struct {
	sensorName string
	logChannel chan *log
	// stream
	readStream  *os.File
	writeStream *os.File
	scanner     *bufio.Scanner
	// thread control
	ctx    context.Context
	cancel context.CancelFunc
	// wait group for scanner thread print all logs before close
	waitScanner sync.WaitGroup
	wrapFunc    func(string) (*log, error)
	promise     Promise
	// conditional variable
	isClosed int32
	// dependency
	logger plogger.PolvoLogger
}

func NewPipe[log any](
	sensorName string,
	maxSize uint,
	logger plogger.PolvoLogger,
	wrapFunc func(string) (*log, error)) (Pipe[log], error) {

	var err error

	// param check
	if sensorName == "" {
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrInvalidSensorName,
			Origin: fmt.Errorf("invalid sensor name %s", sensorName),
			Msg:    "error while construct new pipeline",
		}
	}
	newPipe := new(pipe[log])

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

func (p *pipe[log]) openStream(maxSize uint) (err error) {
	if atomic.LoadInt32(&p.isClosed) > 0 || p.logChannel != nil {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is already opened"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Open()", p.sensorName),
		}
	}

	// init pipeline
	if maxSize == 0 {
		p.logChannel = make(chan *log)
	} else {
		p.logChannel = make(chan *log, maxSize)
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

func (p *pipe[log]) Name() string {
	return p.sensorName
}

func (p *pipe[log]) LogChannel() <-chan *log {
	return p.logChannel
}

/****************************************************
* Pipeline methods
****************************************************/

func (p *pipe[log]) Start(arg0 string, arg1 ...string) {
	// start scanner thread
	go p.scannerThread()
	// start sensor thread
	go p.sensorThread(arg0, arg1...)
	// set conditional variable to 0
	atomic.AddInt32(&p.isClosed, 0)
}

func (p *pipe[log]) Stop() (err error) {
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

	// close logChannel with goroutine. wait until all logs are exported by filter worker thread.
	go p.pipeCloser()

	// set conditional variable to 1
	atomic.AddInt32(&p.isClosed, 1)
	p.promise = nil
	return nil
}

func (p *pipe[log]) scannerThread() {
	var (
		lg  *log
		err error
	)

	p.logger.PrintInfo("pipeline [%s]: scanner thread is started", p.sensorName)
	// set waitGroup
	p.waitScanner.Add(1)
	defer p.waitScanner.Done()
	// read from readStream
	// write to logger
	for p.scanner.Scan() {
		// fmt.Printf("log: %v\n", p.scanner.Text())
		lg, err = p.wrapFunc(p.scanner.Text())
		if err != nil {
			// if error while wrap log, just skip this log
			p.logger.PrintError("pipeline [%s] sensor: error while wrap log. %s", p.sensorName, err.Error())
			continue
		}
		// send log to pipeline
		// fmt.Printf("log: %v\n", lg)
		p.logChannel <- lg
	}
	if err = p.scanner.Err(); err != nil {
		// EOF
		p.logger.PrintError("pipeline [%s] sensor: error while read from sensor. %s", p.sensorName, err.Error())
	}
	p.logger.PrintInfo("pipeline [%s]: scanner thread is closed", p.sensorName)
}

func (p *pipe[log]) pipeCloser() {
	// wait until all logs are exported in logChannel
	for {
		if len(p.logChannel) == 0 {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	// close logChannel
	close(p.logChannel)
	p.logChannel = nil

}

func (p *pipe[logWrapper]) sensorThread(argv0 string, argv1 ...string) {
	var err error
	// execute sensor
	// prevent duplicated sensor thread
	if p.promise != nil {
		return
	}
	// run sensor
	p.promise, err = Run(os.Stdin, p.writeStream, argv0, argv1...)
	if err != nil {
		// if error occurs, uncontrollable error. so panic
		p.logger.PrintError("failed to start pipeline [%s]: %v", p.sensorName, err)
		panic(err)
	}
	p.logger.PrintInfo("pipeline [%s]: sensor thread is started", p.sensorName)
	// blocked until sensor thread is finished
	if err := p.promise.Wait(); err != nil {
		// kill process to prevent zombie process
		err = p.Stop()
		if err != nil {
			// if error occurs, uncontrollable error. so panic
			p.logger.PrintError("pipeline [%s]: error while stop sensor[%s]. panic...", p.sensorName, err.Error())
			// TODO: handle panic
			return
		}
		p.logger.PrintError("pipeline [%s]: error while execute sensor[%s]. kill sensor...", p.sensorName, err.Error())
	}
	p.logger.PrintInfo("pipeline [%s]: sensor thread is closed", p.sensorName)
}
