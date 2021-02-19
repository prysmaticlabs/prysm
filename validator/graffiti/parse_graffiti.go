package graffiti

import (
	"io/ioutil"

	types "github.com/prysmaticlabs/eth2-types"
	"gopkg.in/yaml.v2"
)

type Graffiti struct {
	Default  string                          `yaml:"default,omitempty"`
	Random   []string                        `yaml:"random,omitempty"`
	Specific map[types.ValidatorIndex]string `yaml:"specific,omitempty"`
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
