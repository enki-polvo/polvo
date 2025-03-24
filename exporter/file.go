package exporter

import (
	"context"
	"fmt"
	"os"
	"polvo/compose"
	perror "polvo/error"
	plogger "polvo/logger"
	"sync"
	"sync/atomic"
	"time"
)

type fileExporter[log any] struct {
	// dependency injection
	logger plogger.PolvoLogger
	info   *compose.ExporterInfo
	// fields
	exporterName string
	// stream
	logChannel  chan *log
	writeStream *os.File
	// thread control
	ctx      context.Context
	cancel   context.CancelFunc
	waitGrp  sync.WaitGroup
	wrapFunc func(*log) ([]byte, error)
	// conditional variable
	isClosed  int32
	isStarted int32
}

/************************************************************
* Getter & Setter
************************************************************/

func (fe *fileExporter[log]) Name() string {
	return fe.exporterName
}

func (fe *fileExporter[log]) LogChannel() chan<- *log {
	return fe.logChannel
}

/************************************************************
* Methods
************************************************************/

func (fe *fileExporter[log]) Start() {
	go fe.exportThread()
	// set conditional variable to 1
	atomic.StoreInt32(&fe.isStarted, 1)
}

func (fe *fileExporter[log]) Stop() error {
	// prevent double close
	if atomic.LoadInt32(&fe.isClosed) > 0 {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is already closed"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Close()", fe.exporterName),
		}
	}
	// prevent call Stop() before Start()
	if atomic.LoadInt32(&fe.isStarted) <= 0 {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is not started"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Close()", fe.exporterName),
		}
	}
	// wait until all logs are exported
	for {
		if len(fe.logChannel) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	// cancel context
	fe.cancel()
	// close log channel
	close(fe.logChannel)
	// close write stream
	err := fe.writeStream.Close()
	if err != nil {
		return perror.PolvoGeneralError{
			Code:   perror.SystemError,
			Origin: err,
			Msg:    fmt.Sprintf("error while close Destination file[%s]", fe.info.Destination),
		}
	}
	// set conditional variable to 1
	atomic.StoreInt32(&fe.isClosed, 1)
	return nil
}

func (fe *fileExporter[log]) Wait() {
	fe.waitGrp.Wait()
}

/************************************************************
* Constructor
************************************************************/

func NewFileExporter[log any](name string,
	maxSize uint,
	wrapFunc func(*log) ([]byte, error),
	logger plogger.PolvoLogger,
	info *compose.ExporterInfo) (Exporter[log], error) {

	newFE := new(fileExporter[log])
	// dependency injection
	newFE.logger = logger
	newFE.info = info
	// context
	newFE.ctx, newFE.cancel = context.WithCancel(context.Background())
	// set fields
	newFE.exporterName = name
	newFE.wrapFunc = wrapFunc
	// open stream
	err := newFE.openStream(maxSize)
	if err != nil {
		return nil, perror.PolvoPipelineError{
			Code:   perror.ErrExporterCreate,
			Origin: err,
			Msg:    fmt.Sprintf("error while construct new exporter[%s]", name),
		}
	}
	// set conditional variable to 0
	atomic.StoreInt32(&newFE.isClosed, 0)
	atomic.StoreInt32(&newFE.isStarted, 0)
	return newFE, nil
}

func (fe *fileExporter[log]) openStream(maxSize uint) error {
	var err error

	// check if pipeline is already closed
	if atomic.LoadInt32(&fe.isClosed) > 0 || fe.logChannel != nil {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: fmt.Errorf("pipeline is already opened"),
			Msg:    fmt.Sprintf("error while execute pipeline[%s].Open()", fe.exporterName),
		}
	}
	// init pipeline
	if maxSize == 0 {
		fe.logChannel = make(chan *log)
	} else {
		fe.logChannel = make(chan *log, maxSize)
	}
	// open target file
	fe.writeStream, err = os.OpenFile(fe.info.Destination, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Origin: err,
			Msg:    fmt.Sprintf("error while open Destination file[%s]", fe.info.Destination),
		}
	}
	return nil
}

func (fe *fileExporter[log]) exportThread() {
	var (
		logWrapper *log
		out        []byte
		err        error
	)
	// wait group
	fe.waitGrp.Add(1)
	defer fe.waitGrp.Done()

	for {
		select {
		case <-fe.ctx.Done():
			return
		case logWrapper = <-fe.logChannel:
			// wrap log
			out, err = fe.wrapFunc(logWrapper)
			if err != nil {
				fe.logger.PrintError("error while wrap log %s", err.Error())
				continue
			}
			// append newline
			out = append(out, '\n')
			// export log
			_, err = fe.writeStream.Write(out)
			if err != nil {
				fe.logger.PrintError("error while write log %s", err.Error())
				// this is critical error and unrecoverable. so panic
				panic(err)
			}
		}
	}
}
