package spectest

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// SetConfig sets the global params for spec tests depending on the option chosen.
func SetConfig(config string) error {
	fileName := config + ".yaml"
	fpath, err := filepath.Abs(filepath.Dir(fileName))
	if err != nil {
		return err
	}
	file, err := ioutil.ReadFile(path.Join(fpath, fileName))
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
