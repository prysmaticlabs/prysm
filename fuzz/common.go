package fuzz

import (
	"os"

	"github.com/prysmaticlabs/prysm/config/features"
)

// EnvBls defines an environment variable name to check whether BLS is enabled or not.
const EnvBls = "BLS_ENABLED"

func init() {
	var blsEnabled bool
	if value, exists := os.LookupEnv(EnvBls); exists {
		blsEnabled = value == "1"
	}
	features.Init(&features.Flags{
		SkipBLSVerify: !blsEnabled,
	})
}
