package validator_service_config

import (
	"github.com/ethereum/go-ethereum/common"
	field_params "github.com/prysmaticlabs/prysm/config/fieldparams"
)

// ProposerSettingsConfig is the struct representation of the JSON or YAML payload set in the validator through the CLI.
// ProposeConfig is the map of validator address to fee recipient options all in hex format.
// DefaultConfig is the default fee recipient address for all validators unless otherwise specified in the propose config.required.
type ProposerSettingsConfig struct {
	ProposeConfig map[string]*ProposerOptionsConfig `json:"proposer_config" yaml:"proposer_config"`
	DefaultConfig *ProposerOptionsConfig            `json:"default_config" yaml:"default_config"`
}

// ProposerOptionsConfig is the struct representation of the JSON config file set in the validator through the CLI.
// FeeRecipient is set to an eth address in hex string format with 0x prefix.
// GasLimit is a number set to help the network decide on the maximum gas in each block.
type ProposerOptionsConfig struct {
	FeeRecipient string `json:"fee_recipient" yaml:"fee_recipient"`
	GasLimit     uint64 `json:"gas_limit,omitempty" yaml:"gas_limit,omitempty"`
}

// ProposerSettings is a Prysm internal representation of the fee recipient config on the validator client.
// ProposerSettingsConfig maps to ProposerSettings on import through the CLI.
type ProposerSettings struct {
	ProposeConfig map[[field_params.BLSPubkeyLength]byte]*ProposerOptions
	DefaultConfig *ProposerOptions
}

// ProposerOptions is a Prysm internal representation of the ProposerOptionsConfig on the validator client in bytes format instead of hex.
type ProposerOptions struct {
	FeeRecipient common.Address
	GasLimit     uint64
}
