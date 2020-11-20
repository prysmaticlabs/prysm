package local

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/db/kv"
	dbtest "github.com/prysmaticlabs/prysm/validator/db/testing"
)

func TestService_IsSlashableAttestation_OK(t *testing.T) {
	ctx := context.Background()
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := privKey.PublicKey()
	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], pubKey.Marshal())
	validatorDB := dbtest.SetupDB(t, [][48]byte{pubKeyBytes})
	srv := &Service{
		validatorDB: validatorDB,
	}
	require.NoError(
		t,
		validatorDB.SaveAttestationHistoryForPubKey(ctx, pubKeyBytes, kv.NewAttestationHistoryArray(0)),
	)
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
	dummySigningRoot := [32]byte{}
	copy(dummySigningRoot[:], "root")
	slashable, err := srv.IsSlashableAttestation(ctx, att, pubKeyBytes, dummySigningRoot)
	require.NoError(t, err, "Expected allowed attestation not to throw error")
	assert.Equal(t, false, slashable)
}

func TestAttestationHistory_BlocksSurroundAttestationPostSignature(t *testing.T) {
	ctx := context.Background()
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := privKey.PublicKey()
	pubKeyBytes := [48]byte{}
	copy(pubKeyBytes[:], pubKey.Marshal())

	validatorDB := dbtest.SetupDB(t, [][48]byte{pubKeyBytes})
	srv := &Service{
		validatorDB: validatorDB,
	}
	require.NoError(
		t,
		validatorDB.SaveAttestationHistoryForPubKey(ctx, pubKeyBytes, kv.NewAttestationHistoryArray(0)),
	)
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
	dummySigningRoot := [32]byte{}
	copy(dummySigningRoot[:], "root")
	notSlashable := 0
	slashable := 0
	var wg sync.WaitGroup
	totalAttestations := 100
	for i := 0; i < totalAttestations; i++ {
		wg.Add(1)
		go func(i int) {
			att.Data.Source.Epoch = 110 - uint64(i)
			att.Data.Target.Epoch = 111 + uint64(i)
			isSlashable, err := srv.IsSlashableAttestation(ctx, att, pubKeyBytes, dummySigningRoot)
			require.NoError(t, err)
			if isSlashable {
				slashable++
			} else {
				notSlashable++
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

	validatorDB := dbtest.SetupDB(t, [][48]byte{pubKeyBytes})
	srv := &Service{
		validatorDB: validatorDB,
	}
	require.NoError(
		t,
		validatorDB.SaveAttestationHistoryForPubKey(ctx, pubKeyBytes, kv.NewAttestationHistoryArray(0)),
	)
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
			dummySigningRoot := [32]byte{}
			copy(dummySigningRoot[:], fmt.Sprintf("%d", i))
			isSlashable, err := srv.IsSlashableAttestation(ctx, att, pubKeyBytes, dummySigningRoot)
			require.NoError(t, err)
			if isSlashable {
				slashable++
			} else {
				notSlashable++
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	require.Equal(t, totalAttestations, notSlashable+slashable)
	require.Equal(t, 1, notSlashable, "Expecting only one attestations to not be slashable")
	require.Equal(t, totalAttestations-1, slashable, "Expecting all other attestations to be found as slashable")
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

func Test_isDoubleVote(t *testing.T) {
	ctx := context.Background()
	history := kv.NewAttestationHistoryArray(0)
	signingRoot1 := bytesutil.PadTo([]byte{1}, 32)
	signingRoot2 := bytesutil.PadTo([]byte{2}, 32)
	hist, err := history.SetTargetData(ctx, 1, &kv.HistoryData{
		Source:      0,
		SigningRoot: signingRoot1,
	})
	require.NoError(t, err)
	history = hist
	tests := []struct {
		name        string
		targetEpoch uint64
		history     kv.EncHistoryData
		signingRoot []byte
		want        bool
		wantErr     bool
	}{
		{
			name:        "vote exists but matching signing root should not lead to double vote",
			targetEpoch: 1,
			history:     history,
			signingRoot: signingRoot1,
			want:        false,
		},
		{
			name:        "vote exists and non-matching signing root should lead to double vote",
			targetEpoch: 1,
			history:     history,
			signingRoot: signingRoot2,
			want:        true,
		},
		{
			name:        "vote does not exist should not lead to double vote",
			targetEpoch: 2,
			history:     history,
			signingRoot: []byte{},
			want:        false,
		},
		{
			name:        "error retrieving target data should not lead to double vote",
			targetEpoch: 0,
			history:     kv.EncHistoryData{},
			signingRoot: []byte{},
			want:        false,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := [32]byte{}
			copy(root[:], tt.signingRoot)
			got, err := isDoubleVote(ctx, tt.history, tt.targetEpoch, root)
			if (err != nil) != tt.wantErr {
				t.Errorf("isDoubleVote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isDoubleVote() got = %v, want %v", got, tt.want)
			}
		})
	}
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
