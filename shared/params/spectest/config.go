// Package spectest allows for easy switching of chain
// configuration parameters in spec conformity unit tests.
package spectest

import (
	"errors"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// SetConfig sets the global params for spec tests depending on the option chosen.
// Provides reset function allowing to get back to the previous configuration at the end of a test.
func SetConfig(config string) (func(), error) {
	switch config {
	case "minimal":
		resetFunc := params.OverrideBeaconConfigWithReset(params.MinimalSpecConfig().Copy())
		return resetFunc, nil
	case "mainnet":
		resetFunc := params.OverrideBeaconConfigWithReset(params.MainnetConfig().Copy())
		return resetFunc, nil
	case "":
		return nil, errors.New("no config provided")
	default:
		return nil, fmt.Errorf("did not receive a valid config, instead received this %s", config)
	}
}
