package filter

import (
	"gopkg.in/yaml.v3"
)

type Filter struct {
	Version string `yaml:"version"`
}

type Rule map[string]yaml.Node

type Detection struct {
	Detections map[string]Rule `yaml:"detection"`
	Conditions string          `yaml:"condition"`
}
