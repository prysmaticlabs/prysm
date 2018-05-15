package sharding

import (
	"math/big"
	//"github.com/ethereum/go-ethereum/rlp"
	//"reflect"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// fieldAccess is to access unexported fields in structs in another package
func fieldAccess(i interface{}, fields []string) reflect.Value {
	val := reflect.ValueOf(i)
	for i := 0; i < len(fields); i++ {
		val = reflect.Indirect(val).FieldByName(fields[i])
	}
	return val

}
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

//TODO: Add test for converting *types.Transaction into raw blobs

//Tests that Transactions can be serialised
func TestSerialize_Deserialize(t *testing.T) {

	tests := []struct {
		transactions []*types.Transaction
	}{
		{
			transactions: []*types.Transaction{
				makeTxWithGasLimit(0),
				makeTxWithGasLimit(5),
				makeTxWithGasLimit(20),
				makeTxWithGasLimit(100),
			},
		},
	}

	for _, tt := range tests {
		c := &Collation{}
		for _, tx := range tt.transactions {
			c.AddTransaction(tx)
		}

		tx := c.transactions

		results, err := c.Serialize()

		if err != nil {
			t.Errorf("Unable to Serialize transactions, %v", err)
		}

		err = c.Deserialize(results)

		if err != nil {
			t.Errorf("Unable to deserialize collation body, %v", err)
		}

		if len(tx) != len(c.transactions) {
			t.Errorf("Transaction length is different before and after serialization: %v, %v", len(tx), len(c.transactions))
		}

		for i := 0; i < len(tx); i++ {

			aval := fieldAccess(tx[i], []string{"data", "AccountNonce"})
			aval2 := fieldAccess(c.transactions[i], []string{"data", "AccountNonce"})

			if aval != aval2 {

				t.Errorf("Data before serialization and after deserialization are not the same: %v, %v", aval, aval2)

			}

			gval := fieldAccess(tx[i], []string{"data", "GasLimit"})
			gval2 := fieldAccess(c.transactions[i], []string{"data", "GasLimit"})
			if gval != gval2 {

				t.Errorf("Data before serialization and after deserialization are not the same: %v, %v", gval, gval2)

			}

			pval := fieldAccess(tx[i], []string{"data", "Price"})
			pval2 := fieldAccess(c.transactions[i], []string{"data", "Price"})
			if pval != pval2 {

				t.Errorf("Data before serialization and after deserialization are not the same: %v, %v", pval, pval2)

			}

			rval := fieldAccess(tx[i], []string{"data", "Recipient"})
			rval2 := fieldAccess(c.transactions[i], []string{"data", "Recipient"})
			if rval != rval2 {

				t.Errorf("Data before serialization and after deserialization are not the same: %v, %v", rval, rval2)

			}

			amval := fieldAccess(tx[i], []string{"data", "Amount"})
			amval2 := fieldAccess(c.transactions[i], []string{"data", "Amount"})
			if amval != amval2 {

				t.Errorf("Data before serialization and after deserialization are not the same: %v, %v", amval, amval2)

			}

			paval := fieldAccess(tx[i], []string{"data", "Payload"})
			paval2 := fieldAccess(c.transactions[i], []string{"data", "Payload"})
			if paval != paval2 {

				t.Errorf("Data before serialization and after deserialization are not the same: %v, %v", paval, paval2)

			}

		}

	}

}

func makeTxWithGasLimit(gl uint64) *types.Transaction {
	return types.NewTransaction(0 /*nonce*/, common.HexToAddress("0x0") /*to*/, nil /*amount*/, gl, nil /*gasPrice*/, nil /*data*/)
}
