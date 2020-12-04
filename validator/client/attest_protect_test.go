package client

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
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
		gomock.Any(), // epoch
	).Times(2).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)
	err := validator.preAttSignValidations(context.Background(), att, pubKey)
	require.ErrorContains(t, failedPreAttSignExternalErr, err)
	mockProtector.AllowAttestation = true
	err = validator.preAttSignValidations(context.Background(), att, pubKey)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
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
		gomock.Any(), // epoch2
	).Return(&ethpb.DomainResponse{SignatureDomain: make([]byte, 32)}, nil /*err*/)
	_, sr, err := validator.getDomainAndSigningRoot(ctx, att.Data)
	require.NoError(t, err)
	err = validator.postAttSignUpdate(context.Background(), att, pubKey, sr)
	require.ErrorContains(t, failedPostAttSignExternalErr, err, "Expected error on post signature update is detected as slashable")
	mockProtector.AllowAttestation = true
	err = validator.postAttSignUpdate(context.Background(), att, pubKey, sr)
	require.NoError(t, err, "Expected allowed attestation not to throw error")

	e, err := validator.db.LowestSignedSourceEpoch(context.Background(), pubKey)
	require.NoError(t, err)
	require.Equal(t, uint64(4), e)
	e, err = validator.db.LowestSignedTargetEpoch(context.Background(), pubKey)
	require.NoError(t, err)
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
	e, err := validator.db.LowestSignedSourceEpoch(context.Background(), fakePubkey)
	require.NoError(t, err)
	require.Equal(t, uint64(0), e)
	e, err = validator.db.LowestSignedTargetEpoch(context.Background(), fakePubkey)
	require.NoError(t, err)
	require.Equal(t, uint64(0), e)
}

func TestAttestationHistory_BlocksDoubleAttestation(t *testing.T) {
	ctx := context.Background()
	history := kv.NewAttestationHistoryArray(3)
	// Mark an attestation spanning epochs 0 to 3.
	newAttSource := uint64(0)
	newAttTarget := uint64(3)
	sr1 := [32]byte{1}
	newHist, err := kv.MarkAllAsAttestedSinceLatestWrittenEpoch(ctx, history, newAttTarget, &kv.HistoryData{
		Source:      newAttSource,
		SigningRoot: sr1[:],
	})
	require.NoError(t, err)
	history = newHist
	lew, err := history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, newAttTarget, lew, "Unexpected latest epoch written")

	// Try an attestation that should be slashable (double att) spanning epochs 1 to 3.
	sr2 := [32]byte{2}
	newAttSource = uint64(1)
	newAttTarget = uint64(3)
	slashable, err := isNewAttSlashable(ctx, history, newAttSource, newAttTarget, sr2)
	require.NoError(t, err)
	if !slashable {
		t.Fatalf("Expected attestation of source %d and target %d to be considered slashable", newAttSource, newAttTarget)
	}
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

func TestAttestationHistory_Prunes(t *testing.T) {
	ctx := context.Background()
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod

	signingRoot := [32]byte{1}
	signingRoot2 := [32]byte{2}
	signingRoot3 := [32]byte{3}
	signingRoot4 := [32]byte{4}
	history := kv.NewAttestationHistoryArray(0)

	// Try an attestation on totally unmarked history, should not be slashable.
	slashable, err := isNewAttSlashable(ctx, history, 0, wsPeriod+5, signingRoot)
	require.NoError(t, err)
	require.Equal(t, false, slashable, "Should not be slashable")

	// Mark attestations spanning epochs 0 to 3 and 6 to 9.
	prunedNewAttSource := uint64(0)
	prunedNewAttTarget := uint64(3)
	newHist, err := kv.MarkAllAsAttestedSinceLatestWrittenEpoch(ctx, history, prunedNewAttTarget, &kv.HistoryData{
		Source:      prunedNewAttSource,
		SigningRoot: signingRoot[:],
	})
	require.NoError(t, err)
	history = newHist
	newAttSource := prunedNewAttSource + 6
	newAttTarget := prunedNewAttTarget + 6
	newHist, err = kv.MarkAllAsAttestedSinceLatestWrittenEpoch(ctx, history, newAttTarget, &kv.HistoryData{
		Source:      newAttSource,
		SigningRoot: signingRoot2[:],
	})
	require.NoError(t, err)
	history = newHist
	lte, err := history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, newAttTarget, lte, "Unexpected latest epoch")

	// Mark an attestation spanning epochs 54000 to 54003.
	farNewAttSource := newAttSource + wsPeriod
	farNewAttTarget := newAttTarget + wsPeriod
	newHist, err = kv.MarkAllAsAttestedSinceLatestWrittenEpoch(ctx, history, farNewAttTarget, &kv.HistoryData{
		Source:      farNewAttSource,
		SigningRoot: signingRoot3[:],
	})
	require.NoError(t, err)
	history = newHist
	lte, err = history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, farNewAttTarget, lte, "Unexpected latest epoch")

	histAtt, err := checkHistoryAtTargetEpoch(ctx, history, lte, prunedNewAttTarget)
	require.NoError(t, err)
	require.Equal(t, (*kv.HistoryData)(nil), histAtt, "Unexpectedly marked attestation")
	histAtt, err = checkHistoryAtTargetEpoch(ctx, history, lte, farNewAttTarget)
	require.NoError(t, err)
	require.Equal(t, farNewAttSource, histAtt.Source, "Unexpectedly marked attestation")

	// Try an attestation from existing source to outside prune, should slash.
	slashable, err = isNewAttSlashable(ctx, history, newAttSource, farNewAttTarget, signingRoot4)
	require.NoError(t, err)
	if !slashable {
		t.Fatalf("Expected attestation of source %d, target %d to be considered slashable", newAttSource, farNewAttTarget)
	}
	// Try an attestation from before existing target to outside prune, should slash.
	slashable, err = isNewAttSlashable(ctx, history, newAttTarget-1, farNewAttTarget, signingRoot4)
	require.NoError(t, err)
	if !slashable {
		t.Fatalf("Expected attestation of source %d, target %d to be considered slashable", newAttTarget-1, farNewAttTarget)
	}
	// Try an attestation larger than pruning amount, should slash.
	slashable, err = isNewAttSlashable(ctx, history, 0, farNewAttTarget+5, signingRoot4)
	require.NoError(t, err)
	if !slashable {
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
	newHist, err := kv.MarkAllAsAttestedSinceLatestWrittenEpoch(ctx, history, newAttTarget, &kv.HistoryData{
		Source:      newAttSource,
		SigningRoot: signingRoot[:],
	})
	require.NoError(t, err)
	history = newHist
	lte, err := history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, newAttTarget, lte)

	// Try an attestation that should be slashable (being surrounded) spanning epochs 1 to 2.
	newAttSource = uint64(1)
	newAttTarget = uint64(2)
	slashable, err := isNewAttSlashable(ctx, history, newAttSource, newAttTarget, signingRoot)
	require.NoError(t, err)
	require.Equal(t, true, slashable, "Expected slashable attestation")
}

func TestAttestationHistory_BlocksSurroundingAttestation(t *testing.T) {
	ctx := context.Background()
	history := kv.NewAttestationHistoryArray(0)
	signingRoot := [32]byte{1}

	// Mark an attestation spanning epochs 1 to 2.
	newAttSource := uint64(1)
	newAttTarget := uint64(2)
	newHist, err := kv.MarkAllAsAttestedSinceLatestWrittenEpoch(ctx, history, newAttTarget, &kv.HistoryData{
		Source:      newAttSource,
		SigningRoot: signingRoot[:],
	})
	require.NoError(t, err)
	history = newHist
	lte, err := history.GetLatestEpochWritten(ctx)
	require.NoError(t, err)
	require.Equal(t, newAttTarget, lte)
	ts, err := history.GetTargetData(ctx, newAttTarget)
	require.NoError(t, err)
	require.Equal(t, newAttSource, ts.Source)

	// Try an attestation that should be slashable (surrounding) spanning epochs 0 to 3.
	newAttSource = uint64(0)
	newAttTarget = uint64(3)
	slashable, err := isNewAttSlashable(ctx, history, newAttSource, newAttTarget, signingRoot)
	require.NoError(t, err)
	require.Equal(t, true, slashable)
}

func Test_isSurroundVote(t *testing.T) {
	ctx := context.Background()
	source := uint64(1)
	target := uint64(4)
	history := kv.NewAttestationHistoryArray(0)
	signingRoot1 := bytesutil.PadTo([]byte{1}, 32)
	hist, err := history.SetTargetData(ctx, target, &kv.HistoryData{
		Source:      source,
		SigningRoot: signingRoot1,
	})
	require.NoError(t, err)
	history = hist
	tests := []struct {
		name               string
		history            kv.EncHistoryData
		latestEpochWritten uint64
		sourceEpoch        uint64
		targetEpoch        uint64
		want               bool
		wantErr            bool
	}{
		{
			name:               "ignores attestations outside of weak subjectivity bounds",
			history:            kv.NewAttestationHistoryArray(0),
			latestEpochWritten: 2 * params.BeaconConfig().WeakSubjectivityPeriod,
			targetEpoch:        params.BeaconConfig().WeakSubjectivityPeriod,
			sourceEpoch:        params.BeaconConfig().WeakSubjectivityPeriod,
			want:               false,
		},
		{
			name:               "detects surrounding attestations",
			history:            history,
			latestEpochWritten: target,
			targetEpoch:        target + 1,
			sourceEpoch:        source - 1,
			want:               true,
		},
		{
			name:               "detects surrounded attestations",
			history:            history,
			latestEpochWritten: target,
			targetEpoch:        target - 1,
			sourceEpoch:        source + 1,
			want:               true,
		},
		{
			name:               "new attestation source == old source, but new target < old target",
			history:            history,
			latestEpochWritten: target,
			targetEpoch:        target - 1,
			sourceEpoch:        source,
			want:               false,
		},
		{
			name:               "new attestation source > old source, but new target == old target",
			history:            history,
			latestEpochWritten: target,
			targetEpoch:        target,
			sourceEpoch:        source + 1,
			want:               false,
		},
		{
			name:               "new attestation source and targets equal to old one",
			history:            history,
			latestEpochWritten: target,
			targetEpoch:        target,
			sourceEpoch:        source,
			want:               false,
		},
		{
			name:               "new attestation source == old source, but new target > old target",
			history:            history,
			latestEpochWritten: target,
			targetEpoch:        target + 1,
			sourceEpoch:        source,
			want:               false,
		},
		{
			name:               "new attestation source < old source, but new target == old target",
			history:            history,
			latestEpochWritten: target,
			targetEpoch:        target,
			sourceEpoch:        source - 1,
			want:               false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isSurroundVote(ctx, tt.history, tt.latestEpochWritten, tt.sourceEpoch, tt.targetEpoch)
			if (err != nil) != tt.wantErr {
				t.Errorf("isSurroundVote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isSurroundVote() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_surroundedByPrevAttestation(t *testing.T) {
	type args struct {
		oldSource uint64
		oldTarget uint64
		newSource uint64
		newTarget uint64
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "0 values returns false",
			args: args{
				oldSource: 0,
				oldTarget: 0,
				newSource: 0,
				newTarget: 0,
			},
			want: false,
		},
		{
			name: "new attestation is surrounded by an old one",
			args: args{
				oldSource: 2,
				oldTarget: 6,
				newSource: 3,
				newTarget: 5,
			},
			want: true,
		},
		{
			name: "new attestation source and targets equal to old one",
			args: args{
				oldSource: 3,
				oldTarget: 5,
				newSource: 3,
				newTarget: 5,
			},
			want: false,
		},
		{
			name: "new attestation source == old source, but new target < old target",
			args: args{
				oldSource: 3,
				oldTarget: 5,
				newSource: 3,
				newTarget: 4,
			},
			want: false,
		},
		{
			name: "new attestation source > old source, but new target == old target",
			args: args{
				oldSource: 3,
				oldTarget: 5,
				newSource: 4,
				newTarget: 5,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := surroundedByPrevAttestation(tt.args.oldSource, tt.args.oldTarget, tt.args.newSource, tt.args.newTarget); got != tt.want {
				t.Errorf("surroundedByPrevAttestation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_surroundingPrevAttestation(t *testing.T) {
	type args struct {
		oldSource uint64
		oldTarget uint64
		newSource uint64
		newTarget uint64
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "0 values returns false",
			args: args{
				oldSource: 0,
				oldTarget: 0,
				newSource: 0,
				newTarget: 0,
			},
			want: false,
		},
		{
			name: "new attestation is surrounding an old one",
			args: args{
				oldSource: 3,
				oldTarget: 5,
				newSource: 2,
				newTarget: 6,
			},
			want: true,
		},
		{
			name: "new attestation source and targets equal to old one",
			args: args{
				oldSource: 3,
				oldTarget: 5,
				newSource: 3,
				newTarget: 5,
			},
			want: false,
		},
		{
			name: "new attestation source == old source, but new target > old target",
			args: args{
				oldSource: 3,
				oldTarget: 5,
				newSource: 3,
				newTarget: 6,
			},
			want: false,
		},
		{
			name: "new attestation source < old source, but new target == old target",
			args: args{
				oldSource: 3,
				oldTarget: 5,
				newSource: 2,
				newTarget: 5,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := surroundingPrevAttestation(tt.args.oldSource, tt.args.oldTarget, tt.args.newSource, tt.args.newTarget); got != tt.want {
				t.Errorf("surroundingPrevAttestation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkHistoryAtTargetEpoch(t *testing.T) {
	ctx := context.Background()
	history := kv.NewAttestationHistoryArray(0)
	signingRoot1 := bytesutil.PadTo([]byte{1}, 32)
	hist, err := history.SetTargetData(ctx, 1, &kv.HistoryData{
		Source:      0,
		SigningRoot: signingRoot1,
	})
	require.NoError(t, err)
	history = hist
	tests := []struct {
		name               string
		history            kv.EncHistoryData
		latestEpochWritten uint64
		targetEpoch        uint64
		want               *kv.HistoryData
		wantErr            bool
	}{
		{
			name:               "ignores difference in epochs outside of weak subjectivity bounds",
			history:            kv.NewAttestationHistoryArray(0),
			latestEpochWritten: 2 * params.BeaconConfig().WeakSubjectivityPeriod,
			targetEpoch:        params.BeaconConfig().WeakSubjectivityPeriod,
			want:               nil,
			wantErr:            false,
		},
		{
			name:               "ignores target epoch > latest written epoch",
			history:            kv.NewAttestationHistoryArray(0),
			latestEpochWritten: params.BeaconConfig().WeakSubjectivityPeriod,
			targetEpoch:        params.BeaconConfig().WeakSubjectivityPeriod + 1,
			want:               nil,
			wantErr:            false,
		},
		{
			name:               "target epoch == latest written epoch should return correct results",
			history:            history,
			latestEpochWritten: 1,
			targetEpoch:        1,
			want: &kv.HistoryData{
				Source:      0,
				SigningRoot: signingRoot1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkHistoryAtTargetEpoch(ctx, tt.history, tt.latestEpochWritten, tt.targetEpoch)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkHistoryAtTargetEpoch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkHistoryAtTargetEpoch() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_differenceOutsideWeakSubjectivityBounds(t *testing.T) {
	tests := []struct {
		name               string
		want               bool
		latestEpochWritten uint64
		targetEpoch        uint64
	}{
		{
			name:               "difference of weak subjectivity period - 1 returns false",
			latestEpochWritten: (2 * params.BeaconConfig().WeakSubjectivityPeriod) - 1,
			targetEpoch:        params.BeaconConfig().WeakSubjectivityPeriod,
			want:               false,
		},
		{
			name:               "difference of weak subjectivity period returns true",
			latestEpochWritten: 2 * params.BeaconConfig().WeakSubjectivityPeriod,
			targetEpoch:        params.BeaconConfig().WeakSubjectivityPeriod,
			want:               true,
		},
		{
			name:               "difference > weak subjectivity period returns true",
			latestEpochWritten: (2 * params.BeaconConfig().WeakSubjectivityPeriod) + 1,
			targetEpoch:        params.BeaconConfig().WeakSubjectivityPeriod,
			want:               true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := differenceOutsideWeakSubjectivityBounds(tt.latestEpochWritten, tt.targetEpoch); got != tt.want {
				t.Errorf("differenceOutsideWeakSubjectivityBounds() = %v, want %v", got, tt.want)
			}
		})
	}
}
