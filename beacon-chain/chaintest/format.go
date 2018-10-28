package main

type chainTests struct {
	Title     string
	Summary   string
	TestSuite string
	TestCases []*cases
}

type cases struct {
	Config *config
	Slots  []*slot
	Result *result
}

type config struct {
	ValidatorCount   int
	CycleLength      int
	ShardCount       int
	MinCommitteeSize int
}

type slot struct {
	SlotNumber   int
	NewBlock     *testBlock
	Attestations []*testAttestation
}

type result struct {
	Head               string
	LastJustifiedBlock string
	LastFinalizedBlock string
}

type testBlock struct {
	ID     string
	Parent string
}

type testAttestation struct {
	Block         string
	Validators    string
	CommitteeSlot int
}
