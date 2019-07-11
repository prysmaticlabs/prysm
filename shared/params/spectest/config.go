package spectest

import (
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// SetConfig sets the global params for spec tests depending on the option chosen.
func SetConfig(config string) error {
	switch config {
	case "minimal":
		newConfig := params.MinimalSpecConfig()
		params.OverrideBeaconConfig(newConfig)
		return nil
	case "mainnet":
		newConfig := params.MainnetConfig()
		params.OverrideBeaconConfig(newConfig)
		return nil
	case "":
		return errors.New("no config provided")
	default:
		return fmt.Errorf("did not receive a valid config, instead received this %s", config)
	}
}
