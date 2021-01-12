package client

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

func Test_slashableAttestationCheck(t *testing.T) {
	config := &featureconfig.Flags{
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: bytesutil.PadTo([]byte("great block"), 32),
			Source: &ethpb.Checkpoint{
				Epoch: 4,
				Root:  bytesutil.PadTo([]byte("good source"), 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("good target"), 32),
			},
		},
	}
	mockProtector := &mockSlasher.MockProtector{AllowAttestation: false}
	validator.protector = mockProtector
	err := validator.slashableAttestationCheck(context.Background(), att, pubKey, [32]byte{})
	require.ErrorContains(t, failedPostAttSignExternalErr, err)
	mockProtector.AllowAttestation = true
	err = validator.slashableAttestationCheck(context.Background(), att, pubKey, [32]byte{})
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}

func Test_slashableAttestationCheck_Allowed(t *testing.T) {
	config := &featureconfig.Flags{
		SlasherProtection: false,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, _, _, finish := setup(t)
	defer finish()
	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: bytesutil.PadTo([]byte("great block"), 32),
			Source: &ethpb.Checkpoint{
				Epoch: 4,
				Root:  bytesutil.PadTo([]byte("good source"), 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("good target"), 32),
			},
		},
	}
	fakePubkey := bytesutil.ToBytes48([]byte("test"))
	err := validator.slashableAttestationCheck(context.Background(), att, fakePubkey, [32]byte{})
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}

func Test_slashableAttestationCheck_UpdatesLowestSignedEpochs(t *testing.T) {
	t.Skip("Skipped till #8100, when we will save lowest source and target epochs again.")
	config := &featureconfig.Flags{
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, m, validatorKey, finish := setup(t)
	defer finish()
	ctx := context.Background()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: bytesutil.PadTo([]byte("great block"), 32),
			Source: &ethpb.Checkpoint{
				Epoch: 4,
				Root:  bytesutil.PadTo([]byte("good source"), 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("good target"), 32),
			},
		},
	}
	mockProtector := &mockSlasher.MockProtector{AllowAttestation: false}
	validator.protector = mockProtector
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch2
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)
	_, sr, err := validator.getDomainAndSigningRoot(ctx, att.Data)
	require.NoError(t, err)
	err = validator.slashableAttestationCheck(context.Background(), att, pubKey, sr)
	require.ErrorContains(t, "rejected", err, "Expected error on post signature update is detected as slashable")
	mockProtector.AllowAttestation = true
	err = validator.slashableAttestationCheck(context.Background(), att, pubKey, sr)
	require.NoError(t, err, "Expected allowed attestation not to throw error")

	e, err := validator.db.LowestSignedSourceEpoch(context.Background(), pubKey)
	require.NoError(t, err)
	require.Equal(t, uint64(4), e)
	e, err = validator.db.LowestSignedTargetEpoch(context.Background(), pubKey)
	require.NoError(t, err)
	require.Equal(t, uint64(10), e)
}

func Test_slashableAttestationCheck_OK(t *testing.T) {
	config := &featureconfig.Flags{
		SlasherProtection: false,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	ctx := context.Background()
	validator, _, _, finish := setup(t)
	defer finish()
	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: []byte("great block"),
			Source: &ethpb.Checkpoint{
				Epoch: 4,
				Root:  []byte("good source"),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 10,
				Root:  []byte("good target"),
			},
		},
	}
	sr := [32]byte{1}
	fakePubkey := bytesutil.ToBytes48([]byte("test"))
	err := validator.slashableAttestationCheck(ctx, att, fakePubkey, sr)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}

func Test_slashableAttestationCheck_GenesisEpoch(t *testing.T) {
	config := &featureconfig.Flags{
		SlasherProtection: false,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	ctx := context.Background()
	validator, _, _, finish := setup(t)
	defer finish()
	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: bytesutil.PadTo([]byte("great block root"), 32),
			Source: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("great root"), 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  bytesutil.PadTo([]byte("great root"), 32),
			},
		},
	}
	fakePubkey := bytesutil.ToBytes48([]byte("test"))
	err := validator.slashableAttestationCheck(ctx, att, fakePubkey, [32]byte{})
	require.NoError(t, err, "Expected allowed attestation not to throw error")
	e, err := validator.db.LowestSignedSourceEpoch(context.Background(), fakePubkey)
	require.NoError(t, err)
	require.Equal(t, uint64(0), e)
	e, err = validator.db.LowestSignedTargetEpoch(context.Background(), fakePubkey)
	require.NoError(t, err)
	require.Equal(t, uint64(0), e)
}
