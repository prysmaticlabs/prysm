// Package utils allows for easy switching of chain configuration parameters in spec conformity unit tests.
package utils

import (
	"errors"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// SetConfig sets the global params for spec tests depending on the option chosen.
// Provides reset function allowing to get back to the previous configuration at the end of a test.
func SetConfig(t testing.TB, config string) error {
	params.SetupTestConfigCleanup(t)
	switch config {
	case "minimal":
		params.OverrideBeaconConfig(params.MinimalSpecConfig())
		return nil
	case "mainnet":
		params.OverrideBeaconConfig(params.MainnetConfig())
		return nil
	case "":
		return errors.New("no config provided")
	default:
		return fmt.Errorf("did not receive a valid config, instead received this %s", config)
	}
}
