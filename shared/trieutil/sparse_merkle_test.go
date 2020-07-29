package trieutil

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
	m, err := GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(2)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	if len(proof) != int(params.BeaconConfig().DepositContractTreeDepth)+1 {
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
	enc, err := dep.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	dec := &ethpb.Deposit{}
	if err := dec.UnmarshalSSZ(enc); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dec, dep) {
		t.Errorf("Wanted %v, received %v", dep, dec)
	}
}

func TestMerkleTrie_MerkleProofOutOfRange(t *testing.T) {
	h := hashutil.Hash([]byte("hi"))
	m := &SparseMerkleTrie{
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

	depRoot, err := testAccount.Contract.GetDepositRoot(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if depRoot != trie.HashTreeRoot() {
		t.Errorf("Trie root for an empty trie isn't as expected. Expected: %#x but got %#x", depRoot, trie.Root())
	}
}

func TestGenerateTrieFromItems_NoItemsProvided(t *testing.T) {
	if _, err := GenerateTrieFromItems(nil, int(params.BeaconConfig().DepositContractTreeDepth)); err == nil {
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
	m, err := GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	if len(proof) != int(params.BeaconConfig().DepositContractTreeDepth)+1 {
		t.Errorf("Received len %d, wanted 33", len(proof))
	}
	root := m.Root()
	if ok := VerifyMerkleBranch(root[:], items[0], 0, proof); !ok {
		t.Error("First Merkle proof did not verify")
	}
	proof, err = m.MerkleProof(3)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	if ok := VerifyMerkleBranch(root[:], items[3], 3, proof); !ok {
		t.Error("Second Merkle proof did not verify")
	}
	if ok := VerifyMerkleBranch(root[:], []byte("buzz"), 3, proof); ok {
		t.Error("Item not in tree should fail to verify")
	}
}

func TestMerkleTrie_VerifyMerkleProof_TrieUpdated(t *testing.T) {
	items := [][]byte{
		{1},
		{2},
		{3},
		{4},
	}
	m, err := GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(0)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	root := m.Root()
	if ok := VerifyMerkleBranch(root[:], items[0], 0, proof); !ok {
		t.Error("First Merkle proof did not verify")
	}

	// Now we update the trie.
	m.Insert([]byte{5}, 3)
	proof, err = m.MerkleProof(3)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	root = m.Root()
	if ok := VerifyMerkleBranch(root[:], []byte{5}, 3, proof); !ok {
		t.Error("Second Merkle proof did not verify")
	}
	if ok := VerifyMerkleBranch(root[:], []byte{4}, 3, proof); ok {
		t.Error("Old item should not verify")
	}

	// Now we update the trie at an index larger than the number of items.
	m.Insert([]byte{6}, 15)
}

func TestRoundtripProto_OK(t *testing.T) {
	items := [][]byte{
		{1},
		{2},
		{3},
		{4},
	}
	m, err := GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth)+1)
	if err != nil {
		t.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	protoTrie := m.ToProto()
	depositRoot := m.HashTreeRoot()

	newTrie := CreateTrieFromProto(protoTrie)

	if newTrie.HashTreeRoot() != depositRoot {
		t.Errorf("Wanted a deposit trie root of %#x but got %#x", depositRoot, newTrie.HashTreeRoot())
	}
}

func TestCopy_OK(t *testing.T) {
	items := [][]byte{
		{1},
		{2},
		{3},
		{4},
	}
	source, err := GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth)+1)
	if err != nil {
		t.Fatalf("Could not generate Merkle trie from items: %v", err)
	}

	copy := source.Copy()

	if copy == source {
		t.Errorf("Original trie returned.")
	}
	sourceHash := source.HashTreeRoot()
	copyHash := copy.HashTreeRoot()
	if sourceHash != copyHash {
		t.Errorf("Trie not copied correctly. Got root hash %x vs expected %x", copyHash, sourceHash)
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
		if _, err := GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth)); err != nil {
			b.Fatalf("Could not generate Merkle trie from items: %v", err)
		}
	}
}

func BenchmarkInsertTrie_Optimized(b *testing.B) {
	b.StopTimer()
	numDeposits := 16000
	items := make([][]byte, numDeposits)
	for i := 0; i < numDeposits; i++ {
		someRoot := bytesutil.ToBytes32([]byte(strconv.Itoa(i)))
		items[i] = someRoot[:]
	}
	tr, err := GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		b.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	someItem := bytesutil.ToBytes32([]byte("hello-world"))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tr.Insert(someItem[:], i%numDeposits)
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
	normalTrie, err := GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth))
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
	m, err := GenerateTrieFromItems(items, int(params.BeaconConfig().DepositContractTreeDepth))
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
		if ok := VerifyMerkleBranch(root[:], items[2], 2, proof); !ok {
			b.Error("Merkle proof did not verify")
		}
	}
}
