package trieutil

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	contracts "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	amount33Eth, _        = new(big.Int).SetString("33000000000000000000", 10)
	amount32Eth, _        = new(big.Int).SetString("32000000000000000000", 10)
	amountLessThan1Eth, _ = new(big.Int).SetString("500000000000000000", 10)
)

type testAccount struct {
	addr         common.Address
	contract     *contracts.DepositContract
	contractAddr common.Address
	backend      *backends.SimulatedBackend
	txOpts       *bind.TransactOpts
}

func setup() (*testAccount, error) {
	genesis := make(core.GenesisAlloc)
	privKey, _ := crypto.GenerateKey()
	pubKeyECDSA, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	// strip off the 0x and the first 2 characters 04 which is always the EC prefix and is not required.
	publicKeyBytes := crypto.FromECDSAPub(pubKeyECDSA)[4:]
	var pubKey = make([]byte, 48)
	copy(pubKey[:], []byte(publicKeyBytes))

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	txOpts := bind.NewKeyedTransactor(privKey)
	startingBalance, _ := new(big.Int).SetString("100000000000000000000000000000000000000", 10)
	genesis[addr] = core.GenesisAccount{Balance: startingBalance}
	backend := backends.NewSimulatedBackend(genesis, 210000000000)

	depositsRequired := big.NewInt(8)
	minDeposit := big.NewInt(1e9)
	maxDeposit := big.NewInt(32e9)
	contractAddr, _, contract, err := contracts.DeployDepositContract(txOpts, backend, depositsRequired, minDeposit, maxDeposit, big.NewInt(1), addr)
	if err != nil {
		return nil, err
	}
	backend.Commit()

	return &testAccount{addr, contract, contractAddr, backend, txOpts}, nil
}

func TestMerkleTrie_BranchIndices(t *testing.T) {
	indices := BranchIndices(1024, 3 /* depth */)
	expected := []int{1024, 512, 256}
	for i := 0; i < len(indices); i++ {
		if expected[i] != indices[i] {
			t.Errorf("Expected %d, received %d", expected[i], indices[i])
		}
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
	}
	if _, err := m.MerkleProof(-1); err == nil {
		t.Error("Expected out of range failure, received nil", err)
	}
	if _, err := m.MerkleProof(2); err == nil {
		t.Error("Expected out of range failure, received nil", err)
	}
	if _, err := m.MerkleProof(0); err == nil {
		t.Error("Expected out of range failure, received nil", err)
	}
}

func TestMerkleTrieRoot_EmptyTrie(t *testing.T) {
	trie, err := NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatalf("Could not create empty trie %v", err)
	}

	var zeroByte [32]byte
	rootHash := zeroByte

	for i := 1; i < int(params.BeaconConfig().DepositContractTreeDepth); i++ {
		newRoot := parentHash(rootHash[:], rootHash[:])
		rootHash = bytesutil.ToBytes32(newRoot)
	}
	if rootHash != trie.Root() {
		t.Errorf("Trie root for an empty trie isn't as expected. Expected: %#x but got %#x", rootHash, trie.Root())
	}

}

func TestGenerateTrieFromItems_NoItemsProvided(t *testing.T) {
	if _, err := GenerateTrieFromItems(nil, 32); err == nil {
		t.Error("Expected error when providing nil items received nil")
	}
}

func TestMerkleTrie_VerifyMerkleProof(t *testing.T) {
	items := [][]byte{
		[]byte("short"),
		[]byte("eos"),
		[]byte("long"),
		[]byte("eth"),
		[]byte("4ever"),
		[]byte("eth2"),
		[]byte("moon"),
	}
	m, err := GenerateTrieFromItems(items, 32)
	if err != nil {
		t.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(2)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	root := m.Root()
	if ok := VerifyMerkleProof(root[:], items[2], 2, proof); !ok {
		t.Error("Merkle proof did not verify")
	}
	proof, err = m.MerkleProof(3)
	if err != nil {
		t.Fatalf("Could not generate Merkle proof: %v", err)
	}
	if ok := VerifyMerkleProof(root[:], items[3], 3, proof); !ok {
		t.Error("Merkle proof did not verify")
	}
	if ok := VerifyMerkleProof(root[:], []byte("btc"), 3, proof); ok {
		t.Error("Item not in tree should fail to verify")
	}
}

func BenchmarkGenerateTrieFromItems(b *testing.B) {
	items := [][]byte{
		[]byte("short"),
		[]byte("eos"),
		[]byte("long"),
		[]byte("eth"),
		[]byte("4ever"),
		[]byte("eth2"),
		[]byte("moon"),
	}
	for i := 0; i < b.N; i++ {
		if _, err := GenerateTrieFromItems(items, 32); err != nil {
			b.Fatalf("Could not generate Merkle trie from items: %v", err)
		}
	}
}

func BenchmarkVerifyMerkleBranch(b *testing.B) {
	items := [][]byte{
		[]byte("short"),
		[]byte("eos"),
		[]byte("long"),
		[]byte("eth"),
		[]byte("4ever"),
		[]byte("eth2"),
		[]byte("moon"),
	}
	m, err := GenerateTrieFromItems(items, 32)
	if err != nil {
		b.Fatalf("Could not generate Merkle trie from items: %v", err)
	}
	proof, err := m.MerkleProof(2)
	if err != nil {
		b.Fatalf("Could not generate Merkle proof: %v", err)
	}
	for i := 0; i < b.N; i++ {
		if ok := VerifyMerkleProof(m.branches[0][0], items[2], 2, proof); !ok {
			b.Error("Merkle proof did not verify")
		}
	}
}

func TestDepositTrieRoot_OK(t *testing.T) {
	testAccount, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	localTrie, err := NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}

	depRoot, err := testAccount.contract.GetDepositRoot(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}

	if depRoot != localTrie.Root() {
		t.Errorf("Local deposit trie root and contract deposit trie root are not equal. Expected %#x , Got %#x", depRoot, localTrie.Root())
	}
}
