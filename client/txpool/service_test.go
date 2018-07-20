package txpool

import "github.com/prysmaticlabs/prysm/client/types"

// Verifies that TXPool implements the Service interface.
var _ = types.Service(&TXPool{})
