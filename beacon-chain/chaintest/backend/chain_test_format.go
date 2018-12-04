package backend

// ChainTest --
type ChainTest struct {
	Title     string
	Summary   string
	TestSuite string           `yaml:"test_suite"`
	TestCases []*ChainTestCase `yaml:"test_cases"`
}

// ChainTestCase --
type ChainTestCase struct {
	Config  *ChainTestConfig  `yaml:"config"`
	Slots   []*ChainTestSlot  `yaml:"slots,flow"`
	Results *ChainTestResults `yaml:"results"`
}

// ChainTestConfig --
type ChainTestConfig struct {
	ValidatorCount   uint64 `yaml:"validator_count"`
	CycleLength      uint64 `yaml:"cycle_length"`
	ShardCount       uint64 `yaml:"shard_count"`
	MinCommitteeSize uint64 `yaml:"min_committee_size"`
}

// ChainTestSlot --
type ChainTestSlot struct {
	SlotNumber   uint64             `yaml:"slot_number"`
	NewBlock     *TestBlock         `yaml:"new_block"`
	Attestations []*TestAttestation `yaml:",flow"`
}

// ChainTestResults --
type ChainTestResults struct {
	Head               string
	LastJustifiedBlock string `yaml:"last_justified_block"`
	LastFinalizedBlock string `yaml:"last_finalized_block"`
}

// TestBlock --
type TestBlock struct {
	ID     string `yaml:"ID"`
	Parent string `yaml:"parent"`
}

// TestAttestation --
type TestAttestation struct {
	Block             string `yaml:"block"`
	ValidatorRegistry string `yaml:"validators"`
	CommitteeSlot     uint64 `yaml:"committee_slot"`
}
