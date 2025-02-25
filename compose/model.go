package compose

type Sensor struct {
	os       string
	arch     string
	toolPath string
	command  string
}

type SensorWrapper struct {
	Enable   bool   `yaml:"enable"`
	ToolPath string `yaml:"tool_path"`
	Command  string `yaml:"command"`
}

type ComposeWrapper struct {
	Os      string                   `yaml:"os"`
	Arch    string                   `yaml:"arch"`
	Sensors map[string]SensorWrapper `yaml:"sensors"`
}
