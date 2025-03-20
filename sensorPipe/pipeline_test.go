package sensorPipe_test

import (
	"fmt"
	"os"
	"path/filepath"
	plogger "polvo/logger"
	"polvo/sensorPipe"
	"strings"
	"testing"
)

type Samplelog struct {
	Event   string
	Pid     string
	Content string
}

func Wrap(log string) (*Samplelog, error) {
	chunk := strings.Split(log, " ")
	return &Samplelog{
		Event:   chunk[0],
		Pid:     chunk[1],
		Content: chunk[2],
	}, nil
}

var (
	pwd     string
	loger   plogger.PolvoLogger
	logpath string
)

func TestMain(m *testing.M) {
	var (
		err error
	)
	// setup
	pwd, err = os.Getwd()
	if err != nil {
		fmt.Printf("error while get working directory %v", err)
		os.Exit(1)
	}
	logpath = filepath.Join(pwd, "testdata")
	loger = plogger.NewLogger(logpath)
	// run tests
	code := m.Run()
	// teardown
	loger.Close()
	os.Remove(filepath.Join(logpath, "service.log"))
	os.Exit(code)
}

func TestPipelineLoop100(t *testing.T) {
	pipe, err := sensorPipe.NewPipe("sensor", 0, loger, Wrap)
	if err != nil {
		fmt.Printf("error while create pipeline %v", err)
		os.Exit(1)
	}
	pipe.Start(filepath.Join(pwd, "testdata", "dummy.sh"))
	for i := 0; i < 100; i++ {
		<-pipe.LogChannel()
		// t.Logf("log: %v", *log)
	}
	err = pipe.Stop()
	if err != nil {
		t.Errorf("Error while stopping pipeline %v", err)
	}
}

func TestPipelineWaitReturnsError(t *testing.T) {
	pipe, err := sensorPipe.NewPipe("sensor", 0, loger, Wrap)
	if err != nil {
		fmt.Printf("error while create pipeline %v", err)
		os.Exit(1)
	}
	err = pipe.Start(filepath.Join(pwd, "testdata", "dummy_fail.sh"))
	if err != nil {
		t.Errorf("Error while starting pipeline %v", err)
		return
	}
	defer pipe.Stop()

	go func() {
		for _ = range pipe.LogChannel() {
			// t.Logf("log: %v", *log)
		}
	}()

	err = pipe.Wait()
	if err == nil {
		t.Errorf("Error should raised in waiting pipeline %v", err)
	}
	t.Logf("Error: %v", err)
}

func TestPipelineStartDuplicated(t *testing.T) {
	pipe, err := sensorPipe.NewPipe("sensor", 0, loger, Wrap)
	if err != nil {
		fmt.Printf("error while create pipeline %v", err)
		os.Exit(1)
	}
	err = pipe.Start(filepath.Join(pwd, "testdata", "dummy.sh"))
	if err != nil {
		t.Errorf("Error while starting pipeline %v", err)
		return
	}
	defer pipe.Stop()
	err = pipe.Start(filepath.Join(pwd, "testdata", "dummy.sh"))
	if err == nil {
		t.Errorf("Error should have been raised")
	}
	t.Logf("Error: %v", err)
}

func TestPipelineStopBeforeStart(t *testing.T) {
	pipe, err := sensorPipe.NewPipe("sensor", 0, loger, Wrap)
	if err != nil {
		fmt.Printf("error while create pipeline %v", err)
		os.Exit(1)
	}
	err = pipe.Stop()
	if err == nil {
		t.Errorf("Error should have been raised")
	}
	t.Logf("Error: %v", err)
}
