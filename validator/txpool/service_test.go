package txpool

import "github.com/prysmaticlabs/prysm/shared"

// Verifies that TXPool implements the Service interface.
var _ = shared.Service(&TXPool{})
