package validator_service_config

import (
	"github.com/ethereum/go-ethereum/common"
	field_params "github.com/prysmaticlabs/prysm/config/fieldparams"
)

// FeeRecipientFileConfig is the struct representation of the JSON config file set in the validator through the CLI.
// ProposeConfig is the map of validator address to fee recipient options all in hex format.
// DefaultConfig is the default fee recipient address for all validators unless otherwise specified in the propose config.required.
type FeeRecipientFileConfig struct {
	ProposeConfig map[string]*FeeRecipientFileOptions `json:"proposer_config"`
	DefaultConfig *FeeRecipientFileOptions            `json:"default_config"`
}

// FeeRecipientFileOptions is the struct representation of the JSON config file set in the validator through the CLI.
// FeeRecipient is set to an eth address in hex string format with 0x prefix.
type FeeRecipientFileOptions struct {
	FeeRecipient string `json:"fee_recipient"`
}

// FeeRecipientConfig is a Prysm internal representation of the fee recipient config on the validator client.
// FeeRecipientFileConfig maps to FeeRecipientConfig on import through the CLI.
type FeeRecipientConfig struct {
	ProposeConfig map[[field_params.BLSPubkeyLength]byte]*FeeRecipientOptions
	DefaultConfig *FeeRecipientOptions
}

// FeeRecipientOptions is a Prysm internal representation of the FeeRecipientFileOptions on the validator client in bytes format instead of hex.
type FeeRecipientOptions struct {
	FeeRecipient common.Address
}
