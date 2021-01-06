package kv

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	bolt "go.etcd.io/bbolt"
)

func TestStore_CheckSlashableAttestation_DoubleVote(t *testing.T) {
	ctx := context.Background()
	numValidators := 1
	pubKeys := make([][48]byte, numValidators)
	validatorDB := setupDB(t, pubKeys)
	tests := []struct {
		name                string
		existingAttestation *ethpb.IndexedAttestation
		existingSigningRoot [32]byte
		incomingAttestation *ethpb.IndexedAttestation
		incomingSigningRoot [32]byte
		want                bool
	}{
		{
			name:                "different signing root at same target equals a double vote",
			existingAttestation: createAttestation(0, 1 /* target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestation(0, 1 /* target */),
			incomingSigningRoot: [32]byte{2},
			want:                true,
		},
		{
			name:                "same signing root at same target is safe",
			existingAttestation: createAttestation(0, 1 /* target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestation(0, 1 /* target */),
			incomingSigningRoot: [32]byte{1},
			want:                false,
		},
		{
			name:                "different signing root at different target is safe",
			existingAttestation: createAttestation(0, 1 /* target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestation(0, 2 /* target */),
			incomingSigningRoot: [32]byte{2},
			want:                false,
		},
		{
			name:                "no data stored at target should not be considered a double vote",
			existingAttestation: createAttestation(0, 1 /* target */),
			existingSigningRoot: [32]byte{1},
			incomingAttestation: createAttestation(0, 2 /* target */),
			incomingSigningRoot: [32]byte{1},
			want:                false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatorDB.ApplyAttestationForPubKey(
				ctx,
				pubKeys[0],
				tt.existingSigningRoot,
				tt.existingAttestation,
			)
			require.NoError(t, err)
			slashingKind, err := validatorDB.CheckSlashableAttestation(
				ctx,
				pubKeys[0],
				tt.incomingSigningRoot,
				tt.incomingAttestation,
			)
			if tt.want {
				require.NotNil(t, err)
				assert.Equal(t, DoubleVote, slashingKind)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStore_CheckSlashableAttestation_SurroundVote_54kEpochs(t *testing.T) {
	ctx := context.Background()
	numValidators := 1
	numEpochs := uint64(54000)
	pubKeys := make([][48]byte, numValidators)
	validatorDB := setupDB(t, pubKeys)

	// Attest to every (source = epoch, target = epoch + 1) sequential pair
	// since genesis up to and including the weak subjectivity period epoch (54,000).
	err := validatorDB.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pubKeysBucket)
		pkBucket, err := bucket.CreateBucketIfNotExists(pubKeys[0][:])
		if err != nil {
			return err
		}
		sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
		if err != nil {
			return err
		}
		for epoch := uint64(1); epoch < numEpochs; epoch++ {
			att := createAttestation(epoch-1, epoch)
			sourceEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Source.Epoch)
			targetEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Target.Epoch)
			if err := sourceEpochsBucket.Put(sourceEpoch, targetEpoch); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		signingRoot [32]byte
		attestation *ethpb.IndexedAttestation
		want        SlashingKind
	}{
		{
			name:        "surround vote at half of the weak subjectivity period",
			signingRoot: [32]byte{},
			attestation: createAttestation(numEpochs/2, numEpochs),
			want:        SurroundingVote,
		},
		{
			name:        "spanning genesis to weak subjectivity period surround vote",
			signingRoot: [32]byte{},
			attestation: createAttestation(0, numEpochs),
			want:        SurroundingVote,
		},
		{
			name:        "simple surround vote at end of weak subjectivity period",
			signingRoot: [32]byte{},
			attestation: createAttestation(numEpochs-3, numEpochs),
			want:        SurroundingVote,
		},
		{
			name:        "non-slashable vote",
			signingRoot: [32]byte{},
			attestation: createAttestation(numEpochs, numEpochs+1),
			want:        NotSlashable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slashingKind, err := validatorDB.CheckSlashableAttestation(ctx, pubKeys[0], tt.signingRoot, tt.attestation)
			if tt.want != NotSlashable {
				require.NotNil(t, err)
			}
			assert.Equal(t, tt.want, slashingKind)
		})
	}
}

func TestStore_CheckSlashableAttestation_SurroundVote(t *testing.T) {
	ctx := context.Background()
	pubKeys := [][48]byte{{1}}
	validatorDB := setupDB(t, pubKeys)
	source := uint64(1)
	target := uint64(4)
	err := validatorDB.ApplyAttestationForPubKey(ctx, pubKeys[0], [32]byte{}, createAttestation(source, target))
	require.NoError(t, err)
	tests := []struct {
		name        string
		validatorDB *Store
		sourceEpoch uint64
		targetEpoch uint64
		want        SlashingKind
	}{
		{
			name:        "ignores attestations outside of weak subjectivity bounds",
			validatorDB: validatorDB,
			targetEpoch: params.BeaconConfig().WeakSubjectivityPeriod,
			sourceEpoch: params.BeaconConfig().WeakSubjectivityPeriod,
			want:        NotSlashable,
		},
		{
			name:        "detects surrounding attestations",
			validatorDB: validatorDB,
			targetEpoch: target + 1,
			sourceEpoch: source - 1,
			want:        SurroundingVote,
		},
		{
			name:        "detects surrounded attestations",
			validatorDB: validatorDB,
			targetEpoch: target - 1,
			sourceEpoch: source + 1,
			want:        SurroundedVote,
		},
		{
			name:        "new attestation source == old source, but new target < old target",
			validatorDB: validatorDB,
			targetEpoch: target - 1,
			sourceEpoch: source,
			want:        NotSlashable,
		},
		{
			name:        "new attestation source > old source, but new target == old target",
			validatorDB: validatorDB,
			targetEpoch: target,
			sourceEpoch: source + 1,
			want:        NotSlashable,
		},
		{
			name:        "new attestation source and targets equal to old one",
			validatorDB: validatorDB,
			targetEpoch: target,
			sourceEpoch: source,
			want:        NotSlashable,
		},
		{
			name:        "new attestation source == old source, but new target > old target",
			validatorDB: validatorDB,
			targetEpoch: target + 1,
			sourceEpoch: source,
			want:        NotSlashable,
		},
		{
			name:        "new attestation source < old source, but new target == old target",
			validatorDB: validatorDB,
			targetEpoch: target,
			sourceEpoch: source - 1,
			want:        NotSlashable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slashingKind, err := tt.validatorDB.CheckSlashableAttestation(
				ctx, pubKeys[0], [32]byte{}, createAttestation(tt.sourceEpoch, tt.targetEpoch),
			)
			if tt.want == NotSlashable {
				require.NoError(t, err)
				require.Equal(t, NotSlashable, slashingKind)
			} else {
				require.NotNil(t, err)
				require.Equal(t, tt.want, slashingKind)
			}
		})
	}
}

func Test_isSurrounded(t *testing.T) {
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
			if got := isSurrounded(
				tt.args.oldSource, tt.args.oldTarget, tt.args.newSource, tt.args.newTarget,
			); got != tt.want {
				t.Errorf("isSurrounded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isSurrounding(t *testing.T) {
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
			if got := isSurrounding(
				tt.args.oldSource, tt.args.oldTarget, tt.args.newSource, tt.args.newTarget,
			); got != tt.want {
				t.Errorf("isSurrounding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkStore_CheckSlashableAttestation_Surround_SafeAttestation_54kEpochs(b *testing.B) {
	numValidators := 1
	numEpochs := uint64(54000)
	pubKeys := make([][48]byte, numValidators)
	benchCheckSurroundVote(b, pubKeys, numEpochs, false /* surround */)
}

func BenchmarkStore_CheckSurroundVote_Surround_Slashable_54kEpochs(b *testing.B) {
	numValidators := 1
	numEpochs := uint64(54000)
	pubKeys := make([][48]byte, numValidators)
	benchCheckSurroundVote(b, pubKeys, numEpochs, true /* surround */)
}

func benchCheckSurroundVote(
	b *testing.B,
	pubKeys [][48]byte,
	numEpochs uint64,
	shouldSurround bool,
) {
	ctx := context.Background()
	validatorDB, err := NewKVStore(ctx, filepath.Join(os.TempDir(), "benchsurroundvote"), pubKeys)
	require.NoError(b, err, "Failed to instantiate DB")
	defer func() {
		require.NoError(b, validatorDB.Close(), "Failed to close database")
		require.NoError(b, validatorDB.ClearDB(), "Failed to clear database")
	}()
	// Every validator will have attested every (source, target) sequential pair
	// since genesis up to and including the weak subjectivity period epoch (54,000).
	err = validatorDB.update(func(tx *bolt.Tx) error {
		for _, pubKey := range pubKeys {
			bucket := tx.Bucket(pubKeysBucket)
			pkBucket, err := bucket.CreateBucketIfNotExists(pubKey[:])
			if err != nil {
				return err
			}
			sourceEpochsBucket, err := pkBucket.CreateBucketIfNotExists(attestationSourceEpochsBucket)
			if err != nil {
				return err
			}
			for epoch := uint64(1); epoch < numEpochs; epoch++ {
				att := createAttestation(epoch-1, epoch)
				sourceEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Source.Epoch)
				targetEpoch := bytesutil.Uint64ToBytesBigEndian(att.Data.Target.Epoch)
				if err := sourceEpochsBucket.Put(sourceEpoch, targetEpoch); err != nil {
					return err
				}
			}
		}
		return nil
	})
	require.NoError(b, err)

	// Will surround many attestations.
	var surroundingVote *ethpb.IndexedAttestation
	if shouldSurround {
		surroundingVote = createAttestation(numEpochs/2, numEpochs)
	} else {
		surroundingVote = createAttestation(numEpochs+1, numEpochs+2)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pubKey := range pubKeys {
			slashingKind, err := validatorDB.CheckSlashableAttestation(ctx, pubKey, [32]byte{}, surroundingVote)
			if shouldSurround {
				require.NotNil(b, err)
				assert.Equal(b, SurroundingVote, slashingKind)
			} else {
				require.NoError(b, err)
			}
		}
	}
}

func createAttestation(source, target uint64) *ethpb.IndexedAttestation {
	return &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{
				Epoch: source,
			},
			Target: &ethpb.Checkpoint{
				Epoch: target,
			},
		},
	}
}
