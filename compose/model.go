package compose

type ExporterMode []string

var AvailableExporterMode ExporterMode = ExporterMode{
	"file",
	"network",
}

func (em ExporterMode) IsValid(mode string) bool {
	for _, m := range em {
		if m == mode {
			return true
		}
	}
	return false
}

type SensorWrapper struct {
	ExecPath     string              `yaml:"exec_path"`
	Param        string              `yaml:"param"`
	RunAsRoot    bool                `yaml:"run_as_root"`
	EventsHeader map[string][]string `yaml:"events_header"`
}

type ExporterWrapper struct {
	Mode        string `yaml:"mode"`
	Destination string `yaml:"destination"`
	Timeout     int    `yaml:"timeout"`
}

type PipelineWrapper struct {
	Sensors  []string `yaml:"sensors"`
	Exporter string   `yaml:"exporter"`
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
	Sensors   map[string]*SensorInfo
	Exporters map[string]*ExporterInfo
	Service   *Service
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
	Sensors  []*SensorInfo
	Exporter *ExporterInfo
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
	Mode        string
	Destination string
	Timeout     int
}
