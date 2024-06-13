package electra

import (
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
)

var (
	SwitchToCompoundingValidator        = helpers.SwitchToCompoundingValidator
	QueueExcessActiveBalance            = helpers.QueueExcessActiveBalance
	QueueEntireBalanceAndResetValidator = helpers.QueueEntireBalanceAndResetValidator
)
