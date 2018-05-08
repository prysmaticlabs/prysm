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

func TestCollation_ProposerAddress(t *testing.T) {
	proposerAddr := common.BytesToAddress([]byte("proposer"))
	header := NewCollationHeader(big.NewInt(1), nil, big.NewInt(1), &proposerAddr, []byte{})
	body := []byte{}

	collation := NewCollation(header, body, nil)

	if collation.ProposerAddress().String() != proposerAddr.String() {
		t.Errorf("initialized collation does not contain correct proposer address")
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
