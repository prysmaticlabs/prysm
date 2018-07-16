package legacyutil

import (

	"math/big"


	"github.com/ethereum/go-ethereum/common"
	pb "github.com/prysmaticlabs/geth-sharding/sharding/p2p/proto/v1"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
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