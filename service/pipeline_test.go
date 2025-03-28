package service_test

import (
	"fmt"
	"os"
	"path/filepath"
	"polvo/compose"
	plogger "polvo/logger"
	"polvo/service"
	"polvo/service/filter"
	"testing"
	"time"
)

const sampleFilter = `
version: 1.0
deny:
  "filter_test":
    "condition":
      "eventname|contains":
        - "bashReadline"
    "exception":
      "eventname|contains":
        - "process"
      "Commandline|contains":
        - "ls"
        - "cat"
   
`

var (
	pwd      string
	loger    plogger.PolvoLogger
	logpath  string
	composer compose.ComposeFile
	filterOp filter.FilterOperator
)

func TestMain(m *testing.M) {
	var err error
	// setup
	pwd, err = os.Getwd()
	if err != nil {
		fmt.Printf("error while get working directory %v", err)
		os.Exit(1)
	}
	logpath = filepath.Join(pwd, "testdata")
	loger = plogger.NewLogger(logpath)
	// init compose
	composer, err = compose.NewComposeFile(filepath.Join(pwd, "testdata", "compose.yml"))
	if err != nil {
		os.Remove(filepath.Join(logpath, "service.log"))
		os.Remove(filepath.Join(logpath, "output.log"))
		os.Remove(filepath.Join(logpath, "output2.log"))
		panic(err)
	}
	// print compose
	fmt.Printf("%v\n", composer.String())
	// filter operator
	filterOp, err = filter.NewFilterOperator([]byte(sampleFilter))
	if err != nil {
		loger.Close()
		panic(err)
	}
	// run tests
	code := m.Run()
	// teardown
	loger.Close()
	os.Remove(filepath.Join(logpath, "service.log"))
	os.Remove(filepath.Join(logpath, "output.log"))
	os.Remove(filepath.Join(logpath, "output2.log"))
	os.Exit(code)
}

func TestService(t *testing.T) {
	// create service
	serv, err := service.NewService(composer.GetCompose(), loger, filterOp)
	if err != nil {
		t.Errorf("error while create service %v", err)
		return
	}
	// start service
	serv.Start()
	// wait service 10 seconds
	time.Sleep(10 * time.Second)
	// stop service
	err = serv.Stop()
	if err != nil {
		t.Errorf("error while stop service %v", err)
		return
	}
}
