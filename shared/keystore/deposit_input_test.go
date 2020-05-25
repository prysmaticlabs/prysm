package keystore_test

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestDepositInput_GeneratesPb(t *testing.T) {
	t.Skip("To be resolved until 5119 gets in")
	k1, err := keystore.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	k2, err := keystore.NewKey()
	if err != nil {
		t.Fatal(err)
	}

	result, _, err := keystore.DepositInput(k1, k2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.PublicKey, k1.PublicKey.Marshal()) {
		t.Errorf("Mismatched pubkeys in deposit input. Want = %x, got = %x", result.PublicKey, k1.PublicKey.Marshal())
	}

	sig, err := bls.SignatureFromBytes(result.Signature)
	if err != nil {
		t.Fatal(err)
	}

	sr, err := ssz.SigningRoot(result)
	if err != nil {
		t.Fatal(err)
	}
	dom := params.BeaconConfig().DomainDeposit
	root, err := ssz.HashTreeRoot(&pb.SigningRoot{ObjectRoot: sr[:], Domain: dom[:]})
	if err != nil {
		t.Fatal(err)
	}
	if !sig.Verify(k1.PublicKey, root[:]) {
		t.Error("Invalid proof of deposit input signature")
	}
}
