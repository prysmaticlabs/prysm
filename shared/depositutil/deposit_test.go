package depositutil_test

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestDepositInput_GeneratesPb(t *testing.T) {
	k1 := bls.RandKey()
	k2 := bls.RandKey()

	result, _, err := depositutil.DepositInput(k1, k2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.PublicKey, k1.PublicKey().Marshal()) {
		t.Errorf(
			"Mismatched pubkeys in deposit input. Want = %x, got = %x",
			result.PublicKey,
			k1.PublicKey().Marshal(),
		)
	}

	sig, err := bls.SignatureFromBytes(result.Signature)
	if err != nil {
		t.Fatal(err)
	}
	sr, err := ssz.SigningRoot(result)
	if err != nil {
		t.Fatal(err)
	}
	domain, err := helpers.ComputeDomain(
		params.BeaconConfig().DomainDeposit,
		nil, /*forkVersion*/
		nil, /*genesisValidatorsRoot*/
	)
	if err != nil {
		t.Fatal(err)
	}
	root, err := ssz.HashTreeRoot(&pb.SigningData{ObjectRoot: sr[:], Domain: domain[:]})
	if err != nil {
		t.Fatal(err)
	}
	if !sig.Verify(k1.PublicKey(), root[:]) {
		t.Error("Invalid proof of deposit input signature")
	}
}

func TestVerifyDepositSignature_ValidSig(t *testing.T) {
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatalf("Error Generating Deposits and Keys - %v", err)
	}
	deposit := deposits[0]
	err = depositutil.VerifyDepositSignature(deposit.Data)
	if err != nil {
		t.Fatal("Deposit Verification fails with a valid signature")
	}
}

func TestVerifyDepositSignature_InvalidSig(t *testing.T) {
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatalf("Error Generating Deposits and Keys - %v", err)
	}
	deposit := deposits[0]
	deposit.Data.Signature = deposit.Data.Signature[1:]
	err = depositutil.VerifyDepositSignature(deposit.Data)
	if err == nil {
		t.Fatal("Deposit Verification succeeds with a invalid signature")
	}
}
