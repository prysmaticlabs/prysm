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
	ValidatorCount   int `yaml:"validator_count"`
	CycleLength      int `yaml:"cycle_length"`
	ShardCount       int `yaml:"shard_count"`
	MinCommitteeSize int `yaml:"min_committee_size"`
}

// ChainTestSlot --
type ChainTestSlot struct {
	SlotNumber   int                `yaml:"slot_number"`
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
	ID     string
	Parent string
}

// TestAttestation --
type TestAttestation struct {
	Block         string
	Validators    string
	CommitteeSlot int
}
