package filter

import (
	"gopkg.in/yaml.v3"
)

type Rule map[string]yaml.Node

type Selections map[string]Rule

type Filter struct {
	Version string     `yaml:"version"`
	Allow   Selections `yaml:"allow"`
	Deny    Selections `yaml:"deny"`
}
