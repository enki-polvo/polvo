package sensorPipe_test

import (
	"os"
	"polvo/sensorPipe"
	"testing"
	"time"
)

func TestPromise(t *testing.T) {
	promise := sensorPipe.Run(os.Stdin, os.Stdout, "echo", "hello")
	err := promise.Wait()
	if err != nil {
		t.Errorf("Error while executing promise %v", err)
	}
}

func TestPromiseError(t *testing.T) {
	promise := sensorPipe.Run(os.Stdin, os.Stdout, "asdf", "ghjk")
	err := promise.Wait()
	if err == nil {
		t.Errorf("Error should have been raised")
	}
}

func TestPromiseInput(t *testing.T) {
	promise := sensorPipe.Run(os.Stdin, os.Stdout, "man", "watch")
	err := promise.Wait()
	if err != nil {
		t.Errorf("Error while executing promise %v", err)
	}
}

func TestPromiseWithoutWait(t *testing.T) {
	start := time.Now()

	promise := sensorPipe.Run(os.Stdin, os.Stdout, "watch", "-n", "1", "echo", "hello")
	defer promise.Cancel()

	elapsed := time.Since(start)
	if elapsed > 1*time.Second {
		t.Errorf("Promise should not have waited for the command to finish")
	}
}

/**********************************************************************************
* Deprecated because struct is now private and cannot be accessed without Constructor
**********************************************************************************/

// func TestWaitBeforePromise(t *testing.T) {
// 	promise := new(exec.Promise)
// 	err := promise.Wait()
// 	if err == nil {
// 		t.Errorf("Error should have been raised")
// 	}
// 	t.Logf("Error: %v", err)
// }

// func TestCancelBeforePromise(t *testing.T) {
// 	promise := new(exec.Promise)
// 	err := promise.Cancel()
// 	if err == nil {
// 		t.Errorf("Error should have been raised")
// 	}
// 	t.Logf("Error: %v", err)
// }

func TestWaitFuncCallDuplicated(t *testing.T) {
	promise := sensorPipe.Run(os.Stdin, os.Stdout, "echo", "hello")
	err := promise.Wait()
	if err != nil {
		t.Errorf("Error while executing promise %v", err)
	}
	err = promise.Wait()
	if err == nil {
		t.Errorf("Error should have been raised")
	}
	t.Logf("Error: %v", err)
}
