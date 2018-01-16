package sharding

import (
	//"flag"
	//"fmt"
	//"math/rand"
	//"os"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

var (
	txSimple = NewShardingTransaction(
		0,
		common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
		// amount
		big.NewInt(10),
		// gasLimit
		1000000,
		// gasPrice
		big.NewInt(1),
		// data
		common.FromHex("hello world this is the data"),
		// access list
		[]common.Address{common.HexToAddress("032e7baea6a6c7c4c2dfe98392932326af552d87"), common.HexToAddress("083e7baea6a6c7c4c2dfeb97710293843f552d87")},
	)
)

func TestCreation(t *testing.T) {
	if txSimple.ChainID().Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("ChainID invalid")
	}
	if txSimple.ShardID().Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("ShardID invalid")
	}
}
