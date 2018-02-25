package sharding

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestCollation_AddTransactions(t *testing.T) {
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
		results := c.Transactions()
		if len(results) != len(tt.transactions) {
			t.Fatalf("Wrong number of transactions. want=%d. got=%d", len(tt.transactions), len(results))
		}
		for i, tx := range tt.transactions {
			if results[i] != tx {
				t.Fatalf("Mismatched transactions. wanted=%+v. got=%+v", tt.transactions, results)
			}
		}
	}
}

func TestCollation_GasUsed(t *testing.T) {
	tests := []struct {
		transactions []*types.Transaction
		gasUsed      *big.Int
	}{
		{
			transactions: []*types.Transaction{
				makeTxWithGasLimit(100),
				makeTxWithGasLimit(100000),
				makeTxWithGasLimit(899900),
			},
			gasUsed: big.NewInt(1000000),
		}, {
			transactions: []*types.Transaction{},
			gasUsed:      big.NewInt(0),
		},
		{
			transactions: []*types.Transaction{
				makeTxWithGasLimit(math.MaxUint64),
				makeTxWithGasLimit(9001),
				makeTxWithGasLimit(math.MaxUint64),
			},
			gasUsed: big.NewInt(0).SetUint64(math.MaxUint64),
		},
	}

	for _, tt := range tests {
		got := (&Collation{transactions: tt.transactions}).GasUsed()
		if tt.gasUsed.Cmp(got) != 0 {
			t.Errorf("Returned unexpected gasUsed. Got=%v, wanted=%v", got, tt.gasUsed)
		}
	}
}

func makeTxWithGasLimit(gl uint64) *types.Transaction {
	return types.NewTransaction(0 /*nonce*/, common.HexToAddress("0x0") /*to*/, nil /*amount*/, gl, nil /*gasPrice*/, nil /*data*/)
}
