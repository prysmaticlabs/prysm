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
	SkipSlots             []uint64                     `yaml:"skip_slots"`
	DepositSlots          []uint64                     `yaml:"deposit_slots"`
	Deposits              []*StateTestDeposit          `yaml:"deposits"`
	ProposerSlashings     []*StateTestProposerSlashing `yaml:"proposer_slashings"`
	CasperSlashings       []*StateTestCasperSlashing   `yaml:"casper_slashings"`
	ValidatorExits        []*StateTestValidatorExit    `yaml:"validator_exits"`
	EpochLength           uint64                       `yaml:"epoch_length"`
	ShardCount            uint64                       `yaml:"shard_count"`
	DepositsForChainStart uint64                       `yaml:"deposits_for_chain_start"`
	NumSlots              uint64                       `yaml:"num_slots"`
}

// StateTestDeposit --
type StateTestDeposit struct {
	Slot        uint64 `yaml:"slot"`
	Amount      uint64 `yaml:"amount"`
	MerkleIndex uint64 `yaml:"merkle_index"`
	Pubkey      string `yaml:"pubkey"`
}

// StateTestProposerSlashing --
type StateTestProposerSlashing struct {
	Slot           uint64 `yaml:"slot"`
	ProposerIndex  uint32 `yaml:"proposer_index"`
	Proposal1Shard uint64 `yaml:"proposal_1_shard"`
	Proposal2Shard uint64 `yaml:"proposal_2_shard"`
	Proposal1Slot  uint64 `yaml:"proposal_1_slot"`
	Proposal2Slot  uint64 `yaml:"proposal_2_slot"`
	Proposal1Root  string `yaml:"proposal_1_root"`
	Proposal2Root  string `yaml:"proposal_2_root"`
}

// StateTestCasperSlashing --
type StateTestCasperSlashing struct {
	Slot                     uint64   `yaml:"slot"`
	Votes1Slot               uint64   `yaml:"votes_1_slot"`
	Votes1JustifiedSlot      uint64   `yaml:"votes_1_justified_slot"`
	Votes1CustodyBit0Indices []uint32 `yaml:"votes_1_custody_0_indices"`
	Votes1CustodyBit1Indices []uint32 `yaml:"votes_1_custody_1_indices"`
	Votes2Slot               uint64   `yaml:"votes_2_slot"`
	Votes2JustifiedSlot      uint64   `yaml:"votes_2_justified_slot"`
	Votes2CustodyBit0Indices []uint32 `yaml:"votes_2_custody_0_indices"`
	Votes2CustodyBit1Indices []uint32 `yaml:"votes_2_custody_1_indices"`
}

// StateTestValidatorExit --
type StateTestValidatorExit struct {
	Slot           uint64 `yaml:"slot"`
	ValidatorIndex uint32 `yaml:"validator_index"`
}

// StateTestResults --
type StateTestResults struct {
	Slot                uint64
	NumValidators       int      `yaml:"num_validators"`
	PenalizedValidators []uint32 `yaml:"penalized_validators"`
	ExitedValidators    []uint32 `yaml:"exited_validators"`
}
