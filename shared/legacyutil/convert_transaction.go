// Package legacyutil exists to convert Ethereum Serenity types to go-ethereum or
// Ethereum 1.0 types.
package legacyutil

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
)

// TransformTransaction of proto transaction to geth's transction.
func TransformTransaction(t *pb.Transaction) *gethTypes.Transaction {
	return gethTypes.NewTransaction(
		t.Nonce,
		common.BytesToAddress(t.Recipient),
		big.NewInt(0).SetUint64(t.Value),
		t.GasLimit,
		big.NewInt(0).SetUint64(t.GasPrice),
		t.Input,
	)
}
