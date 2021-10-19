package shuffle

import types "github.com/prysmaticlabs/eth2-types"

// ShuffleTestCase --
type ShuffleTestCase struct {
	Seed    string                 `yaml:"seed"`
	Count   uint64                 `yaml:"count"`
	Mapping []types.ValidatorIndex `yaml:"mapping"`
}
