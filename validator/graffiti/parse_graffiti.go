package graffiti

import (
	"encoding/hex"
	"io/ioutil"
	"strings"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"gopkg.in/yaml.v2"
)

// Graffiti is a graffiti container.
type Graffiti struct {
	Hash     [32]byte
	Default  string                          `yaml:"default,omitempty"`
	Ordered  []string                        `yaml:"ordered,omitempty"`
	Random   []string                        `yaml:"random,omitempty"`
	Specific map[types.ValidatorIndex]string `yaml:"specific,omitempty"`
}

// ParseGraffitiFile parses the graffiti file and returns the graffiti struct.
func ParseGraffitiFile(f string) (*Graffiti, error) {
	parseGraffitiStrings := func(input []string) []string {
		output := make([]string, len(input))
		for _, i := range input {
			output = append(output, ParseHexGraffiti(i))
		}
		return output
	}

	yamlFile, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	g := &Graffiti{}
	if err := yaml.Unmarshal(yamlFile, g); err != nil {
		return nil, err
	}

	specific := make(map[types.ValidatorIndex]string, len(g.Specific))
	for i, o := range g.Specific {
		specific[types.ValidatorIndex(i)] = ParseHexGraffiti(o)
	}
	g.Specific = specific
	g.Default = ParseHexGraffiti(g.Default)
	g.Ordered = parseGraffitiStrings(g.Ordered)
	g.Random = parseGraffitiStrings(g.Random)
	g.Hash = hashutil.Hash(yamlFile)
	return g, nil
}

// ParseHexGraffiti checks if a graffiti input is being represented in hex and converts it to ASCII if so
func ParseHexGraffiti(rawGraffiti string) string {
	splitGraffiti := strings.SplitN(rawGraffiti, ":", 2)
	if strings.ToLower(splitGraffiti[0]) == "hex" {
		if splitGraffiti[1] == "" {
			log.Debug("Blank hex tag to be interpreted as itself")
			return rawGraffiti
		}

		graffiti, err := hex.DecodeString(splitGraffiti[1])
		if err != nil {
			log.WithError(err).Debug("Error while decoding hex string")
			return rawGraffiti
		}
		return string(graffiti)
	}
	return rawGraffiti
}
