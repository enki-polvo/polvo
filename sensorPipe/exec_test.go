package sensorPipe_test

import (
	"os"
	perror "polvo/error"
	"polvo/sensorPipe"
	"reflect"
	"testing"
)

func TestPromise(t *testing.T) {
	promise, err := sensorPipe.Run(os.Stdin, os.Stdout, "echo", "hello")
	if err != nil {
		t.Errorf("Error while executing promise %v", err)
		return
	}
	_, err = promise.Wait()
	if err != nil {
		t.Errorf("Error while executing promise %v", err)
	}
}

func TestPromiseError(t *testing.T) {
	_, err := sensorPipe.Run(os.Stdin, os.Stdout, "asdf", "ghjk")
	if err != nil && reflect.TypeOf(err) == reflect.TypeOf(perror.PolvoGeneralError{}) {
		t.Logf("Error while executing promise %v", err)
		return
	}
	t.Errorf("Error should have been raised")
}

func TestPromiseWaitAfterCancel(t *testing.T) {
	promise, err := sensorPipe.Run(os.Stdin, os.Stdout, "echo", "hello")
	if err != nil {
		t.Errorf("Error while executing promise %v", err)
		return
	}
	err = promise.Cancel()
	if err != nil {
		t.Errorf("Error while executing promise %v", err)
		return
	}
	_, err = promise.Wait()
	if err == nil {
		t.Errorf("Error should have been raised")
		return
	}
	t.Logf("Error: %v", err)
}

// (Deprecated)
// Because of sensors will not use stdin, this test is deprecated.
//
// func TestPromiseInput(t *testing.T) {
// 	promise, err := sensorPipe.Run(os.Stdin, os.Stdout, "man", "watch")
// 	if err != nil {
// 		t.Errorf("Error while executing promise %v", err)
// 		return
// 	}
// 	err = promise.Wait()
// 	if err != nil {
// 		t.Errorf("Error while executing promise %v", err)
// 	}
// }

/**********************************************************************************
* Deprecated because struct is now private and cannot be accessed without Constructor
**********************************************************************************/

// func TestWaitBeforePromise(t *testing.T) {
// 	promise := new(sensorPipe.Promise)
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
	promise, err := sensorPipe.Run(os.Stdin, os.Stdout, "echo", "hello")
	if err != nil {
		t.Errorf("Error while executing promise %v", err)
		return
	}
	_, err = promise.Wait()
	if err != nil {
		t.Errorf("Error while executing promise %v", err)
		return
	}
	_, err = promise.Wait()
	if err == nil {
		t.Errorf("Error should have been raised when calling Wait() twice")
	}
	t.Logf("Error: %v", err)
}
