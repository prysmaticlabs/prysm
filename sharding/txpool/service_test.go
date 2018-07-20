package txpool

import "github.com/prysmaticlabs/prysm/sharding/types"

// Verifies that TXPool implements the Service interface.
var _ = types.Service(&TXPool{})
