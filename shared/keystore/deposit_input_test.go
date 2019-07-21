package keystore_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestDepositInput_GeneratesPb(t *testing.T) {
	k1, err := keystore.NewKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := keystore.NewKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	result, err := keystore.DepositInput(k1, k2, 0)
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

	dom := bytesutil.FromBytes4(params.BeaconConfig().DomainDeposit)
	if !sig.Verify(sr[:], k1.PublicKey, dom) {
		t.Error("Invalid proof of deposit input signature")
	}
}
