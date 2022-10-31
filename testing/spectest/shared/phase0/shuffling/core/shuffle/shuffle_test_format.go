package shuffle

import types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"

// ShuffleTestCase --
type ShuffleTestCase struct {
	Seed    string                 `yaml:"seed"`
	Mapping []types.ValidatorIndex `yaml:"mapping"`
	Count   uint64                 `yaml:"count"`
}
