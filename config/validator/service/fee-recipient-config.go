package validator_service_config

import (
	"github.com/ethereum/go-ethereum/common"
	field_params "github.com/prysmaticlabs/prysm/config/fieldparams"
)

// ValidatorProposerSettingsConfig is the struct representation of the JSON or YAML payload set in the validator through the CLI.
// ProposeConfig is the map of validator address to fee recipient options all in hex format.
// DefaultConfig is the default fee recipient address for all validators unless otherwise specified in the propose config.required.
type ValidatorProposerSettingsConfig struct {
	ProposeConfig map[string]*ValidatorProposerOptionsConfig `json:"proposer_config" yaml:"proposer_config"`
	DefaultConfig *ValidatorProposerOptionsConfig            `json:"default_config" yaml:"default_config"`
}

// ValidatorProposerOptionsConfig is the struct representation of the JSON config file set in the validator through the CLI.
// FeeRecipient is set to an eth address in hex string format with 0x prefix.
// GasLimit is a number set to help the network decide on the maximum gas in each block.
type ValidatorProposerOptionsConfig struct {
	FeeRecipient string `json:"fee_recipient" yaml:"fee_recipient"`
	GasLimit     uint64 `json:"gas_limit,omitempty" yaml:"gas_limit,omitempty"`
}

// ValidatorProposerSettings is a Prysm internal representation of the fee recipient config on the validator client.
// ValidatorProposerSettingsConfig maps to ValidatorProposerSettings on import through the CLI.
type ValidatorProposerSettings struct {
	ProposeConfig map[[field_params.BLSPubkeyLength]byte]*ValidatorProposerOptions
	DefaultConfig *ValidatorProposerOptions
}

// ValidatorProposerOptions is a Prysm internal representation of the ValidatorProposerOptionsConfig on the validator client in bytes format instead of hex.
type ValidatorProposerOptions struct {
	FeeRecipient common.Address
	GasLimit     uint64
}
