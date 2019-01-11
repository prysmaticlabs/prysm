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
	SkipSlots             []uint64            `yaml:"skip_slots"`
	DepositSlots          []uint64            `yaml:"deposit_slots"`
	Deposits              []*StateTestDeposit `yaml:"deposits"`
	EpochLength           uint64              `yaml:"epoch_length"`
	ShardCount            uint64              `yaml:"shard_count"`
	DepositsForChainStart uint64              `yaml:"deposits_for_chain_start"`
	NumSlots              uint64              `yaml:"num_slots"`
}

// StateTestDeposit --
type StateTestDeposit struct {
	Slot        uint64 `yaml:"slot"`
	Amount      uint64 `yaml:"amount"`
	MerkleIndex uint64 `yaml:"merkle_index"`
	Pubkey      string `yaml:"pubkey"`
}

// StateTestResults --
type StateTestResults struct {
	Slot          uint64
	NumValidators int `yaml:"num_validators"`
}
