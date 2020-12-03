package graffiti

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Graffiti struct {
	DefaultGraffiti   string            `yaml:"default,omitempty"`
	RandomGraffiti    []string          `yaml:"random,omitempty"`
	ValidatorGraffiti map[uint64]string `yaml:"validators,omitempty"`
}

// ParseGraffitiFile parses the graffiti file and returns the graffiti struct.
func ParseGraffitiFile(f string) (*Graffiti, error) {
	yamlFile, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	g := &Graffiti{}
	if err := yaml.Unmarshal(yamlFile, g); err != nil {
		return nil, err
	}
	return g, nil
}
