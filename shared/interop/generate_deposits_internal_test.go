package interop

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestSPMPDeposits(t *testing.T) {
	// Confirm that internal single-processor deposits end up with the same results as the multi-processor version

	numDeposits := 557

	spPrivKeys, spPubKeys, err := deterministicallyGenerateKeys(0 /*startIndex*/, uint64(numDeposits))
	if err != nil {
		t.Fatal(err)
	}
	if len(spPrivKeys) != numDeposits {
		t.Fatalf("Incorrect number of SP private keys: expected %d, received %d", numDeposits, len(spPrivKeys))
	}
	if len(spPubKeys) != numDeposits {
		t.Fatalf("Incorrect number of SP public keys: expected %d, received %d", numDeposits, len(spPubKeys))
	}

	privKeys, pubKeys, err := DeterministicallyGenerateKeys(0 /*startIndex*/, uint64(numDeposits))
	if err != nil {
		t.Fatal(err)
	}
	if len(privKeys) != numDeposits {
		t.Fatalf("Incorrect number of private keys: expected %d, received %d", numDeposits, len(privKeys))
	}
	if len(pubKeys) != numDeposits {
		t.Fatalf("Incorrect number of public keys: expected %d, received %d", numDeposits, len(pubKeys))
	}

	for i := range privKeys {
		if bytes.Compare(privKeys[i].Marshal(), spPrivKeys[i].Marshal()) != 0 {
			t.Fatalf("Private key mismatch at index %d: %x vs %x", i, spPrivKeys[i].Marshal(), privKeys[i].Marshal())
		}
		if bytes.Compare(pubKeys[i].Marshal(), spPubKeys[i].Marshal()) != 0 {
			t.Fatalf("Public key mismatch at index %d: %x vs %x", i, spPubKeys[i].Marshal(), pubKeys[i].Marshal())
		}
	}

	spDepositDataItems, spDepositDataRoots, err := depositDataFromKeys(spPrivKeys, spPubKeys)
	if err != nil {
		t.Fatal(err)
	}
	if len(spDepositDataItems) != numDeposits {
		t.Fatalf("Incorrect number of SP deposit data items: expected %d, received %d", numDeposits, len(spDepositDataItems))
	}
	if len(spDepositDataRoots) != numDeposits {
		t.Fatalf("Incorrect number of SP deposit data roots: expected %d, received %d", numDeposits, len(spDepositDataRoots))
	}

	depositDataItems, depositDataRoots, err := DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		t.Fatal(err)
	}
	if len(depositDataItems) != numDeposits {
		t.Fatalf("Incorrect number of deposit data items: expected %d, received %d", numDeposits, len(depositDataItems))
	}
	if len(depositDataRoots) != numDeposits {
		t.Fatalf("Incorrect number of deposit data roots: expected %d, received %d", numDeposits, len(depositDataRoots))
	}

	for i := range depositDataItems {
		spDDI, err := json.Marshal(spDepositDataItems[i])
		if err != nil {
			t.Fatal(err)
		}
		ddi, err := json.Marshal(depositDataItems[i])
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Compare(spDDI, ddi) != 0 {
			t.Fatalf("Deposit data mismatch at index %d: %v vs %v", i, spDDI, ddi)
		}
		spDDR, err := json.Marshal(spDepositDataRoots[i])
		if err != nil {
			t.Fatal(err)
		}
		ddr, err := json.Marshal(depositDataRoots[i])
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Compare(spDDR, ddr) != 0 {
			t.Fatalf("Deposit data root mismatch at index %d: %v vs %v", i, spDDR, ddr)
		}
	}

	trie, err := trieutil.GenerateTrieFromItems(
		depositDataRoots,
		int(params.BeaconConfig().DepositContractTreeDepth),
	)
	if err != nil {
		t.Fatal(err)
	}

	spDeposits, err := generateDepositsFromData(spDepositDataItems, 0, trie)
	if err != nil {
		t.Fatal(err)
	}
	if len(spDeposits) != numDeposits {
		t.Fatalf("Incorrect number of SP deposits: expected %d, received %d", numDeposits, len(spDeposits))
	}

	deposits, err := GenerateDepositsFromData(depositDataItems, trie)
	if err != nil {
		t.Fatal(err)
	}
	if len(deposits) != numDeposits {
		t.Fatalf("Incorrect number of deposits: expected %d, received %d", numDeposits, len(deposits))
	}

	for i := range deposits {
		spD, err := spDeposits[i].Marshal()
		if err != nil {
			t.Fatal(err)
		}
		d, err := deposits[i].Marshal()
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Compare(d, spD) != 0 {
			t.Fatalf("Deposit mismatch at index %d: %v vs %v", i, spD, d)
		}
	}

}
