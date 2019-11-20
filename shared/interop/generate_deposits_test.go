package interop_test

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/interop"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestSPMPDeposits(t *testing.T) {
	// Confirm that internal single-processor deposits end up with the same results as the multi-processor version

	numDeposits := 557

	cfg := &featureconfig.Flags{
		Scatter: false,
	}
	featureconfig.Init(cfg)
	spPrivKeys, spPubKeys, err := interop.DeterministicallyGenerateKeys(0 /*startIndex*/, uint64(numDeposits))
	if err != nil {
		t.Fatal(err)
	}
	spDepositDataItems, spDepositDataRoots, err := interop.DepositDataFromKeys(spPrivKeys, spPubKeys)
	if err != nil {
		t.Fatal(err)
	}
	trie, err := trieutil.GenerateTrieFromItems(
		spDepositDataRoots,
		int(params.BeaconConfig().DepositContractTreeDepth),
	)
	if err != nil {
		t.Fatal(err)
	}
	spDeposits, err := interop.GenerateDepositsFromData(spDepositDataItems, trie)
	if err != nil {
		t.Fatal(err)
	}

	cfg = &featureconfig.Flags{
		Scatter: true,
	}
	featureconfig.Init(cfg)
	privKeys, pubKeys, err := interop.DeterministicallyGenerateKeys(0 /*startIndex*/, uint64(numDeposits))
	if err != nil {
		t.Fatal(err)
	}
	depositDataItems, _, err := interop.DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		t.Fatal(err)
	}
	deposits, err := interop.GenerateDepositsFromData(depositDataItems, trie)
	if err != nil {
		t.Fatal(err)
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
