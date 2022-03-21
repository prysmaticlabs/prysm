package validator_service_config

import (
	"github.com/ethereum/go-ethereum/common"
	field_params "github.com/prysmaticlabs/prysm/config/fieldparams"
)

type FeeRecipientFileConfig struct {
	ProposeConfig map[string]*FeeRecipientFileOptions `json:"proposer_config"`
	DefaultConfig *FeeRecipientFileOptions            `json:"default_config"`
}

type FeeRecipientFileOptions struct {
	FeeRecipient string `json:"fee_recipient"`
}

type FeeRecipientConfig struct {
	ProposeConfig map[[field_params.BLSPubkeyLength]byte]*FeeRecipientOptions
	DefaultConfig *FeeRecipientOptions
}

type FeeRecipientOptions struct {
	FeeRecipient common.Address
}
