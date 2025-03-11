package compose

import (
	"fmt"
	"net"
	"os"
	perror "polvo/error"
	"runtime"
	"strconv"
	"strings"

	_ "github.com/alecthomas/participle"

	"gopkg.in/yaml.v3"
)

// # ComposeFile
//
// reads polvo-compose.yml file and constructs a Compose struct.
//
// # Compose struct
//
// - sensors: sensor contains sensor executable path, parameter, runAsRoot, and eventsHeader. eventsHeader is a map that contains event name and format of sensor event logs.
//
// - exporters: exporter contains exporter destination and timeout.
//
// - service: service contains machine, os, arch, description, and pipeline. pipeline defines sensor & exporter composition.
type ComposeFile interface {
	GetSensorCompose(name string) *Sensor
	GetExporterCompose(name string) *Exporter
	GetServiceCompose() *Service
	String() string
}

type composeFile struct {
	compose *Compose
}

// Getter for Sensor
func (c *composeFile) GetSensorCompose(name string) *Sensor {
	var (
		ret Sensor
		ok  bool
	)

	if ret, ok = c.compose.sensors[name]; !ok {
		return nil
	}
	return &ret
}

// Getter for Exporter
func (c *composeFile) GetExporterCompose(name string) *Exporter {
	var (
		ret Exporter
		ok  bool
	)

	if ret, ok = c.compose.exporters[name]; !ok {
		return nil
	}
	return &ret
}

// Getter for Service
func (c *composeFile) GetServiceCompose() *Service {
	return c.compose.service
}

// Stringer for ComposeFile
func (c *composeFile) String() string {
	sensorStr := "compose: \n-------------Sensor --------------\n"
	for sensorName, sensor := range c.compose.sensors {
		sensorStr += fmt.Sprintf("%s:\n\texec_path: %v\n\tparam: %s\n\trun_as_root: %v\n\tevents_header: %v\n", sensorName,
			sensor.execPath,
			sensor.param,
			sensor.runAsRoot,
			sensor.eventsHeader)
	}
	exporterStr := "-------------Exporter --------------\n"
	for exporterName, exporter := range c.compose.exporters {
		exporterStr += fmt.Sprintf("%s:\n\tdestination: %v\n\ttimeout: %v\n", exporterName, exporter.destination, exporter.timeout)
	}
	serviceStr := fmt.Sprintf("-------------Service --------------\n\tmachine: %v\n\tos: %v\n\tarch: %v\n\tgroup: %v\n\tdescription: %v",
		c.compose.service.machine,
		c.compose.service.os,
		c.compose.service.arch,
		c.compose.service.group,
		c.compose.service.description)
	for pipeName, pipeline := range c.compose.service.pipeline {
		serviceStr += fmt.Sprintf("\n\t%s:\n\t\t%s: %v", pipeName, "sensors", func() []string {
			ret := make([]string, 0)
			for _, sensor := range pipeline.sensors {
				ret = append(ret, sensor.Name)
			}
			return ret
		}())
		serviceStr += fmt.Sprintf("\n\t\t%s: %v", "exporters", func() []string {
			ret := make([]string, 0)
			for _, exporter := range pipeline.exporters {
				ret = append(ret, exporter.Name)
			}
			return ret
		}())
	}
	return sensorStr + exporterStr + serviceStr
}

func NewComposeFile(composeFilePath string) (ComposeFile, error) {
	var (
		wrapper ComposeWrapper
	)

	newComp := new(composeFile)

	// read config file
	file, err := os.ReadFile(composeFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, perror.PolvoGeneralError{
				Code:   perror.InvalidArgumentError,
				Origin: err,
				Msg:    "error while Construct new composeFile",
			}
		}
		return nil, perror.PolvoGeneralError{
			Code:   perror.SystemError,
			Origin: err,
			Msg:    "error while Construct new composeFile",
		}
	}

	// parse config file
	err = yaml.Unmarshal(file, &wrapper)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composeFile",
		}
	}

	newComp.compose = new(Compose)
	// get sensor from wrapper
	newComp.compose.sensors, err = newComp.getSensor(wrapper.Sensors)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composeFile",
		}
	}
	// get exporter from wrapper
	newComp.compose.exporters, err = newComp.getExporters(wrapper.Exporters)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composeFile",
		}
	}
	// get service from wrapper
	newComp.compose.service, err = newComp.getService(wrapper.Service)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composeFile",
		}
	}

	return newComp, nil
}

// getSensor constructs Sensor struct from SensorWrapper & verifies the sensor compose file.
func (c *composeFile) getSensor(wrapperMap map[string]SensorWrapper) (map[string]Sensor, error) {
	var (
		sensorMap    map[string]Sensor
		execFileInfo os.FileInfo
		err          error
	)

	sensorMap = make(map[string]Sensor)

	for sensorName, sensorObj := range wrapperMap {
		// null check
		if sensorObj.ExecPath == "" {
			return nil, perror.PolvoComposeError{
				Code:   perror.SensorNotFoundError,
				Msg:    "error in getSensor.",
				Origin: fmt.Errorf("exec_path is empty"),
			}
		}
		// check execPath is exist.
		if execFileInfo, err = os.Stat(sensorObj.ExecPath); err != nil {
			if os.IsNotExist(err) {
				return nil, perror.PolvoComposeError{
					Code:   perror.SensorNotFoundError,
					Msg:    "error in getSensor.",
					Origin: err,
				}
			}
			return nil, perror.PolvoGeneralError{
				Code:   perror.SystemError,
				Msg:    "error in getSensor.",
				Origin: err,
			}
		}
		// check exePath is executable
		if execFileInfo.Mode()&0111 == 0 {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidSensorError,
				Msg:    "error in getSensor.",
				Origin: fmt.Errorf("exec_path is not executable"),
			}
		}
		// check events header exists
		if len(sensorObj.EventsHeader) <= 0 {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidSensorError,
				Msg:    "error in getSensor.",
				Origin: fmt.Errorf("events_header is empty"),
			}
		}
		// add sensor
		sensorMap[sensorName] = Sensor{
			Name:         sensorName,
			execPath:     sensorObj.ExecPath,
			param:        sensorObj.Param,
			runAsRoot:    sensorObj.RunAsRoot,
			eventsHeader: sensorObj.EventsHeader,
		}
	}
	return sensorMap, nil
}

func isValidIPPort(addr string) bool {
	// replace localhost to 127.0.0.1
	addr = strings.Replace(addr, "localhost", "127.0.0.1", -1)
	// IP check
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	// check IP
	if net.ParseIP(host) == nil {
		return false
	}

	// check port (0~65535)
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 0 || portNum > 65535 {
		return false
	}

	return true
}

// getExporter constructs Exporter struct from ExporterWrapper & verifies the exporter compose file.
func (c *composeFile) getExporters(wrapperMap map[string]ExporterWrapper) (map[string]Exporter, error) {
	exporterMap := make(map[string]Exporter)

	for exporterName, exporterObj := range wrapperMap {
		// null check
		if exporterObj.Destination == "" {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidExporterError,
				Msg:    "error in getExporter.",
				Origin: fmt.Errorf("destination is empty"),
			}
		}
		// check destination is ip url format
		if !isValidIPPort(exporterObj.Destination) {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidExporterError,
				Msg:    "error in getExporter.",
				Origin: fmt.Errorf("destination is not valid url"),
			}

		}
		// check timeout is valid
		if exporterObj.Timeout <= 0 {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidExporterError,
				Msg:    "error in getExporter.",
				Origin: fmt.Errorf("timeout is invalid"),
			}
		}
		// add exporter to map
		exporterMap[exporterName] = Exporter{
			Name:        exporterName,
			destination: exporterObj.Destination,
			timeout:     exporterObj.Timeout,
		}
	}
	return exporterMap, nil
}

// getService constructs Service struct from ServiceWrapper & verifies the service compose file.
func (c *composeFile) getService(wrapper ServiceWrapper) (*Service, error) {
	var (
		service Service
	)

	// null check
	if wrapper.Description == "" {
		return nil, perror.PolvoComposeError{
			Code:   perror.InvalidServiceError,
			Msg:    "error in getService.",
			Origin: fmt.Errorf("description is empty"),
		}
	}
	if wrapper.Group == "" {
		return nil, perror.PolvoComposeError{
			Code:   perror.InvalidServiceError,
			Msg:    "error in getService.",
			Origin: fmt.Errorf("group is empty"),
		}
	}
	if len(wrapper.Pipelines) <= 0 {
		return nil, perror.PolvoComposeError{
			Code:   perror.InvalidServiceError,
			Msg:    "error in getService.",
			Origin: fmt.Errorf("sensors is empty"),
		}
	}
	// check pipeline is valid
	pipelines := make(map[string]Pipeline)
	for pipeName, pipeline := range wrapper.Pipelines {
		sensors := make([]*Sensor, 0)
		exporters := make([]*Exporter, 0)
		// null check
		if len(pipeline.Sensors) <= 0 {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidServiceError,
				Msg:    "error in getService.",
				Origin: fmt.Errorf("%s's sensors is empty", pipeName),
			}
		}
		if len(pipeline.Exporters) <= 0 {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidServiceError,
				Msg:    "error in getService.",
				Origin: fmt.Errorf("%s's exporters is empty", pipeName),
			}
		}
		// check pipeline sensors & exporters are valid
		for _, sensorName := range pipeline.Sensors {
			sensor, ok := c.compose.sensors[sensorName]
			if !ok {
				return nil, perror.PolvoComposeError{
					Code:   perror.InvalidServiceError,
					Msg:    "error in getService.",
					Origin: fmt.Errorf("%s's sensor %s is not defined", pipeName, sensorName),
				}
			}
			sensors = append(sensors, &sensor)
		}
		for _, exporterName := range pipeline.Exporters {
			exporter, ok := c.compose.exporters[exporterName]
			if !ok {
				return nil, perror.PolvoComposeError{
					Code:   perror.InvalidServiceError,
					Msg:    "error in getService.",
					Origin: fmt.Errorf("%s's exporter %s is not defined", pipeName, exporterName),
				}
			}
			exporters = append(exporters, &exporter)
		}
		// TODO: read valid exporter & sensor from config file
		// add pipeline to map
		pipelines[pipeName] = Pipeline{
			sensors:   sensors,
			exporters: exporters,
		}
	}
	// get machine from os
	serviceMachine, err := os.Hostname()
	if err != nil {
		return nil, perror.PolvoGeneralError{
			Code:   perror.SystemError,
			Msg:    "error in getService.",
			Origin: err,
		}
	}
	// get os version from runtime
	serviceOS := runtime.GOOS
	serviceArch := runtime.GOARCH

	// add service
	service = Service{
		machine:     serviceMachine,
		os:          serviceOS,
		arch:        serviceArch,
		group:       wrapper.Group,
		description: wrapper.Description,
		pipeline:    pipelines,
	}

	return &service, nil
}
