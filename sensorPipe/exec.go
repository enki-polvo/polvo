package sensorPipe

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	perror "polvo/error"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type Promise interface {
	Wait() error
	Cancel() error
}

type promise struct {
	ctx     context.Context
	cancel  context.CancelFunc
	cmd     exec.Cmd
	eg      *errgroup.Group
	waitCnt int32
}

func Run(inStream *os.File, outStream *os.File, arg0 string, args ...string) Promise {
	prom := new(promise)
	prom.ctx, prom.cancel = context.WithCancel(context.Background())
	// set conditional variable to -1
	atomic.StoreInt32(&prom.waitCnt, -1)
	// set the commandline
	prom.cmd = *exec.CommandContext(prom.ctx, arg0, args...)
	prom.cmd.Stdin = inStream
	prom.cmd.Stdout = outStream
	prom.cmd.Stderr = outStream

	// set the error for run commandline goroutine
	prom.eg, _ = errgroup.WithContext(prom.ctx)
	prom.eg.Go(func() error {
		return prom.cmd.Run()
	})
	// set the conditional variable to 0
	atomic.StoreInt32(&prom.waitCnt, 0)
	return prom
}

func (p *promise) Wait() (err error) {
	// if errgroup is not initialized, return error
	if p.eg == nil {
		return perror.PolvoGeneralError{
			Code:   perror.ErrInvalidOperation,
			Origin: fmt.Errorf("promise is not initialized"),
			Msg:    "error while execute promise.Wait()",
		}
	}
	// if conditional variable is not set to 0, return error
	if atomic.LoadInt32(&p.waitCnt) != 0 {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Msg:    fmt.Sprintf("error while execute %s %v promise.Wait()", p.cmd.Path, p.cmd.Args),
			Origin: fmt.Errorf("promise is not initialized [%d]", atomic.LoadInt32(&p.waitCnt)),
		}
	}
	// set the conditional variable to 1
	atomic.StoreInt32(&p.waitCnt, 1)
	// wait for the errgroup to finish
	err = p.eg.Wait()
	if err != nil {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Msg:    fmt.Sprintf("error while execute %s %v promise.Wait()", p.cmd.Path, p.cmd.Args),
			Origin: err,
		}
	}
	return nil
}

func (p *promise) Cancel() (err error) {
	if p.cancel == nil {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Msg:    "error while execute promise.Cancel()",
			Origin: fmt.Errorf("promise is not initialized"),
		}
	}
	if atomic.LoadInt32(&p.waitCnt) != 0 {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Msg:    fmt.Sprintf("error while execute %s %v promise.Cancel()", p.cmd.Path, p.cmd.Args),
			Origin: fmt.Errorf("promise is not initialized [%d]", atomic.LoadInt32(&p.waitCnt)),
		}
	}
	// cancel the context
	p.cancel()
	return nil
}
