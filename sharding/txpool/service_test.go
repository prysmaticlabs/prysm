package txpool

import "github.com/prysmaticlabs/geth-sharding/sharding"

// Verifies that TXPool implements the Service interface.
var _ = sharding.Service(&TXPool{})
