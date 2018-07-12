package contracts

import (
	"math/big"
)

// Registry describes the Attester Registry in the SMC.
type Registry struct {
	DeregisteredPeriod *big.Int
	PoolIndex          *big.Int
	Balance            *big.Int
	Deposited          bool
}
