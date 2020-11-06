package fuzz

import (
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

func init() {
	featureconfig.Init(&featureconfig.Flags{
		SkipBLSVerify: true,
	})
}
