package spectest

import (
	"fmt"
	"io/ioutil"

	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// SetConfig sets the global params for spec tests depending on the option chosen.
func SetConfig(config string) error {
	file, err := ioutil.ReadFile(config + ".yaml")
	if err != nil {
		return fmt.Errorf("could not find config yaml %v", err)
	}
	decoded := &params.BeaconChainConfig{}
	if err := yaml.Unmarshal(file, decoded); err != nil {
		return fmt.Errorf("could not unmarshal YAML file into config struct: %v", err)
	}
	params.OverrideBeaconConfig(decoded)
	return nil
}
