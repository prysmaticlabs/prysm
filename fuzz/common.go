package fuzz

import (
	"os"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

const ENV_BLS = "BLS_ENABLED"

func init() {
	var blsEnabled bool
	if value, exists := os.LookupEnv(ENV_BLS); exists {
		blsEnabled = value == "1"
	}
	featureconfig.Init(&featureconfig.Flags{
		SkipBLSVerify: !blsEnabled,
	})
}
