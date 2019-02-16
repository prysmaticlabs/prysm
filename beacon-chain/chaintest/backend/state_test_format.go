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
	AttesterSlashings     []*StateTestAttesterSlashing `yaml:"attester_slashings"`
	ValidatorExits        []*StateTestValidatorExit    `yaml:"validator_exits"`
	SlotsPerEpoch         uint64                       `yaml:"slots_per_epoch"`
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
	ProposerIndex  uint64 `yaml:"proposer_index"`
	Proposal1Shard uint64 `yaml:"proposal_1_shard"`
	Proposal2Shard uint64 `yaml:"proposal_2_shard"`
	Proposal1Slot  uint64 `yaml:"proposal_1_slot"`
	Proposal2Slot  uint64 `yaml:"proposal_2_slot"`
	Proposal1Root  string `yaml:"proposal_1_root"`
	Proposal2Root  string `yaml:"proposal_2_root"`
}

// StateTestAttesterSlashing --
type StateTestAttesterSlashing struct {
	Slot                                  uint64   `yaml:"slot"`
	SlashableAttestation1Slot             uint64   `yaml:"slashable_attestation_1_slot"`
	SlashableAttestation1JustifiedEpoch   uint64   `yaml:"slashable_attestation_1_justified_epoch"`
	SlashableAttestation1ValidatorIndices []uint64 `yaml:"slashable_attestation_1_validator_indices"`
	SlashableAttestation1CustodyBitField  string   `yaml:"slashable_attestation_1_custody_bitfield"`
	SlashableAttestation2Slot             uint64   `yaml:"slashable_attestation_2_slot"`
	SlashableAttestation2JustifiedEpoch   uint64   `yaml:"slashable_attestation_2_justified_epoch"`
	SlashableAttestation2ValidatorIndices []uint64 `yaml:"slashable_attestation_2_validator_indices"`
	SlashableAttestation2CustodyBitField  string   `yaml:"slashable_attestation_2_custody_bitfield"`
}

// StateTestValidatorExit --
type StateTestValidatorExit struct {
	Epoch          uint64 `yaml:"epoch"`
	ValidatorIndex uint64 `yaml:"validator_index"`
}

// StateTestResults --
type StateTestResults struct {
	Slot              uint64
	NumValidators     int      `yaml:"num_validators"`
	SlashedValidators []uint64 `yaml:"slashed_validators"`
	ExitedValidators  []uint64 `yaml:"exited_validators"`
}
