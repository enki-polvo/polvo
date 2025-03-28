package filter

import (
	"gopkg.in/yaml.v3"
)

type Rule map[string]yaml.Node

type Deny struct {
	Condition Rule `yaml:"condition"`
	Exception Rule `yaml:"exception,omitempty"`
}

type Filter struct {
	Version string          `yaml:"version"`
	Deny    map[string]Deny `yaml:"deny"`
}
