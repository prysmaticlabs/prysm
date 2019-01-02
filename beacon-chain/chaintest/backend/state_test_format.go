package backend

// StateTest --
type StateTest struct {
	Title     string
	Summary   string
	Fork      string           `yaml:"fork"`
	Version   string           `yaml:"version"`
	TestSuite string           `yaml:"test_suite"`
	TestCases []*StateTestCase `yaml:"test_cases"`
}

// StateTestCase --
type StateTestCase struct {
	Config               *StateTestConfig     `yaml:"config"`
	TransitionParameters *StateTestTransition `yaml:"state_transition,flow"`
	Results              *StateTestResults    `yaml:"results"`
}

// StateTestConfig --
type StateTestConfig struct {
	ValidatorCount   uint64 `yaml:"validator_count"`
	CycleLength      uint64 `yaml:"cycle_length"`
	ShardCount       uint64 `yaml:"shard_count"`
	MinCommitteeSize uint64 `yaml:"min_committee_size"`
}

// StateTestTransition --
type StateTestTransition struct {
	NumSlots uint64 `yaml:"num_slots"`
}

// StateTestResults --
type StateTestResults struct {
	Slot uint64
}
