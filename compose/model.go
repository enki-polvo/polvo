package compose

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
	sensors   map[string]SensorInfo
	exporters map[string]ExporterInfo
	service   *Service
}

type Service struct {
	Machine     string
	OS          string
	Arch        string
	Group       string
	Description string
	Pipeline    map[string]PipelineInfo
}

type PipelineInfo struct {
	Sensors   []*SensorInfo
	Exporters []*ExporterInfo
}

type SensorInfo struct {
	Name         string
	ExecPath     string
	Param        string
	RunAsRoot    bool
	EventsHeader map[string][]string
}

type ExporterInfo struct {
	Name        string
	Destination string
	Timeout     int
}
