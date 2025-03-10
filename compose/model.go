package compose

import (
	"net/url"
)

type SensorWrapper struct {
	ExecPath     string              `yaml:"exec_path"`
	Param        string              `yaml:"param"`
	RunAsRoot    bool                `yaml:"run_as_root"`
	EventsHeader map[string][]string `yaml:"events_header"`
}

type ExporterWrapper struct {
	Destination string `yaml:"destination"`
	Timeout     int    `yaml:"timeout"`
}

type PipelineWrapper struct {
	Sensors   []string `yaml:"sensors"`
	Exporters []string `yaml:"exporters"`
}

type ServiceWrapper struct {
	Os          string                     `yaml:"os"`
	Arch        string                     `yaml:"arch"`
	Description string                     `yaml:"description"`
	Group       string                     `yaml:"group"`
	Pipelines   map[string]PipelineWrapper `yaml:"pipelines"`
}

type ComposeWrapper struct {
	Sensors   map[string]SensorWrapper   `yaml:"sensors"`
	Exporters map[string]ExporterWrapper `yaml:"exporters"`
	Service   ServiceWrapper             `yaml:"service"`
}

type Compose struct {
	sensors   map[string]Sensor
	exporters map[string]Exporter
	service   Service
}

type Service struct {
	machine     string
	os          string
	arch        string
	group       string
	description string
	pipeline    map[string]Pipeline
}

type Pipeline struct {
	sensors   []*Sensor
	exporters []*Exporter
}

type Sensor struct {
	execPath     string
	param        string
	runAsRoot    bool
	eventsHeader map[string][]string
}

type Exporter struct {
	destination *url.URL
	timeout     int
}
