package service_test

import (
	"encoding/json"
	"polvo/service"
	"testing"
)

func TestCreateLogWrapperFromSensorInfo(t *testing.T) {
	// create log wrapper from sensor info
	// createLogWrapperFromSensorInfo("sensor", []string{"field1", "field2"})
	jsonStr := `{"eventname": "bashReadline", "source": "eBPF", "timestamp": "2025-03-11T15:29:34+09:00", "log": "A user has entered a command in the bash shell", "metadata": {"Commandline":"echo hello world","PID":191998,"UID":1000,"Username":"shhong"}}`
	// out := service.CreateDynamicLogWrapperFromSensorInfo([]string{"Commandline", "PID", "UID", "Username"})
	common := service.CommonHeaderWrapper{}
	// common.MetaData = reflect.New(out).Interface()
	err := json.Unmarshal([]byte(jsonStr), &common)
	if err != nil {
		panic(err)
	}
	t.Logf("result: %v\n", common.MetaData.(map[string]interface{})["Commandline"])
	result, err := json.MarshalIndent(common, "", "  ")
	if err != nil {
		panic(err)
	}
	t.Logf("result: %v", string(result))
}
