package sensorPipe

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	perror "polvo/error"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type Promise interface {
	Wait() error
	Cancel() error
}

type promise struct {
	ctx      context.Context
	cancel   context.CancelFunc
	cmd      exec.Cmd
	eg       *errgroup.Group
	waitOnce sync.Once
	waitCnt  int32
}

func Run(inStream *os.File, outStream *os.File, arg0 string, args ...string) (Promise, error) {
	prom := new(promise)
	// set the context timeout is 5 seconds
	prom.ctx, prom.cancel = context.WithTimeout(context.Background(), 5*time.Second)
	// set conditional variable to -1
	atomic.StoreInt32(&prom.waitCnt, -1)
	// set the commandline
	prom.cmd = *exec.CommandContext(prom.ctx, arg0, args...)
	prom.cmd.Stdin = inStream
	prom.cmd.Stdout = outStream
	// prom.cmd.Stderr = outStream

	// init once object for wait
	prom.waitOnce = sync.Once{}

	// generate process group
	prom.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// set the error for run commandline goroutine
	err := prom.cmd.Start()
	if err != nil {
		return nil, perror.PolvoGeneralError{
			Code:   perror.SystemError,
			Origin: err,
			Msg:    fmt.Sprintf("error while execute %s %v promise.Start()", prom.cmd.Path, prom.cmd.Args),
		}
	}
	// set the conditional variable to 0
	atomic.StoreInt32(&prom.waitCnt, 0)
	return prom, nil
}

func (p *promise) Wait() (err error) {
	// if conditional variable is not set to 0, return error
	val := atomic.LoadInt32(&p.waitCnt)
	if val < 0 || val > 0 {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Msg:    fmt.Sprintf("error while execute %s %v promise.Wait()", p.cmd.Path, p.cmd.Args),
			Origin: fmt.Errorf("wait is called while promise is not initialized [%d]", atomic.LoadInt32(&p.waitCnt)),
		}
	}
	// set the conditional variable to 1
	atomic.StoreInt32(&p.waitCnt, 1)
	p.cmd.Process.Wait()
	// Do not check error here, because it will be handled in SIGKILL signal
	//
	// _, err = p.cmd.Process.Wait()
	// if err != nil {
	// 	return perror.PolvoGeneralError{
	// 		Code:   perror.InvalidOperationError,
	// 		Msg:    fmt.Sprintf("error while execute %s %v promise.Wait()", p.cmd.Path, p.cmd.Args),
	// 		Origin: err,
	// 	}
	// }
	return nil
}

func (p *promise) Cancel() (err error) {
	if p.cancel == nil {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Msg:    "error while execute promise.Cancel()",
			Origin: fmt.Errorf("cancel is called while promise is not initialized"),
		}
	}
	if atomic.LoadInt32(&p.waitCnt) < 0 {
		return perror.PolvoGeneralError{
			Code:   perror.InvalidOperationError,
			Msg:    fmt.Sprintf("error while execute %s %v promise.Cancel()", p.cmd.Path, p.cmd.Args),
			Origin: fmt.Errorf("cancel is called while promise is not initialized [%d]", atomic.LoadInt32(&p.waitCnt)),
		}
	}
	p.eg, _ = errgroup.WithContext(p.ctx)
	p.eg.Go(func() error {

		// kill proc group
		//
		// (DEPRECATED) kill process group
		//
		// pgid, err := syscall.Getpgid(p.cmd.Process.Pid)
		// if err == nil {
		// 	fmt.Fprintf(os.Stderr, "Killing process group %d\n", pgid)
		// 	syscall.Kill(-pgid, syscall.SIGKILL)
		// }
		//
		// kill process by SIGKILL signal
		p.cmd.Process.Signal(syscall.SIGKILL)
		// wait to prevent zombie process
		err = p.cmd.Wait()
		// release the process resource
		return p.cmd.Process.Release()
	})
	// Do not wait killing goroutine here, because it will be handled in timeout context.
	//
	// // wait for the errgroup to finish killing the process
	// err = p.eg.Wait()
	// if err != nil {
	// 	p.cancel()
	// 	return perror.PolvoGeneralError{
	// 		Code:   perror.InvalidOperationError,
	// 		Msg:    fmt.Sprintf("error while execute %s %v promise.Cancel()", p.cmd.Path, p.cmd.Args),
	// 		Origin: err,
	// 	}
	// }
	return nil
}
