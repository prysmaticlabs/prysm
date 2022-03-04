package validator_service_config

import github_com_prysmaticlabs_eth2_types "github.com/prysmaticlabs/eth2-types"

type PrepareBeaconProposalFileConfig struct {
	ProposeConfig map[string]*ValidatorProposerOptions `json:"proposer_config" validate:"required"`
	DefaultConfig *ValidatorProposerOptions            `json:"default_config" validate:"required"`
}

type ValidatorProposerOptions struct {
	FeeRecipient string `json:"fee_recipient" validate:"required"`
}

// Temporary should be moved to the proto package
type ValidatorFeeRecipient struct {
	ValidatorIndex github_com_prysmaticlabs_eth2_types.ValidatorIndex `json:"validator_index" validate:"required"`
	FeeRecipient   string                                             `json:"fee_recipient" validate:"required"`
}
