package compose_test

import (
	"os"
	"path/filepath"
	"polvo/compose"
	perror "polvo/error"
	"reflect"
	"testing"
)

var (
	pwd string
)

func TestMain(m *testing.M) {
	var err error
	// setup
	pwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	code := m.Run()
	// teardown
	os.Exit(code)
}

func TestComposeFileOpen(t *testing.T) {
	composr, err := compose.NewComposeFile(filepath.Join(pwd, "testdata", "compose_true.yml"))
	if err != nil {
		t.Errorf("error while create composer %v", err)
		return
	}
	sensor := composr.GetSensorCompose("sensor1")
	if sensor == nil {
		t.Errorf("sensor1 not found")
		return
	}
	exporter := composr.GetExporterCompose("exporter1")
	if exporter == nil {
		t.Errorf("exporter1 not found")
		return
	}
	service := composr.GetServiceCompose()
	if service == nil {
		t.Errorf("service not found")
		return
	}
	t.Logf("sensor: %v", composr.String())
}

func TestComposeFileFailedInSensorWrongExecPath(t *testing.T) {
	_, err := compose.NewComposeFile(filepath.Join(pwd, "testdata", "compose_sensor_wrong_execute_path.yml"))
	if err == nil {
		t.Errorf("error should be created in composer %v", err)
		return
	}
	if reflect.TypeOf(err) == reflect.TypeOf(perror.PolvoComposeError{}) {
		t.Logf("errorType is %v\n %v", reflect.TypeOf(err), err)
		return
	} else {
		t.Errorf("errorType is %v. but error should be %v", reflect.TypeOf(err), reflect.TypeOf(perror.PolvoComposeError{}))
		return
	}
}

func TestComposeFileFailedInSensorNotExecutable(t *testing.T) {
	_, err := compose.NewComposeFile(filepath.Join(pwd, "testdata", "compose_sensor_not_executable.yml"))
	if err == nil {
		t.Errorf("error should be created in composer %v", err)
		return
	}
	if reflect.TypeOf(err) == reflect.TypeOf(perror.PolvoComposeError{}) {
		t.Logf("errorType is %v\n %v", reflect.TypeOf(err), err)
		return
	} else {
		t.Errorf("errorType is %v. but error should be %v", reflect.TypeOf(err), reflect.TypeOf(perror.PolvoComposeError{}))
		return
	}
}

func TestComposeFileFailedInExporterWrongDestination(t *testing.T) {
	_, err := compose.NewComposeFile(filepath.Join(pwd, "testdata", "compose_exporter_wrong_destination.yml"))
	if err == nil {
		t.Errorf("error should be created in composer %v", err)
		return
	}
	if reflect.TypeOf(err) == reflect.TypeOf(perror.PolvoComposeError{}) {
		t.Logf("errorType is %v\n %v", reflect.TypeOf(err), err)
		return
	} else {
		t.Errorf("errorType is %v. but error should be %v", reflect.TypeOf(err), reflect.TypeOf(perror.PolvoComposeError{}))
		return
	}
}

func TestComposeFileFailedInExporterWrongTimeout(t *testing.T) {
	_, err := compose.NewComposeFile(filepath.Join(pwd, "testdata", "compose_exporter_wrong_timeout.yml"))
	if err == nil {
		t.Errorf("error should be created in composer %v", err)
		return
	}
	if reflect.TypeOf(err) == reflect.TypeOf(perror.PolvoComposeError{}) {
		t.Logf("errorType is %v\n %v", reflect.TypeOf(err), err)
		return
	} else {
		t.Errorf("errorType is %v. but error should be %v", reflect.TypeOf(err), reflect.TypeOf(perror.PolvoComposeError{}))
		return
	}
}

func TestComposeFileFailedInServiceWrongSensorInPipeline(t *testing.T) {
	_, err := compose.NewComposeFile(filepath.Join(pwd, "testdata", "compose_service_wrong_sensor_in_pipeline.yml"))
	if err == nil {
		t.Errorf("error should be created in composer %v", err)
		return
	}
	if reflect.TypeOf(err) == reflect.TypeOf(perror.PolvoComposeError{}) {
		t.Logf("errorType is %v\n %v", reflect.TypeOf(err), err)
		return
	} else {
		t.Errorf("errorType is %v. but error should be %v", reflect.TypeOf(err), reflect.TypeOf(perror.PolvoComposeError{}))
		return
	}
}

func TestComposeFileFailedInServiceWrongExporterInPipeline(t *testing.T) {
	_, err := compose.NewComposeFile(filepath.Join(pwd, "testdata", "compose_service_wrong_exporter_in_pipeline.yml"))
	if err == nil {
		t.Errorf("error should be created in composer %v", err)
		return
	}
	if reflect.TypeOf(err) == reflect.TypeOf(perror.PolvoComposeError{}) {
		t.Logf("errorType is %v\n %v", reflect.TypeOf(err), err)
		return
	} else {
		t.Errorf("errorType is %v. but error should be %v", reflect.TypeOf(err), reflect.TypeOf(perror.PolvoComposeError{}))
		return
	}
}
