package backend

// ChainTest --
type ChainTest struct {
	Title     string
	Summary   string
	TestSuite string
	TestCases []*ChainTestCases
}

// ChainTestCases --
type ChainTestCases struct {
	Config *ChainTestConfig
	Slots  []*ChainTestSlot
	Result *ChainTestResult
}

// ChainTestConfig --
type ChainTestConfig struct {
	ValidatorCount   int
	CycleLength      int
	ShardCount       int
	MinCommitteeSize int
}

// ChainTestSlot --
type ChainTestSlot struct {
	SlotNumber   int
	NewBlock     *TestBlock
	Attestations []*TestAttestation
}

// ChainTestResult --
type ChainTestResult struct {
	Head               string
	LastJustifiedBlock string
	LastFinalizedBlock string
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
