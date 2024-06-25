package blocks_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	state_native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/container/trie"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestBatchVerifyDepositsSignatures_Ok(t *testing.T) {
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			PublicKey:             bytesutil.PadTo([]byte{1, 2, 3}, 48),
			WithdrawalCredentials: make([]byte, 32),
			Signature:             make([]byte, 96),
		},
	}
	leaf, err := deposit.Data.HashTreeRoot()
	require.NoError(t, err)
	// We then create a merkle branch for the test.
	depositTrie, err := trie.GenerateTrieFromItems([][]byte{leaf[:]}, params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not generate trie")
	proof, err := depositTrie.MerkleProof(0)
	require.NoError(t, err, "Could not generate proof")

	deposit.Proof = proof
	require.NoError(t, err)
	ok, err := blocks.BatchVerifyDepositsSignatures(context.Background(), []*ethpb.Deposit{deposit})
	require.NoError(t, err)
	require.Equal(t, true, ok)
}

func TestVerifyDeposit_MerkleBranchFailsVerification(t *testing.T) {
	deposit := &ethpb.Deposit{
		Data: &ethpb.Deposit_Data{
			PublicKey:             bytesutil.PadTo([]byte{1, 2, 3}, 48),
			WithdrawalCredentials: make([]byte, 32),
			Signature:             make([]byte, 96),
		},
	}
	leaf, err := deposit.Data.HashTreeRoot()
	require.NoError(t, err)

	// We then create a merkle branch for the test.
	depositTrie, err := trie.GenerateTrieFromItems([][]byte{leaf[:]}, params.BeaconConfig().DepositContractTreeDepth)
	require.NoError(t, err, "Could not generate trie")
	proof, err := depositTrie.MerkleProof(0)
	require.NoError(t, err, "Could not generate proof")

	deposit.Proof = proof
	beaconState, err := state_native.InitializeFromProtoAltair(&ethpb.BeaconStateAltair{
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: []byte{0},
			BlockHash:   []byte{1},
		},
	})
	require.NoError(t, err)
	want := "deposit root did not verify"
	err = blocks.VerifyDeposit(beaconState, deposit)
	require.ErrorContains(t, want, err)
}

func TestIsValidDepositSignature_Ok(t *testing.T) {
	sk, err := bls.RandKey()
	require.NoError(t, err)
	depositData := &ethpb.Deposit_Data{
		PublicKey:             sk.PublicKey().Marshal(),
		Amount:                0,
		WithdrawalCredentials: make([]byte, 32),
		Signature:             make([]byte, fieldparams.BLSSignatureLength),
	}
	dm := &ethpb.DepositMessage{
		PublicKey:             sk.PublicKey().Marshal(),
		WithdrawalCredentials: make([]byte, 32),
		Amount:                0,
	}
	domain, err := signing.ComputeDomain(params.BeaconConfig().DomainDeposit, nil, nil)
	require.NoError(t, err)
	sr, err := signing.ComputeSigningRoot(dm, domain)
	require.NoError(t, err)
	sig := sk.Sign(sr[:])
	depositData.Signature = sig.Marshal()
	valid, err := blocks.IsValidDepositSignature(depositData)
	require.NoError(t, err)
	require.Equal(t, true, valid)
}
