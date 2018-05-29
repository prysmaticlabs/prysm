package sharding

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/sharding/utils"
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
func makeRandomTransactions(numTransactions int) []*types.Transaction {
	var txs []*types.Transaction
	for i := 0; i < numTransactions; i++ {
		data := make([]byte, 650)
		rand.Read(data)
		txs = append(txs, types.NewTransaction(0 /*nonce*/, common.HexToAddress("0x0") /*to*/, nil /*amount*/, 0 /*gasLimit*/, nil /*gasPrice*/, data))
	}

	return txs
}

func runSerializeRoundtrip(b *testing.B, numTransactions int) {
	txs := makeRandomTransactions(numTransactions)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		blob, err := SerializeTxToBlob(txs)
		if err != nil {
			b.Errorf("SerializeTxToBlob failed: %v", err)
		}

		_, err = DeserializeBlobToTx(blob)
		if err != nil {
			b.Errorf("DeserializeBlobToTx failed: %v", err)
		}
	}
}

func runSerializeBenchmark(b *testing.B, numTransactions int) {
	txs := makeRandomTransactions(numTransactions)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := SerializeTxToBlob(txs)
		if err != nil {
			b.Errorf("SerializeTxToBlob failed: %v", err)
		}
	}
}

func runSerializeBlobOnlyBenchmark(b *testing.B, numTransactions int) {
	txs := makeRandomTransactions(numTransactions)
	blobs, err := SerializeTxToRawBlob(txs)
	if err != nil {
		b.Errorf("SerializeTxToRawBlock failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := utils.Serialize(blobs)
		if err != nil {
			b.Errorf("utils.Serialize failed: %v", err)
		}
	}
}

func runDeserializeBenchmark(b *testing.B, numTransactions int) {
	txs := makeRandomTransactions(numTransactions)
	blob, err := SerializeTxToBlob(txs)
	if err != nil {
		b.Errorf("SerializeTxToRawBlock failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := DeserializeBlobToTx(blob)
		if err != nil {
			b.Errorf("DeserializeBlobToTx failed: %v", err)
		}
	}
}

func runDeserializeBlobOnlyBenchmark(b *testing.B, numTransactions int) {
	txs := makeRandomTransactions(numTransactions)
	blob, err := SerializeTxToBlob(txs)
	if err != nil {
		b.Errorf("SerializeTxToBlob failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := utils.Deserialize(blob)
		if err != nil {
			b.Errorf("utils.Deserialize failed: %v", err)
		}
	}
}

func BenchmarkSerializeBlobOnly10(b *testing.B) {
	runSerializeBlobOnlyBenchmark(b, 10)
}

func BenchmarkSerializeBlobOnly100(b *testing.B) {
	runSerializeBlobOnlyBenchmark(b, 100)
}

func BenchmarkSerializeBlobOnly1000(b *testing.B) {
	runSerializeBlobOnlyBenchmark(b, 1000)
}

func BenchmarkSerialize10(b *testing.B) {
	runSerializeBenchmark(b, 10)
}

func BenchmarkSerialize100(b *testing.B) {
	runSerializeBenchmark(b, 100)
}

func BenchmarkSerialize1000(b *testing.B) {
	runSerializeBenchmark(b, 1000)
}

func BenchmarkDeserialize10(b *testing.B) {
	runDeserializeBenchmark(b, 10)
}

func BenchmarkDeserialize100(b *testing.B) {
	runDeserializeBenchmark(b, 100)
}

func BenchmarkDeserialize1000(b *testing.B) {
	runDeserializeBenchmark(b, 1000)
}

func BenchmarkDeserializeBlobOnly10(b *testing.B) {
	runDeserializeBlobOnlyBenchmark(b, 10)
}

func BenchmarkDeserializeBlobOnly100(b *testing.B) {
	runDeserializeBlobOnlyBenchmark(b, 100)
}

func BenchmarkDeserializeBlobOnly1000(b *testing.B) {
	runDeserializeBlobOnlyBenchmark(b, 1000)
}

func BenchmarkSerializeRoundtrip10(b *testing.B) {
	runSerializeRoundtrip(b, 10)
}

func BenchmarkSerializeRoundtrip100(b *testing.B) {
	runSerializeRoundtrip(b, 100)
}

func BenchmarkSerializeRoundtrip1000(b *testing.B) {
	runSerializeRoundtrip(b, 1000)
}
