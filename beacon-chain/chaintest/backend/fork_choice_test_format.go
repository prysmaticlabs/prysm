package backend

// ForkChoiceTest --
type ForkChoiceTest struct {
	Title     string
	Summary   string
	TestSuite string                `yaml:"test_suite"`
	TestCases []*ForkChoiceTestCase `yaml:"test_cases"`
}

// ForkChoiceTestCase --
type ForkChoiceTestCase struct {
	Config  *ForkChoiceTestConfig `yaml:"config"`
	Slots   []*ForkChoiceTestSlot `yaml:"slots,flow"`
	Results *ForkChoiceTestResult `yaml:"results"`
}

// ForkChoiceTestConfig --
type ForkChoiceTestConfig struct {
	ValidatorCount   uint32 `yaml:"validator_count"`
	CycleLength      uint32 `yaml:"cycle_length"`
	ShardCount       uint32 `yaml:"shard_count"`
	MinCommitteeSize uint32 `yaml:"min_committee_size"`
}

// ForkChoiceTestSlot --
type ForkChoiceTestSlot struct {
	SlotNumber   uint32             `yaml:"slot_number"`
	NewBlock     *TestBlock         `yaml:"new_block"`
	Attestations []*TestAttestation `yaml:",flow"`
}

// ForkChoiceTestResult --
type ForkChoiceTestResult struct {
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
	CommitteeSlot     uint32 `yaml:"committee_slot"`
}
