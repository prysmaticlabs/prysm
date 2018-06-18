package mainchain

import (
	"testing"

	"github.com/ethereum/go-ethereum/sharding"
)

// Verifies that SMCCLient implements the sharding Service inteface.
var _ = sharding.Service(&SMCClient{})

func TestWaitForTransaction(t *testing.T) {
	_ = &SMCClient{}

}
