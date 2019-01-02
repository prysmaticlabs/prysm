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
	Config  *StateTestConfig  `yaml:"config"`
	Results *StateTestResults `yaml:"results"`
}

// StateTestConfig --
type StateTestConfig struct {
	PublishBlocks         bool   `yaml:"publish_blocks"`
	EpochLength           uint64 `yaml:"epoch_length"`
	ShardCount            uint64 `yaml:"shard_count"`
	DepositsForChainStart uint64 `yaml:"deposits_for_chain_start"`
	NumSlots              uint64 `yaml:"num_slots"`
}

// StateTestResults --
type StateTestResults struct {
	Slot uint64
}
