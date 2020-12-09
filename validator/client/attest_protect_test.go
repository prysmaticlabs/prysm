package client

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	mockSlasher "github.com/prysmaticlabs/prysm/validator/testing"
)

func TestPreSignatureValidation(t *testing.T) {
	config := &featureconfig.Flags{
		SlasherProtection: true,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, m, validatorKey, finish := setup(t)
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
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		&ethpb.DomainRequest{Epoch: 10, Domain: []byte{1, 0, 0, 0}},
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)
	err := validator.preAttSignValidations(context.Background(), att, pubKey)
	require.ErrorContains(t, failedPreAttSignExternalErr, err)
	mockProtector.AllowAttestation = true
	err = validator.preAttSignValidations(context.Background(), att, pubKey)
	require.NoError(t, err, "Expected allowed attestation not to throw error")

	e, exists, err := validator.db.LowestSignedSourceEpoch(context.Background(), pubKey)
	require.NoError(t, err)
	require.Equal(t, false, exists)
	require.Equal(t, uint64(0), e)
	e, exists, err = validator.db.LowestSignedTargetEpoch(context.Background(), pubKey)
	require.NoError(t, err)
	require.Equal(t, false, exists)
	require.Equal(t, uint64(0), e)

	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		&ethpb.DomainRequest{Epoch: 10, Domain: []byte{1, 0, 0, 0}},
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)
	require.NoError(t, validator.db.SaveLowestSignedTargetEpoch(context.Background(), pubKey, att.Data.Target.Epoch+1))
	err = validator.preAttSignValidations(context.Background(), att, pubKey)
	require.ErrorContains(t, "could not sign attestation lower than lowest target epoch in db", err)
	require.NoError(t, validator.db.SaveLowestSignedSourceEpoch(context.Background(), pubKey, att.Data.Source.Epoch+1))
	err = validator.preAttSignValidations(context.Background(), att, pubKey)
	require.ErrorContains(t, "could not sign attestation lower than lowest source epoch in db", err)
}

func TestPreSignatureValidation_NilLocal(t *testing.T) {
	config := &featureconfig.Flags{
		SlasherProtection: false,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	validator, m, _, finish := setup(t)
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
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)
	err := validator.preAttSignValidations(context.Background(), att, fakePubkey)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}

func TestPostSignatureUpdate(t *testing.T) {
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
		&ethpb.DomainRequest{Epoch: 10, Domain: []byte{1, 0, 0, 0}},
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)
	_, sr, err := validator.getDomainAndSigningRoot(ctx, att.Data)
	require.NoError(t, err)
	err = validator.postAttSignUpdate(context.Background(), att, pubKey, sr)
	require.ErrorContains(t, failedPostAttSignExternalErr, err, "Expected error on post signature update is detected as slashable")
	mockProtector.AllowAttestation = true
	err = validator.postAttSignUpdate(context.Background(), att, pubKey, sr)
	require.NoError(t, err, "Expected allowed attestation not to throw error")

	e, exists, err := validator.db.LowestSignedSourceEpoch(context.Background(), pubKey)
	require.NoError(t, err)
	require.Equal(t, true, exists)
	require.Equal(t, uint64(4), e)
	e, exists, err = validator.db.LowestSignedTargetEpoch(context.Background(), pubKey)
	require.NoError(t, err)
	require.Equal(t, true, exists)
	require.Equal(t, uint64(10), e)
}

func TestPostSignatureUpdate_NilLocal(t *testing.T) {
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
	err := validator.postAttSignUpdate(ctx, att, fakePubkey, sr)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}

func TestPrePostSignatureUpdate_NilLocalGenesis(t *testing.T) {
	config := &featureconfig.Flags{
		SlasherProtection: false,
	}
	reset := featureconfig.InitWithReset(config)
	defer reset()
	ctx := context.Background()
	validator, m, _, finish := setup(t)
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
	m.validatorClient.EXPECT().DomainData(
		gomock.Any(), // ctx
		gomock.Any(), // epoch2
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)
	sr := [32]byte{1}
	fakePubkey := bytesutil.ToBytes48([]byte("test"))
	err := validator.preAttSignValidations(ctx, att, fakePubkey)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
	err = validator.postAttSignUpdate(ctx, att, fakePubkey, sr)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
	e, exists, err := validator.db.LowestSignedSourceEpoch(context.Background(), fakePubkey)
	require.NoError(t, err)
	require.Equal(t, true, exists)
	require.Equal(t, uint64(0), e)
	e, exists, err = validator.db.LowestSignedTargetEpoch(context.Background(), fakePubkey)
	require.NoError(t, err)
	require.Equal(t, true, exists)
	require.Equal(t, uint64(0), e)
}

func TestAttestationHistory_BlocksSurroundAttestationPostSignature(t *testing.T) {
	ctx := context.Background()
	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: []byte("great block"),
			Source: &ethpb.Checkpoint{
				Root: []byte("good source"),
			},
			Target: &ethpb.Checkpoint{
				Root: []byte("good target"),
			},
		},
	}

	v, _, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	passThrough := 0
	slashable := 0
	var wg sync.WaitGroup
	for i := uint64(0); i < 100; i++ {

		wg.Add(1)
		//Test surround and surrounded attestations.
		go func(i uint64) {
			sr := [32]byte{1}
			att.Data.Source.Epoch = 110 - i
			att.Data.Target.Epoch = 111 + i
			err := v.postAttSignUpdate(ctx, att, pubKey, sr)
			if err == nil {
				passThrough++
			} else {
				if strings.Contains(err.Error(), failedAttLocalProtectionErr) {
					slashable++
				}
				t.Logf("attestation source epoch %d", att.Data.Source.Epoch)
				t.Logf("attestation target epoch %d", att.Data.Target.Epoch)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	require.Equal(t, 1, passThrough, "Expecting only one attestations to go through and all others to be found to be slashable")
	require.Equal(t, 99, slashable, "Expecting 99 attestations to be found as slashable")
}

func TestAttestationHistory_BlocksDoubleAttestationPostSignature(t *testing.T) {
	ctx := context.Background()
	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: []byte("great block"),
			Source: &ethpb.Checkpoint{
				Root: []byte("good source"),
			},
			Target: &ethpb.Checkpoint{
				Root: []byte("good target"),
			},
		},
	}

	v, _, validatorKey, finish := setup(t)
	defer finish()
	pubKey := [48]byte{}
	copy(pubKey[:], validatorKey.PublicKey().Marshal())
	passThrough := 0
	slashable := 0
	var wg sync.WaitGroup
	for i := uint64(0); i < 100; i++ {

		wg.Add(1)
		//Test double attestations.
		go func(i uint64) {
			sr := [32]byte{byte(i)}
			att.Data.Source.Epoch = 110 - i
			att.Data.Target.Epoch = 111
			err := v.postAttSignUpdate(ctx, att, pubKey, sr)
			if err == nil {
				passThrough++
			} else {
				if strings.Contains(err.Error(), failedAttLocalProtectionErr) {
					slashable++
				}
				t.Logf("attestation source epoch %d", att.Data.Source.Epoch)
				t.Logf("signing root %d", att.Data.Target.Epoch)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	require.Equal(t, 1, passThrough, "Expecting only one attestations to go through and all others to be found to be slashable")
	require.Equal(t, 99, slashable, "Expecting 99 attestations to be found as slashable")

}
