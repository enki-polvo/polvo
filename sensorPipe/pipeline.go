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

	"golang.org/x/sync/errgroup"
)

// # Pipe
//
// Pipe runs the sensor as a subprocess and loads the output values ​​into a channel.
// It provides a way to start and stop the sensor and get the output values.
// The values ​​are preprocessed using wrapFunc.
type Pipe[log any] interface {
	// Getter & Setter
	Name() string
	LogChannel() <-chan *log
	// methods
	Start(string, ...string) error
	Wait() error
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
	ctx context.Context
	eg  *errgroup.Group
	// wait group for scanner thread print all logs before close
	waitScanner sync.WaitGroup
	wrapFunc    func(string) (*log, error)
	promise     Promise
	// conditional variable
	isStarted  int32
	isClosed   int32
	waitBefore int32
	// dependency
	logger plogger.PolvoLogger
}

// # NewPipe
//
// NewPipe creates a new pipeline with the given sensor name, maximum size, logger, and wrapFunc.
// Maximum size is the maximum number of logs that can be stored in the logChannel.
// 0 means no limit.
// WrapFunc is a function that converts the log string to the desired log structure.
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
	// init thread control
	newPipe.eg, newPipe.ctx = errgroup.WithContext(context.Background())
	// init waitGroup
	newPipe.waitScanner = sync.WaitGroup{}
	newPipe.promise = nil
	newPipe.wrapFunc = wrapFunc
	// set conditional variable to 0
	atomic.StoreInt32(&newPipe.isClosed, 0)
	atomic.StoreInt32(&newPipe.isStarted, 0)
	atomic.StoreInt32(&newPipe.waitBefore, 0)
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

// # Start
//
// Start starts the sensor and scanner threads.
// It returns an error if the sensor is already started.
func (p *pipe[log]) Start(arg0 string, arg1 ...string) error {
	// prevent duplicated sensor thread
	if atomic.LoadInt32(&p.isStarted) > 0 || p.promise != nil {
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorExecute,
			Origin: fmt.Errorf("sensor is already started"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Start()", p.sensorName),
		}
	}
	// start scanner thread
	// scannerThread is not managed by errgroup because it will be terminated by EOF.
	go p.scannerThread()

	// start sensor thread
	p.eg.Go(func() error {
		return p.sensorThread(arg0, arg1...)
	})
	// set conditional variable to 0
	atomic.AddInt32(&p.isClosed, 0)
	atomic.AddInt32(&p.isStarted, 1)
	return nil
}

// # Stop
//
// Stop stops the sensor and scanner threads.
// It will not be blocked until all logs are exported by the filter worker thread.
// Instead, pipeCloser goroutine will be called to close the logChannel. pipeCloser will be blocked by live lock until all logs are exported.
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
	err = p.flushStreams()
	if err != nil {
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorPanic,
			Origin: err,
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Close()", p.sensorName),
		}
	}
	// set conditional variable to 1
	atomic.AddInt32(&p.isClosed, 1)
	p.promise = nil
	return nil
}

// # Wait
//
// Wait waits for the sensor and scanner threads to finish.
func (p *pipe[log]) Wait() error {
	if atomic.LoadInt32(&p.isClosed) > 0 || atomic.LoadInt32(&p.isStarted) != 1 {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is not started"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Wait()", p.sensorName),
		}
	}
	// check duplicated wait
	if atomic.LoadInt32(&p.waitBefore) > 0 {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is already waited"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Wait()", p.sensorName),
		}
	}
	// set conditional variable to 1
	atomic.AddInt32(&p.waitBefore, 1)
	err := p.eg.Wait()
	if err != nil {
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorPanic,
			Origin: err,
			Msg:    fmt.Sprintf("error while wait for pipeline[%s]", p.sensorName),
		}
	}
	return nil
}

/****************************************************
* goroutines & private methods
****************************************************/

// # flushStreams
//
// flush stream generates EOF, ensuring termination of the scanner thread and closing the file stream.
func (p *pipe[log]) flushStreams() error {
	// send EOF to writeStream
	err := p.writeStream.Close()
	if err != nil {
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorPanic,
			Origin: err,
			Msg:    fmt.Sprintf("error while execute pipeline[%s].streamCloser", p.sensorName),
		}
	}
	// wait until scanner thread is finished
	p.waitScanner.Wait()
	// close readStream
	err = p.readStream.Close()
	if err != nil {
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorPanic,
			Origin: err,
			Msg:    fmt.Sprintf("error while execute pipeline[%s].streamCloser", p.sensorName),
		}
	}
	return nil
}

func (p *pipe[log]) scannerThread() error {
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
		lg, err = p.wrapFunc(p.scanner.Text())
		if err != nil {
			// if error while wrap log, just skip this log
			p.logger.PrintError("pipeline [%s] sensor: error while wrap log. %s", p.sensorName, err.Error())
			continue
		}
		// send log to pipeline
		p.logChannel <- lg
	}
	if err = p.scanner.Err(); err != nil {
		// EOF
		p.logger.PrintError("pipeline [%s] sensor: error while read from sensor. %s", p.sensorName, err.Error())
	}
	p.logger.PrintInfo("pipeline [%s]: scanner thread is closed", p.sensorName)
	fmt.Fprintf(os.Stderr, "scannerThread is closed\n")
	return nil
}

func (p *pipe[logWrapper]) sensorThread(argv0 string, argv1 ...string) error {
	var err error
	// execute sensor
	p.promise, err = Run(os.Stdin, p.writeStream, argv0, argv1...)
	if err != nil {
		// if error occurs, uncontrollable error. so panic
		p.logger.PrintError("failed to start pipeline [%s]: %v", p.sensorName, err)
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorExecute,
			Origin: err,
			Msg:    fmt.Sprintf("error in sensor[%s] thread", p.sensorName),
		}
	}
	p.logger.PrintInfo("pipeline [%s]: sensor thread is started", p.sensorName)
	// blocked until sensor thread is finished
	if err := p.promise.Wait(); err != nil {
		// (DEPRECATED)
		// if error returned from Wait(), it means that subprocess returns exitcode and already released.
		//
		// // kill process to prevent zombie process
		// err = p.Stop()
		// if err != nil {
		// 	// if error occurs, uncontrollable error. so panic
		// 	p.logger.PrintError("pipeline [%s]: error while stop sensor[%s]. panic...", p.sensorName, err.Error())
		// 	return
		// }

		// wait until all logs are exported in logChannel
		// This is useless live lock. Because in golang, channel can pop data when it is closed.
		// So, this is just for safety.
		p.logger.PrintError("pipeline [%s]: error while sensor running... Error: %v", p.sensorName, err.Error())
		for {
			if len(p.logChannel) == 0 {
				break
			}
			time.Sleep(1 * time.Millisecond)
		}
		// close logChannel
		close(p.logChannel)
		return perror.PolvoPipelineError{
			Code:   perror.ErrSensorPanic,
			Origin: err,
			Msg:    fmt.Sprintf("error in sensor[%s] thread", p.sensorName),
		}
	}
	p.logger.PrintInfo("pipeline [%s]: sensor thread is closed", p.sensorName)
	// wait until all logs are exported in logChannel
	// This is useless live lock. Because in golang, channel can pop data when it is closed.
	// So, this is just for safety.
	for {
		if len(p.logChannel) == 0 {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	// close logChannel
	close(p.logChannel)
	return nil
}
