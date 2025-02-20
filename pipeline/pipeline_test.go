package pipeline_test

import (
	"fmt"
	"os"
	"path/filepath"
	plogger "polvo/logger"
	"polvo/pipeline"
	"strings"
	"testing"
)

type logWrapper struct {
	Event   string
	Pid     string
	Content string
}

func Wrap(log string) (logWrapper, error) {
	chunk := strings.Split(log, " ")
	return logWrapper{
		Event:   chunk[0],
		Pid:     chunk[1],
		Content: chunk[2],
	}, nil
}

var (
	pwd     string
	loger   plogger.PolvoLogger
	pipe    pipeline.Pipeline[logWrapper]
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
	pipe, err = pipeline.NewPipeline("sensor", 0, loger, Wrap)
	if err != nil {
		fmt.Printf("error while create pipeline %v", err)
		os.Exit(1)
	}
	// run tests
	code := m.Run()
	// teardown
	loger.Close()
	os.Remove(filepath.Join(logpath, "service.log"))
	os.Exit(code)
}

func TestPipeline(t *testing.T) {
	pipe.Start(filepath.Join(pwd, "testdata", "dummy.sh"))
	for i := 0; i < 100; i++ {
		log := <-pipe.Pipeline()
		t.Logf("log: %v", log)
	}
	pipe.Stop()
}
