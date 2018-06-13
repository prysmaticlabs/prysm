package txpool

import "github.com/ethereum/go-ethereum/sharding"

// Verifies that TXPool implements the Service interface.
var _ = sharding.Service(&TXPool{})
