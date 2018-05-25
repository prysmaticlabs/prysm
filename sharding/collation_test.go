package sharding

import (
	"bytes"
	"crypto/rand"
	"math/big"
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

	header := NewCollationHeader(big.NewInt(1), nil, big.NewInt(1), nil, []byte{})
	body := []byte{}
	transactions := []*types.Transaction{
		makeTxWithGasLimit(0),
		makeTxWithGasLimit(5),
		makeTxWithGasLimit(20),
		makeTxWithGasLimit(100),
	}

	c := NewCollation(header, body, transactions)

	tx := c.transactions

	results, err := SerializeTxToBlob(tx)

	if err != nil {
		t.Errorf("Unable to Serialize transactions, %v", err)
	}

	deserializedTxs, err := DeserializeBlobToTx(results)

	if err != nil {
		t.Errorf("Unable to deserialize collation body, %v", err)
	}

	if len(tx) != len(*deserializedTxs) {
		t.Errorf("Transaction length is different before and after serialization: %v, %v", len(tx), len(*deserializedTxs))
	}

	for i := 0; i < len(tx); i++ {

		beforeSerialization := tx[i]
		afterDeserialization := (*deserializedTxs)[i]

		if beforeSerialization.Nonce() != afterDeserialization.Nonce() {

			t.Errorf("Data before serialization and after deserialization are not the same ,AccountNonce: %v, %v", beforeSerialization.Nonce(), afterDeserialization.Nonce())

		}

		if beforeSerialization.Gas() != afterDeserialization.Gas() {

			t.Errorf("Data before serialization and after deserialization are not the same ,GasLimit: %v, %v", beforeSerialization.Gas(), afterDeserialization.Gas())

		}

		if beforeSerialization.GasPrice().Cmp(afterDeserialization.GasPrice()) != 0 {

			t.Errorf("Data before serialization and after deserialization are not the same ,Price: %v, %v", beforeSerialization.GasPrice(), afterDeserialization.GasPrice())

		}

		beforeAddress := reflect.ValueOf(beforeSerialization.To())
		afterAddress := reflect.ValueOf(afterDeserialization.To())

		if reflect.DeepEqual(beforeAddress, afterAddress) {

			t.Errorf("Data before serialization and after deserialization are not the same ,Recipient: %v, %v", beforeAddress, afterAddress)

		}

		if beforeSerialization.Value().Cmp(afterDeserialization.Value()) != 0 {

			t.Errorf("Data before serialization and after deserialization are not the same ,Amount: %v, %v", beforeSerialization.Value(), afterDeserialization.Value())

		}

		beforeData := beforeSerialization.Data()
		afterData := afterDeserialization.Data()

		if !bytes.Equal(beforeData, afterData) {

			t.Errorf("Data before serialization and after deserialization are not the same ,Payload: %v, %v", beforeData, afterData)

		}

	}

}

func makeTxWithGasLimit(gl uint64) *types.Transaction {
	return types.NewTransaction(0 /*nonce*/, common.HexToAddress("0x0") /*to*/, nil /*amount*/, gl, nil /*gasPrice*/, nil /*data*/)
}

// BENCHMARK TESTS

// Helper function to generate test that completes round trip serialization tests for a specific number of transactions.
func runBenchTest(b *testing.B, numTransactions int) {
	var txs []*types.Transaction
	for i := 0; i < numTransactions; i++ {
		data := make([]byte, 650)
		rand.Read(data)
		txs = append(txs, types.NewTransaction(0 /*nonce*/, common.HexToAddress("0x0") /*to*/, nil /*amount*/, 0 /*gasLimit*/, nil /*gasPrice*/, data))
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		results, _ := SerializeTxToBlob(txs)
		_, _ = DeserializeBlobToTx(results)
	}

}

func BenchmarkSerialization10(b *testing.B) {
	runBenchTest(b, 10)
}

func BenchmarkSerialization100(b *testing.B) {
	runBenchTest(b, 100)
}

func BenchmarkSerialization1000(b *testing.B) {
	runBenchTest(b, 1000)
}

func BenchmarkSerialization10000(b *testing.B) {
	runBenchTest(b, 10000)
}
