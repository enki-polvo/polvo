package compose

import (
	"os"
	perror "polvo/error"
	plogger "polvo/logger"

	"gopkg.in/yaml.v3"
	_ "gopkg.in/yaml.v3"
)

type composer struct {
	os   string
	arch string
	// TODO: dev sensor type
	sensors map[string]Sensor
	// dependency
	loger plogger.PolvoLogger
}

func NewComposer(loger plogger.PolvoLogger, configPath string) (*composer, error) {
	var (
		wrapper ComposeWrapper
	)

	newComp := new(composer)

	// set dependency
	newComp.loger = loger

	// read config file
	file, err := os.ReadFile(configPath)
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

	// TODO: check config contents
	newComp.os = wrapper.Os
	newComp.arch = wrapper.Arch

	newComp.sensors = make(map[string]Sensor)
	for i, sensor := range wrapper.Sensors {
		if !sensor.Enable {
			continue
		}
		newComp.sensors[i] = Sensor{
			os:       newComp.os,
			arch:     newComp.arch,
			toolPath: sensor.ToolPath,
			command:  sensor.Command,
		}
	}

	return newComp, nil
}
