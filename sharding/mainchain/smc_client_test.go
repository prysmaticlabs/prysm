package mainchain

import "github.com/ethereum/go-ethereum/sharding"

// Verifies that SMCCLient implements the sharding Service inteface.
var _ = sharding.Service(&SMCClient{})
