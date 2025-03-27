package filter

import (
	"gopkg.in/yaml.v3"
)

type Rule map[string]yaml.Node

type Filter struct {
	Version string          `yaml:"version"`
	Allow   map[string]Rule `yaml:"allow"`
	Deny    map[string]Rule `yaml:"deny"`
}
