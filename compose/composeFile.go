package compose

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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
	GetSensorCompose(name string) *SensorInfo
	GetExporterCompose(name string) *ExporterInfo
	GetServiceCompose() *Service
	GetCompose() *Compose
	String() string
}

type composeFile struct {
	compose *Compose
}

// Getter for Sensor
func (c *composeFile) GetSensorCompose(name string) *SensorInfo {
	var (
		ret *SensorInfo
		ok  bool
	)

	if ret, ok = c.compose.Sensors[name]; !ok {
		return nil
	}
	return ret
}

// Getter for Exporter
func (c *composeFile) GetExporterCompose(name string) *ExporterInfo {
	var (
		ret *ExporterInfo
		ok  bool
	)

	if ret, ok = c.compose.Exporters[name]; !ok {
		return nil
	}
	return ret
}

// Getter for Service
func (c *composeFile) GetServiceCompose() *Service {
	return c.compose.Service
}

// Getter for Compose
func (c *composeFile) GetCompose() *Compose {
	return c.compose
}

// Stringer for ComposeFile
func (c *composeFile) String() string {
	sensorStr := "compose: \n-------------Sensor --------------\n"
	for sensorName, sensor := range c.compose.Sensors {
		sensorStr += fmt.Sprintf("%s:\n\texec_path: %v\n\tparam: %s\n\trun_as_root: %v\n\tevents_header: %v\n", sensorName,
			sensor.ExecPath,
			sensor.Param,
			sensor.RunAsRoot,
			sensor.EventsHeader)
	}
	exporterStr := "-------------Exporter --------------\n"
	for exporterName, exporter := range c.compose.Exporters {
		exporterStr += fmt.Sprintf("%s:\n\tmode: %v\n\tdestination: %v\n\ttimeout: %v\n", exporterName, exporter.Mode, exporter.Destination, exporter.Timeout)
	}
	serviceStr := fmt.Sprintf("-------------Service --------------\n\tmachine: %v\n\tos: %v\n\tarch: %v\n\tgroup: %v\n\tdescription: %v",
		c.compose.Service.Machine,
		c.compose.Service.OS,
		c.compose.Service.Arch,
		c.compose.Service.Group,
		c.compose.Service.Description)
	for pipeName, pipeline := range c.compose.Service.Pipeline {
		serviceStr += fmt.Sprintf("\n\t%s:\n\t\t%s: %v", pipeName, "sensors", func() []string {
			ret := make([]string, 0)
			for _, sensor := range pipeline.Sensors {
				ret = append(ret, sensor.Name)
			}
			return ret
		}())
		serviceStr += fmt.Sprintf("\n\t\t%s: %v", "exporter", pipeline.Exporter.Name)
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
	newComp.compose.Sensors, err = newComp.getSensor(wrapper.Sensors)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composeFile",
		}
	}
	// get exporter from wrapper
	newComp.compose.Exporters, err = newComp.getExporters(wrapper.Exporters)
	if err != nil {
		return nil, perror.PolvoComposeError{
			Code:   perror.ErrInvalidCompose,
			Origin: err,
			Msg:    "error while Construct new composeFile",
		}
	}
	// get service from wrapper
	newComp.compose.Service, err = newComp.getService(wrapper.Service)
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
func (c *composeFile) getSensor(wrapperMap map[string]SensorWrapper) (map[string]*SensorInfo, error) {
	var (
		sensorMap    map[string]*SensorInfo
		execFileInfo os.FileInfo
		err          error
	)

	sensorMap = make(map[string]*SensorInfo)

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
		sensorMap[sensorName] = &SensorInfo{
			Name:         sensorName,
			ExecPath:     sensorObj.ExecPath,
			Param:        sensorObj.Param,
			RunAsRoot:    sensorObj.RunAsRoot,
			EventsHeader: sensorObj.EventsHeader,
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

func isValidPath(path string) (bool, error) {
	dirPath := filepath.Dir(path)

	// check directory path exists.
	if _, err := os.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return true, perror.PolvoGeneralError{
			Code:   perror.SystemError,
			Msg:    "error in isValidPath.",
			Origin: err,
		}
	}
	return true, nil
}

// getExporter constructs Exporter struct from ExporterWrapper & verifies the exporter compose file.
func (c *composeFile) getExporters(wrapperMap map[string]ExporterWrapper) (map[string]*ExporterInfo, error) {
	exporterMap := make(map[string]*ExporterInfo)

	for exporterName, exporterObj := range wrapperMap {
		// null check
		if exporterObj.Mode == "" {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidExporterError,
				Msg:    "error in getExporter.",
				Origin: fmt.Errorf("mode is empty"),
			}
		}
		if exporterObj.Destination == "" {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidExporterError,
				Msg:    "error in getExporter.",
				Origin: fmt.Errorf("destination is empty"),
			}
		}
		// check mode is valid
		if !AvailableExporterMode.IsValid(exporterObj.Mode) {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidExporterError,
				Msg:    "error in getExporter.",
				Origin: fmt.Errorf("mode [%s] is not valid", exporterObj.Mode),
			}
		}
		// check destination is valid
		switch exporterObj.Mode {
		case "file":
			// check destination is valid path
			result, err := isValidPath(exporterObj.Destination)
			if err != nil || !result {
				return nil, perror.PolvoComposeError{
					Code:   perror.InvalidExporterError,
					Msg:    "error in getExporter.",
					Origin: fmt.Errorf("destination [%s] is not valid path", exporterObj.Destination),
				}
			}
		case "network":
			// check destination is ip url format
			// check destination is ip url format
			if !isValidIPPort(exporterObj.Destination) {
				return nil, perror.PolvoComposeError{
					Code:   perror.InvalidExporterError,
					Msg:    "error in getExporter.",
					Origin: fmt.Errorf("destination is not valid url"),
				}
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
		exporterMap[exporterName] = &ExporterInfo{
			Name:        exporterName,
			Destination: exporterObj.Destination,
			Timeout:     exporterObj.Timeout,
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
	pipelines := make(map[string]PipelineInfo)
	for pipeName, pipeline := range wrapper.Pipelines {
		sensors := make([]*SensorInfo, 0)
		// null check
		if len(pipeline.Sensors) <= 0 {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidServiceError,
				Msg:    "error in getService.",
				Origin: fmt.Errorf("%s's sensors is empty", pipeName),
			}
		}
		if len(pipeline.Exporter) <= 0 {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidServiceError,
				Msg:    "error in getService.",
				Origin: fmt.Errorf("%s's exporters is empty", pipeName),
			}
		}
		// check pipeline sensors & exporters are valid
		for _, sensorName := range pipeline.Sensors {
			sensor, ok := c.compose.Sensors[sensorName]
			if !ok {
				return nil, perror.PolvoComposeError{
					Code:   perror.InvalidServiceError,
					Msg:    "error in getService.",
					Origin: fmt.Errorf("%s's sensor %s is not defined", pipeName, sensorName),
				}
			}
			sensors = append(sensors, sensor)
		}
		exporter, ok := c.compose.Exporters[pipeline.Exporter]
		if !ok {
			return nil, perror.PolvoComposeError{
				Code:   perror.InvalidServiceError,
				Msg:    "error in getService.",
				Origin: fmt.Errorf("%s's exporter %s is not defined", pipeName, pipeline.Exporter),
			}
		}
		// TODO: read valid exporter & sensor from config file
		// add pipeline to map
		pipelines[pipeName] = PipelineInfo{
			Sensors:  sensors,
			Exporter: exporter,
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
		Machine:     serviceMachine,
		OS:          serviceOS,
		Arch:        serviceArch,
		Group:       wrapper.Group,
		Description: wrapper.Description,
		Pipeline:    pipelines,
	}

	return &service, nil
}
