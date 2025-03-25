package filter

import "gopkg.in/yaml.v3"

type Filter struct {
	Version string `yaml:"version"`
}

type Rule struct {
	Conditions string               `yaml:"condition"`
	Detections map[string]Detection `yaml:"detection"`
}

type Detection struct {
	Node yaml.Node `yaml:",omitempty"`
}
