package trieutil

import (
	"math/big"
	"reflect"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/prysmaticlabs/go-ssz"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestMarshalDepositWithProof(t *testing.T) {
	items := [][]byte{
		[]byte("A"),
		[]byte("BB"),
		[]byte("CCC"),
		[]byte("DDDD"),
		[]byte("EEEEE"),
		[]byte("FFFFFF"),
		[]byte("GGGGGGG"),
	}
	m, err := GenerateTrieFromItems(items, 32)
	if err != nil {
		t.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(2)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	if len(proof) != 33 {
		t.Errorf("Received len %d, wanted 33", len(proof))
	}
	someRoot := [32]byte{1, 2, 3, 4}
	someSig := [96]byte{1, 2, 3, 4}
	someKey := [48]byte{1, 2, 3, 4}
	dep := &ethpb.Deposit{
		Proof: proof,
		Data: &ethpb.Deposit_Data{
			PublicKey:             someKey[:],
			WithdrawalCredentials: someRoot[:],
			Amount:                32,
			Signature:             someSig[:],
		},
	}
	enc, err := ssz.Marshal(dep)
	if err != nil {
		t.Fatal(err)
	}
	dec := &ethpb.Deposit{}
	if err := ssz.Unmarshal(enc, &dec); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dec, dep) {
		t.Errorf("Wanted %v, received %v", dep, dec)
	}
}

func TestMerkleTrie_MerkleProofOutOfRange(t *testing.T) {
	h := hashutil.Hash([]byte("hi"))
	m := &MerkleTrie{
		branches: [][][]byte{
			{
				h[:],
			},
			{
				h[:],
			},
			{
				[]byte{},
			},
		},
		depth: 4,
	}
	if _, err := m.MerkleProof(6); err == nil {
		t.Error("Expected out of range failure, received nil", err)
	}
}

func TestMerkleTrieRoot_EmptyTrie(t *testing.T) {
	trie, err := NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not create empty trie %v", err)
	}
	testAccount, err := contracts.Setup()
	if err != nil {
		t.Fatal(err)
	}

	depRoot, err := testAccount.Contract.GetHashTreeRoot(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if depRoot != trie.HashTreeRoot() {
		t.Errorf("Trie root for an empty trie isn't as expected. Expected: %#x but got %#x", depRoot, trie.Root())
	}
}

func TestGenerateTrieFromItems_NoItemsProvided(t *testing.T) {
	if _, err := GenerateTrieFromItems(nil, 32); err == nil {
		t.Error("Expected error when providing nil items received nil")
	}
}

func TestMerkleTrie_VerifyMerkleProof(t *testing.T) {
	items := [][]byte{
		[]byte("A"),
		[]byte("B"),
		[]byte("C"),
		[]byte("D"),
		[]byte("E"),
		[]byte("F"),
		[]byte("G"),
		[]byte("H"),
	}
	m, err := GenerateTrieFromItems(items, 32)
	if err != nil {
		t.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	if len(proof) != 33 {
		t.Errorf("Received len %d, wanted 33", len(proof))
	}
	root := m.Root()
	if ok := VerifyMerkleProof(root[:], items[0], 0, proof); !ok {
		t.Error("First Merkle proof did not verify")
	}
	proof, err = m.MerkleProof(3)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	if ok := VerifyMerkleProof(root[:], items[3], 3, proof); !ok {
		t.Error("Second Merkle proof did not verify")
	}
	if ok := VerifyMerkleProof(root[:], []byte("buzz"), 3, proof); ok {
		t.Error("Item not in tree should fail to verify")
	}
}

func BenchmarkGenerateTrieFromItems(b *testing.B) {
	items := [][]byte{
		[]byte("A"),
		[]byte("BB"),
		[]byte("CCC"),
		[]byte("DDDD"),
		[]byte("EEEEE"),
		[]byte("FFFFFF"),
		[]byte("GGGGGGG"),
	}
	for i := 0; i < b.N; i++ {
		if _, err := GenerateTrieFromItems(items, 32); err != nil {
			b.Fatalf("Could not generate Merkle trie from items: %v", err)
		}
	}
}

func BenchmarkGenerateProof(b *testing.B) {
	b.StopTimer()
	items := [][]byte{
		[]byte("A"),
		[]byte("BB"),
		[]byte("CCC"),
		[]byte("DDDD"),
		[]byte("EEEEE"),
		[]byte("FFFFFF"),
		[]byte("GGGGGGG"),
	}
	normalTrie, err := GenerateTrieFromItems(items, 32)
	if err != nil {
		b.Fatal(err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if _, err := normalTrie.MerkleProof(3); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifyMerkleBranch(b *testing.B) {
	b.StopTimer()
	items := [][]byte{
		[]byte("A"),
		[]byte("BB"),
		[]byte("CCC"),
		[]byte("DDDD"),
		[]byte("EEEEE"),
		[]byte("FFFFFF"),
		[]byte("GGGGGGG"),
	}
	m, err := GenerateTrieFromItems(items, 32)
	if err != nil {
		b.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(2)
	if err != nil {
		b.Fatalf("Could not generate Merkle proof: %v", err)
	}
	root := m.Root()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if ok := VerifyMerkleProof(root[:], items[2], 2, proof); !ok {
			b.Error("Merkle proof did not verify")
		}
	}
}

func TestDepositTrieRoot_OK(t *testing.T) {
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatal(err)
	}

	localTrie, err := NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}

	depRoot, err := testAcc.Contract.GetHashTreeRoot(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}

	if depRoot != localTrie.HashTreeRoot() {
		t.Errorf("Local deposit trie root and contract deposit trie root are not equal. Expected %#x , Got %#x", depRoot, localTrie.Root())
	}

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte

	data := &ethpb.Deposit_Data{
		PublicKey:             pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
		Amount:                big.NewInt(0).Div(contracts.Amount32Eth(), big.NewInt(1e9)).Uint64(), // In Gwei
	}

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	for i := 0; i < 100; i++ {
		copy(data.PublicKey, []byte(strconv.Itoa(i)))
		copy(data.WithdrawalCredentials, []byte(strconv.Itoa(i)))
		copy(data.Signature, []byte(strconv.Itoa(i)))

		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		testAcc.Backend.Commit()
		item, err := ssz.HashTreeRoot(data)
		if err != nil {
			t.Fatal(err)
		}

		err = localTrie.InsertIntoTrie(item[:], i)
		if err != nil {
			t.Error(err)
		}

		depRoot, err = testAcc.Contract.GetHashTreeRoot(&bind.CallOpts{})
		if err != nil {
			t.Fatal(err)
		}

		if depRoot != localTrie.HashTreeRoot() {
			t.Errorf("Local deposit trie root and contract deposit trie root are not equal for index %d. Expected %#x , Got %#x", i, depRoot, localTrie.Root())
		}
	}
}

func TestDepositTrieRoot_Fail(t *testing.T) {
	testAcc, err := contracts.Setup()
	if err != nil {
		t.Fatal(err)
	}

	localTrie, err := NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}

	depRoot, err := testAcc.Contract.GetHashTreeRoot(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}

	if depRoot != localTrie.HashTreeRoot() {
		t.Errorf("Local deposit trie root and contract deposit trie root are not equal. Expected %#x , Got %#x", depRoot, localTrie.Root())
	}

	var pubkey [48]byte
	var withdrawalCreds [32]byte
	var sig [96]byte

	data := &ethpb.Deposit_Data{
		PublicKey:             pubkey[:],
		Signature:             sig[:],
		WithdrawalCredentials: withdrawalCreds[:],
		Amount:                big.NewInt(0).Div(contracts.Amount32Eth(), big.NewInt(1e9)).Uint64(), // In Gwei
	}

	testAcc.TxOpts.Value = contracts.Amount32Eth()
	testAcc.TxOpts.GasLimit = 1000000

	for i := 0; i < 100; i++ {
		copy(data.PublicKey, []byte(strconv.Itoa(i)))
		copy(data.WithdrawalCredentials, []byte(strconv.Itoa(i)))
		copy(data.Signature, []byte(strconv.Itoa(i)))

		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		copy(data.PublicKey, []byte(strconv.Itoa(i+10)))
		copy(data.WithdrawalCredentials, []byte(strconv.Itoa(i+10)))
		copy(data.Signature, []byte(strconv.Itoa(i+10)))

		testAcc.Backend.Commit()
		item, err := ssz.HashTreeRoot(data)
		if err != nil {
			t.Fatal(err)
		}

		err = localTrie.InsertIntoTrie(item[:], i)
		if err != nil {
			t.Error(err)
		}

		depRoot, err = testAcc.Contract.GetHashTreeRoot(&bind.CallOpts{})
		if err != nil {
			t.Fatal(err)
		}

		if depRoot == localTrie.HashTreeRoot() {
			t.Errorf("Local deposit trie root and contract deposit trie root are equal for index %d when they were expected to be not equal", i)
		}
	}
}
