package fuzz

import (
	"os"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// EnvBls defines an environment variable name to check whether BLS is enabled or not.
const EnvBls = "BLS_ENABLED"

func init() {
	var blsEnabled bool
	if value, exists := os.LookupEnv(EnvBls); exists {
		blsEnabled = value == "1"
	}
	featureconfig.Init(&featureconfig.Flags{
		SkipBLSVerify: !blsEnabled,
	})
}
