package validator

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
	// Use minimal config to reduce test setup time.
	prevConfig := params.BeaconConfig().Copy()
	params.OverrideBeaconConfig(params.MinimalSpecConfig())

	retVal := m.Run()

	// Reset configuration.
	params.OverrideBeaconConfig(prevConfig)
	os.Exit(retVal)
}
