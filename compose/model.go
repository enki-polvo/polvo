package compose

type SensorWrapper struct {
	ExecPath string `yaml:"exec_path"`
	Param    string `yaml:"param"`
}

type ExporterWrapper struct {
	Destination string `yaml:"destination"`
	Timeout     int    `yaml:"timeout"`
}

type Pipeline struct {
	Sensors   []string `yaml:"sensors"`
	Exporters []string `yaml:"exporters"`
}

type Service struct {
	Machine     string `yaml:"machine"`
	Os          string `yaml:"os"`
	Arch        string `yaml:"arch"`
	Description string `yaml:"description"`
	Pipeline    string `yaml:"pipeline"`
}

type ComposeWrapper struct {
	Sensors   map[string]SensorWrapper   `yaml:"sensors"`
	Exporters map[string]ExporterWrapper `yaml:"exporters"`
}
