package compose

import (
	"fmt"
	"net/url"
	"os"
	perror "polvo/error"
	"runtime"

	_ "github.com/alecthomas/participle"

	"gopkg.in/yaml.v3"
	_ "gopkg.in/yaml.v3"
)

// # Composer
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
type Composer interface {
	GetSensorCompose(name string) *Sensor
	GetExporterCompose(name string) *Exporter
}

type composer struct {
	compose *Compose
}

// Getter for Sensor
func (c *composer) GetSensorCompose(name string) *Sensor {
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
func (c *composer) GetExporterCompose(name string) *Exporter {
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
func (c *composer) GetServiceCompose() *Service {
	return &c.compose.service
}

func NewComposer(composeFilePath string) (Composer, error) {
	var (
		wrapper ComposeWrapper
	)

	newComp := new(composer)

	// read config file
	file, err := os.ReadFile(composeFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, perror.PolvoGeneralError{
				Code:   perror.InvalidArgumentError,
				Origin: err,
				Msg:    "error while Construct new composer",
			}
		}
		return nil, perror.PolvoGeneralError{
			Code:   perror.SystemError,
			Origin: err,
			Msg:    "error while Construct new composer",
		}
	}

	// parse config file
	err = yaml.Unmarshal(file, &wrapper)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composer",
		}
	}

	newComp.compose = new(Compose)
	// get sensor from wrapper
	newComp.compose.sensors, err = newComp.getSensor(wrapper.Sensors)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composer",
		}
	}
	// get exporter from wrapper
	newComp.compose.exporters, err = newComp.getExporters(wrapper.Exporters)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composer",
		}
	}
	// get service from wrapper
	newComp.compose.service, err = newComp.getService(wrapper.Service)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composer",
		}
	}

	return newComp, nil
}

// getSensor constructs Sensor struct from SensorWrapper & verifies the sensor compose file.
func (c *composer) getSensor(wrapperMap map[string]SensorWrapper) (map[string]Sensor, error) {
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
			execPath:     sensorObj.ExecPath,
			param:        sensorObj.Param,
			runAsRoot:    sensorObj.RunAsRoot,
			eventsHeader: sensorObj.EventsHeader,
		}
	}
	return sensorMap, nil
}

// getExporter constructs Exporter struct from ExporterWrapper & verifies the exporter compose file.
func (c *composer) getExporters(wrapperMap map[string]ExporterWrapper) (map[string]Exporter, error) {
	var (
		exporterMap map[string]Exporter
		err         error
		dest        *url.URL
	)

	exporterMap = make(map[string]Exporter)

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
		if dest, err = url.ParseRequestURI(exporterObj.Destination); err != nil || dest.Host == "" {
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
			destination: dest,
			timeout:     exporterObj.Timeout,
		}
	}
	return exporterMap, nil
}

// getService constructs Service struct from ServiceWrapper & verifies the service compose file.
func (c *composer) getService(wrapper ServiceWrapper) (Service, error) {
	var (
		service Service
	)

	// null check
	if wrapper.Os == "" {
		return Service{}, perror.PolvoComposeError{
			Code:   perror.InvalidServiceError,
			Msg:    "error in getService.",
			Origin: fmt.Errorf("os is empty"),
		}
	}
	if wrapper.Arch == "" {
		return Service{}, perror.PolvoComposeError{
			Code:   perror.InvalidServiceError,
			Msg:    "error in getService.",
			Origin: fmt.Errorf("arch is empty"),
		}
	}
	if wrapper.Description == "" {
		return Service{}, perror.PolvoComposeError{
			Code:   perror.InvalidServiceError,
			Msg:    "error in getService.",
			Origin: fmt.Errorf("description is empty"),
		}
	}
	if wrapper.Group == "" {
		return Service{}, perror.PolvoComposeError{
			Code:   perror.InvalidServiceError,
			Msg:    "error in getService.",
			Origin: fmt.Errorf("group is empty"),
		}
	}
	if len(wrapper.Pipelines) <= 0 {
		return Service{}, perror.PolvoComposeError{
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
			return Service{}, perror.PolvoComposeError{
				Code:   perror.InvalidServiceError,
				Msg:    "error in getService.",
				Origin: fmt.Errorf("%s's sensors is empty", pipeName),
			}
		}
		if len(pipeline.Exporters) <= 0 {
			return Service{}, perror.PolvoComposeError{
				Code:   perror.InvalidServiceError,
				Msg:    "error in getService.",
				Origin: fmt.Errorf("%s's exporters is empty", pipeName),
			}
		}
		// check pipeline sensors & exporters are valid
		for _, sensorName := range pipeline.Sensors {
			sensor, ok := c.compose.sensors[sensorName]
			if !ok {
				return Service{}, perror.PolvoComposeError{
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
				return Service{}, perror.PolvoComposeError{
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
		return Service{}, perror.PolvoGeneralError{
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

	return service, nil
}
