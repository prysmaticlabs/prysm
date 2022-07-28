package kv

import (
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
)

func init() {
	// Override network name so that hardcoded genesis files are not loaded.
	if err := params.SetActive(params.MainnetTestConfig()); err != nil {
		panic(err)
	}
	features.Init(&features.Flags{
		EnableOnlyBlindedBeaconBlocks: true,
	})
}
