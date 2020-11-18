package slashingprotection

import (
	"context"
	"errors"
	"sync"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

func TestService_IsSlashableAttestation_OK(t *testing.T) {
	ctx := context.Background()
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := privKey.PublicKey()
	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], pubKey.Marshal())

	srv := &Service{
		attesterHistoryByPubKey: make(map[[48]byte]kv.EncHistoryData),
	}
	srv.attesterHistoryByPubKey[pubKeyBytes] = kv.NewAttestationHistoryArray(0)

	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: make([]byte, 32),
			Source: &ethpb.Checkpoint{
				Epoch: 4,
				Root:  make([]byte, 32),
			},
			Target: &ethpb.Checkpoint{
				Epoch: 10,
				Root:  make([]byte, 32),
			},
		},
	}
	domainResp := &ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}
	err = srv.IsSlashableAttestation(ctx, att, pubKeyBytes, domainResp)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
}

func Test_isNewAttSlashable_DoubleVote(t *testing.T) {
	ctx := context.Background()
	history := kv.NewAttestationHistoryArray(3)
	// Mark an attestation spanning epochs 0 to 3.
	newAttSource := uint64(0)
	newAttTarget := uint64(3)
	sr1 := [32]byte{1}
	history = markAttestationForTargetEpoch(ctx, history, newAttSource, newAttTarget, sr1)
	lew, err := history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, newAttTarget, lew, "Unexpected latest epoch written")

	// Try an attestation that should be slashable spanning epochs 1 to 3.
	sr2 := [32]byte{2}
	newAttSource = uint64(1)
	newAttTarget = uint64(3)
	assert.Equal(
		t,
		true,
		isNewAttSlashable(ctx, history, newAttSource, newAttTarget, sr2),
		"Expected attestation to be slashable",
	)
}

func TestAttestationHistory_BlocksSurroundAttestationPostSignature(t *testing.T) {
	ctx := context.Background()
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := privKey.PublicKey()
	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], pubKey.Marshal())

	srv := &Service{
		attesterHistoryByPubKey: make(map[[48]byte]kv.EncHistoryData),
	}
	srv.attesterHistoryByPubKey[pubKeyBytes] = kv.NewAttestationHistoryArray(0)

	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: make([]byte, 32),
			Source: &ethpb.Checkpoint{
				Root: make([]byte, 32),
			},
			Target: &ethpb.Checkpoint{
				Root: make([]byte, 32),
			},
		},
	}
	domainResp := &ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}
	notSlashable := 0
	slashable := 0
	var wg sync.WaitGroup
	totalAttestations := 100
	for i := 0; i < totalAttestations; i++ {
		wg.Add(1)
		// Setup many double voting attestations.
		go func(i int) {
			att.Data.Source.Epoch = 110 - uint64(i)
			att.Data.Target.Epoch = 111 + uint64(i)
			err := srv.IsSlashableAttestation(ctx, att, pubKeyBytes, domainResp)
			if err == nil {
				notSlashable++
			} else {
				if errors.Is(err, ErrSlashableAttestation) {
					slashable++
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	require.Equal(t, totalAttestations, notSlashable+slashable)
	require.Equal(t, 1, notSlashable, "Expecting only one attestations to not be slashable")
	require.Equal(t, totalAttestations-1, slashable, "Expecting all other attestations to be found as slashable")
}

func TestService_IsSlashableAttestation_DoubleVote(t *testing.T) {
	ctx := context.Background()
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := privKey.PublicKey()
	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], pubKey.Marshal())

	srv := &Service{
		attesterHistoryByPubKey: make(map[[48]byte]kv.EncHistoryData),
	}
	srv.attesterHistoryByPubKey[pubKeyBytes] = kv.NewAttestationHistoryArray(0)

	att := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{1, 2},
		Data: &ethpb.AttestationData{
			Slot:            5,
			CommitteeIndex:  2,
			BeaconBlockRoot: make([]byte, 32),
			Source: &ethpb.Checkpoint{
				Root: make([]byte, 32),
			},
			Target: &ethpb.Checkpoint{
				Root: make([]byte, 32),
			},
		},
	}
	domainResp := &ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}
	notSlashable := 0
	slashable := 0
	var wg sync.WaitGroup
	totalAttestations := 100
	for i := 0; i < totalAttestations; i++ {
		wg.Add(1)
		// Setup many double voting attestations.
		go func(i int) {
			att.Data.Source.Epoch = 110 - uint64(i)
			att.Data.Target.Epoch = 111
			err := srv.IsSlashableAttestation(ctx, att, pubKeyBytes, domainResp)
			if err == nil {
				notSlashable++
			} else {
				if errors.Is(err, ErrSlashableAttestation) {
					slashable++
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	require.Equal(t, totalAttestations, notSlashable+slashable)
	require.Equal(t, 1, notSlashable, "Expecting only one attestations to not be slashable")
	require.Equal(t, totalAttestations-1, slashable, "Expecting all other attestations to be found as slashable")
}

func TestAttestationHistory_Prunes(t *testing.T) {
	ctx := context.Background()
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod

	signingRoot := [32]byte{1}
	signingRoot2 := [32]byte{2}
	signingRoot3 := [32]byte{3}
	signingRoot4 := [32]byte{4}
	history := kv.NewAttestationHistoryArray(0)

	// Try an attestation on totally unmarked history, should not be slashable.
	require.Equal(t, false, isNewAttSlashable(ctx, history, 0, wsPeriod+5, signingRoot), "Should not be slashable")

	// Mark attestations spanning epochs 0 to 3 and 6 to 9.
	prunedNewAttSource := uint64(0)
	prunedNewAttTarget := uint64(3)
	history = markAttestationForTargetEpoch(ctx, history, prunedNewAttSource, prunedNewAttTarget, signingRoot)
	newAttSource := prunedNewAttSource + 6
	newAttTarget := prunedNewAttTarget + 6
	history = markAttestationForTargetEpoch(ctx, history, newAttSource, newAttTarget, signingRoot2)
	lte, err := history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, newAttTarget, lte, "Unexpected latest epoch")

	// Mark an attestation spanning epochs 54000 to 54003.
	farNewAttSource := newAttSource + wsPeriod
	farNewAttTarget := newAttTarget + wsPeriod
	history = markAttestationForTargetEpoch(ctx, history, farNewAttSource, farNewAttTarget, signingRoot3)
	lte, err = history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, farNewAttTarget, lte, "Unexpected latest epoch")

	require.Equal(t, (*kv.HistoryData)(nil), safeTargetToSource(ctx, history, prunedNewAttTarget), "Unexpectedly marked attestation")
	require.Equal(t, farNewAttSource, safeTargetToSource(ctx, history, farNewAttTarget).Source, "Unexpectedly marked attestation")

	// Try an attestation from existing source to outside prune, should slash.
	if !isNewAttSlashable(ctx, history, newAttSource, farNewAttTarget, signingRoot4) {
		t.Fatalf("Expected attestation of source %d, target %d to be considered slashable", newAttSource, farNewAttTarget)
	}
	// Try an attestation from before existing target to outside prune, should slash.
	if !isNewAttSlashable(ctx, history, newAttTarget-1, farNewAttTarget, signingRoot4) {
		t.Fatalf("Expected attestation of source %d, target %d to be considered slashable", newAttTarget-1, farNewAttTarget)
	}
	// Try an attestation larger than pruning amount, should slash.
	if !isNewAttSlashable(ctx, history, 0, farNewAttTarget+5, signingRoot4) {
		t.Fatalf("Expected attestation of source 0, target %d to be considered slashable", farNewAttTarget+5)
	}
}

func TestAttestationHistory_BlocksSurroundedAttestation(t *testing.T) {
	ctx := context.Background()
	history := kv.NewAttestationHistoryArray(0)

	// Mark an attestation spanning epochs 0 to 3.
	signingRoot := [32]byte{1}
	newAttSource := uint64(0)
	newAttTarget := uint64(3)
	history = markAttestationForTargetEpoch(ctx, history, newAttSource, newAttTarget, signingRoot)
	lte, err := history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, newAttTarget, lte)

	// Try an attestation that should be slashable (being surrounded) spanning epochs 1 to 2.
	newAttSource = uint64(1)
	newAttTarget = uint64(2)
	require.Equal(
		t,
		true,
		isNewAttSlashable(ctx, history, newAttSource, newAttTarget, signingRoot),
		"Expected slashable attestation",
	)
}

func TestAttestationHistory_BlocksSurroundingAttestation(t *testing.T) {
	ctx := context.Background()
	history := kv.NewAttestationHistoryArray(0)
	signingRoot := [32]byte{1}

	// Mark an attestation spanning epochs 1 to 2.
	newAttSource := uint64(1)
	newAttTarget := uint64(2)
	history = markAttestationForTargetEpoch(ctx, history, newAttSource, newAttTarget, signingRoot)
	lte, err := history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, newAttTarget, lte)
	ts, err := history.GetTargetData(ctx, newAttTarget)
	require.NoError(t, err)
	require.Equal(t, newAttSource, ts.Source)

	// Try an attestation that should be slashable (surrounding) spanning epochs 0 to 3.
	newAttSource = uint64(0)
	newAttTarget = uint64(3)
	require.Equal(t, true, isNewAttSlashable(ctx, history, newAttSource, newAttTarget, signingRoot))
}
