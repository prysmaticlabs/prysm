package sharding

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestCollation_Transactions(t *testing.T) {
	header := NewCollationHeader(big.NewInt(1), nil, big.NewInt(1), nil, []byte{})
	body := []byte{}
	transactions := []*types.Transaction{
		makeTxWithGasLimit(0),
		makeTxWithGasLimit(1),
		makeTxWithGasLimit(2),
		makeTxWithGasLimit(3),
	}

	collation := NewCollation(header, body, transactions)

	for i, tx := range collation.Transactions() {
		if tx.Hash().String() != transactions[i].Hash().String() {
			t.Errorf("initialized collation struct does not contain correct transactions")
		}
	}
}

func TestSerialize(t *testing.T) {
	tests := []struct {
		transactions []*types.Transaction
	}{
		{
			transactions: []*types.Transaction{
				makeTxWithGasLimit(0),
				makeTxWithGasLimit(1),
				makeTxWithGasLimit(2),
				makeTxWithGasLimit(3),
			},
		}, {
			transactions: []*types.Transaction{},
		},
	}

	for _, tt := range tests {
		c := &Collation{}
		for _, tx := range tt.transactions {
			c.AddTransaction(tx)
		}
		/*
			results, err := c.Serialize()
			if err != nil {
				t.Fatalf("%v ----%v---%v", err, results, c.transactions)
			}
		*/

	}

}

func makeTxWithGasLimit(gl uint64) *types.Transaction {
	return types.NewTransaction(0 /*nonce*/, common.HexToAddress("0x0") /*to*/, nil /*amount*/, gl, nil /*gasPrice*/, nil /*data*/)
}
