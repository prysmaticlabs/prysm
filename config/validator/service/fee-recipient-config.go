package validator_service_config

type PrepareBeaconProposalFileConfig struct {
	ProposeConfig map[string]*ValidatorProposerOptions `json:"proposer_config"`
	DefaultConfig *ValidatorProposerOptions            `json:"default_config"`
}

type ValidatorProposerOptions struct {
	FeeRecipient string `json:"fee_recipient"`
}
