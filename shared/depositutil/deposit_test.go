package depositutil_test

import (
	"bytes"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	cfg := params.BeaconConfig()
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			Amount:                cfg.MinDepositAmount,
			WithdrawalCredentials: []byte("testing"),
		},
	}

	sk := bls.RandKey()
	deposit.Data.PublicKey = sk.PublicKey().Marshal()
	d, err := helpers.ComputeDomain(
		cfg.DomainDeposit,
		cfg.GenesisForkVersion,
		cfg.ZeroHash[:],
	)
	if err != nil {
		t.Fatal(err)
	}
	signedRoot, err := helpers.ComputeSigningRoot(deposit.Data, d)
	if err != nil {
		t.Fatal(err)
	}
	sig := sk.Sign(signedRoot[:])
	deposit.Data.Signature = sig.Marshal()

	err = depositutil.VerifyDepositSignature(deposit.Data)
	if err != nil {
		t.Fatal("Deposit Signature Verification fails with a valid signature")
	}
}

func TestVerifyDepositSignature_InValidSig(t *testing.T) {
	cfg := params.BeaconConfig()
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			Amount:                cfg.MinDepositAmount,
			WithdrawalCredentials: []byte("testing"),
		},
	}

	sk := bls.RandKey()
	deposit.Data.PublicKey = sk.PublicKey().Marshal()
	d, err := helpers.ComputeDomain(
		cfg.DomainDeposit,
		cfg.GenesisForkVersion,
		cfg.ZeroHash[:],
	)
	if err != nil {
		t.Fatal(err)
	}
	signedRoot, err := helpers.ComputeSigningRoot(deposit.Data, d)
	if err != nil {
		t.Fatal(err)
	}
	// Omitting the last byte of the message to generate a wrong signature
	sig := sk.Sign(signedRoot[1:])
	deposit.Data.Signature = sig.Marshal()

	err = depositutil.VerifyDepositSignature(deposit.Data)
	if err == nil {
		t.Fatal("Invalid Deposit Signature")
	}
}
