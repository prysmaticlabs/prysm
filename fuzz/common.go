package fuzz

import (
	"os"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

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
