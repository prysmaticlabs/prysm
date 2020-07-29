package depositcontract_test

import (
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	depositcontract "github.com/prysmaticlabs/prysm/contracts/deposit-contract"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestDepositTrieRoot_OK(t *testing.T) {
	testAcc, err := depositcontract.Setup()
	if err != nil {
		t.Fatal(err)
	}

	localTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}

	depRoot, err := testAcc.Contract.GetDepositRoot(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}

	if depRoot != localTrie.HashTreeRoot() {
		t.Errorf("Local deposit trie root and contract deposit trie root are not equal. Expected %#x , Got %#x", depRoot, localTrie.Root())
	}

	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0 /*startIndex*/, 101)
	if err != nil {
		t.Fatal(err)
	}
	depositDataItems, depositDataRoots, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		t.Fatal(err)
	}

	testAcc.TxOpts.Value = depositcontract.Amount32Eth()

	for i := 0; i < 100; i++ {
		data := depositDataItems[i]
		dataRoot := [32]byte{}
		copy(dataRoot[:], depositDataRoots[i])

		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, dataRoot); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		testAcc.Backend.Commit()
		item, err := data.HashTreeRoot()
		if err != nil {
			t.Fatal(err)
		}

		localTrie.Insert(item[:], i)
		depRoot, err = testAcc.Contract.GetDepositRoot(&bind.CallOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if depRoot != localTrie.HashTreeRoot() {
			t.Errorf("Local deposit trie root and contract deposit trie root are not equal for index %d. Expected %#x , Got %#x", i, depRoot, localTrie.Root())
		}
	}
}

func TestDepositTrieRoot_Fail(t *testing.T) {
	testAcc, err := depositcontract.Setup()
	if err != nil {
		t.Fatal(err)
	}

	localTrie, err := trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}

	depRoot, err := testAcc.Contract.GetDepositRoot(&bind.CallOpts{})
	if err != nil {
		t.Fatal(err)
	}

	if depRoot != localTrie.HashTreeRoot() {
		t.Errorf("Local deposit trie root and contract deposit trie root are not equal. Expected %#x , Got %#x", depRoot, localTrie.Root())
	}

	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0 /*startIndex*/, 101)
	if err != nil {
		t.Fatal(err)
	}
	depositDataItems, depositDataRoots, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		t.Fatal(err)
	}
	testAcc.TxOpts.Value = depositcontract.Amount32Eth()

	for i := 0; i < 100; i++ {
		data := depositDataItems[i]
		dataRoot := [32]byte{}
		copy(dataRoot[:], depositDataRoots[i])

		if _, err := testAcc.Contract.Deposit(testAcc.TxOpts, data.PublicKey, data.WithdrawalCredentials, data.Signature, dataRoot); err != nil {
			t.Fatalf("Could not deposit to deposit contract %v", err)
		}

		// Change an element in the data when storing locally
		copy(data.PublicKey, strconv.Itoa(i+10))

		testAcc.Backend.Commit()
		item, err := data.HashTreeRoot()
		if err != nil {
			t.Fatal(err)
		}

		localTrie.Insert(item[:], i)

		depRoot, err = testAcc.Contract.GetDepositRoot(&bind.CallOpts{})
		if err != nil {
			t.Fatal(err)
		}

		if depRoot == localTrie.HashTreeRoot() {
			t.Errorf("Local deposit trie root and contract deposit trie root are equal for index %d when they were expected to be not equal", i)
		}
	}
}
