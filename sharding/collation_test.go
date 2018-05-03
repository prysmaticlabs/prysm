package sharding

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TODO: this test needs to change as we will be serializing tx's into blobs
// within the collation body instead.

// func TestCollation_AddTransactions(t *testing.T) {
// 	tests := []struct {
// 		transactions []*types.Transaction
// 	}{
// 		{
// 			transactions: []*types.Transaction{
// 				makeTxWithGasLimit(0),
// 				makeTxWithGasLimit(1),
// 				makeTxWithGasLimit(2),
// 				makeTxWithGasLimit(3),
// 			},
// 		}, {
// 			transactions: []*types.Transaction{},
// 		},
// 	}

// 	for _, tt := range tests {
// 		c := &Collation{}
// 		for _, tx := range tt.transactions {
// 			c.AddTransaction(tx)
// 		}
// 		results := c.Transactions()
// 		if len(results) != len(tt.transactions) {
// 			t.Fatalf("Wrong number of transactions. want=%d. got=%d", len(tt.transactions), len(results))
// 		}
// 		for i, tx := range tt.transactions {
// 			if results[i] != tx {
// 				t.Fatalf("Mismatched transactions. wanted=%+v. got=%+v", tt.transactions, results)
// 			}
// 		}
// 	}
// }

func makeTxWithGasLimit(gl uint64) *types.Transaction {
	return types.NewTransaction(0 /*nonce*/, common.HexToAddress("0x0") /*to*/, nil /*amount*/, gl, nil /*gasPrice*/, nil /*data*/)
}
