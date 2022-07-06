package validator_service_config

import (
	"github.com/ethereum/go-ethereum/common"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
)

// ProposerSettingsPayload is the struct representation of the JSON or YAML payload set in the validator through the CLI.
// ProposerConfig is the map of validator address to fee recipient options all in hex format.
// DefaultConfig is the default fee recipient address for all validators unless otherwise specified in the propose config.required.
type ProposerSettingsPayload struct {
	ProposerConfig map[string]*ProposerOptionPayload `json:"proposer_config" yaml:"proposer_config"`
	DefaultConfig  *ProposerOptionPayload            `json:"default_config" yaml:"default_config"`
}

// ProposerOptionPayload is the struct representation of the JSON config file set in the validator through the CLI.
// FeeRecipient is set to an eth address in hex string format with 0x prefix.
type ProposerOptionPayload struct {
	FeeRecipient          string                 `json:"fee_recipient" yaml:"fee_recipient"`
	ValidatorRegistration *ValidatorRegistration `json:"validator_registration" yaml:"validator_registration"`
}

// ValidatorRegistration is the struct representation of the JSON config file set in the validator through the CLI.
// GasLimit is a number set to help the network decide on the maximum gas in each block.
type ValidatorRegistration struct {
	Enable   bool   `json:"enable" yaml:"enable"`
	GasLimit uint64 `json:"gas_limit,omitempty" yaml:"gas_limit,omitempty"`
}

// ProposerSettings is a Prysm internal representation of the fee recipient config on the validator client.
// ProposerSettingsPayload maps to ProposerSettings on import through the CLI.
type ProposerSettings struct {
	ProposeConfig map[[fieldparams.BLSPubkeyLength]byte]*ProposerOption
	DefaultConfig *ProposerOption
}

// ProposerOption is a Prysm internal representation of the ProposerOptionPayload on the validator client in bytes format instead of hex.
type ProposerOption struct {
	FeeRecipient          common.Address
	ValidatorRegistration *ValidatorRegistration
}

// DefaultProposerOption returns a Proposer Option with defaults filled
func DefaultProposerOption() ProposerOption {
	return ProposerOption{
		FeeRecipient:          params.BeaconConfig().DefaultFeeRecipient,
		ValidatorRegistration: nil,
	}
}
