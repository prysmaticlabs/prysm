package das

import (
	"bytes"
	"context"
	"testing"

	errors "github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filesystem"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/verification"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func Test_commitmentsToCheck(t *testing.T) {
	windowSlots, err := slots.EpochEnd(params.BeaconConfig().MinEpochsForBlobsSidecarsRequest)
	require.NoError(t, err)
	commits := [][]byte{
		bytesutil.PadTo([]byte("a"), 48),
		bytesutil.PadTo([]byte("b"), 48),
		bytesutil.PadTo([]byte("c"), 48),
		bytesutil.PadTo([]byte("d"), 48),
	}
	cases := []struct {
		name    string
		commits [][]byte
		block   func(*testing.T) blocks.ROBlock
		slot    primitives.Slot
		err     error
	}{
		{
			name: "pre deneb",
			block: func(t *testing.T) blocks.ROBlock {
				bb := util.NewBeaconBlockBellatrix()
				sb, err := blocks.NewSignedBeaconBlock(bb)
				require.NoError(t, err)
				rb, err := blocks.NewROBlock(sb)
				require.NoError(t, err)
				return rb
			},
		},
		{
			name: "commitments within da",
			block: func(t *testing.T) blocks.ROBlock {
				d := util.NewBeaconBlockDeneb()
				d.Block.Body.BlobKzgCommitments = commits
				d.Block.Slot = 100
				sb, err := blocks.NewSignedBeaconBlock(d)
				require.NoError(t, err)
				rb, err := blocks.NewROBlock(sb)
				require.NoError(t, err)
				return rb
			},
			commits: commits,
			slot:    100,
		},
		{
			name: "commitments outside da",
			block: func(t *testing.T) blocks.ROBlock {
				d := util.NewBeaconBlockDeneb()
				// block is from slot 0, "current slot" is window size +1 (so outside the window)
				d.Block.Body.BlobKzgCommitments = commits
				sb, err := blocks.NewSignedBeaconBlock(d)
				require.NoError(t, err)
				rb, err := blocks.NewROBlock(sb)
				require.NoError(t, err)
				return rb
			},
			slot: windowSlots + 1,
		},
		{
			name: "excessive commitments",
			block: func(t *testing.T) blocks.ROBlock {
				d := util.NewBeaconBlockDeneb()
				d.Block.Slot = 100
				// block is from slot 0, "current slot" is window size +1 (so outside the window)
				d.Block.Body.BlobKzgCommitments = commits
				// Double the number of commitments, assert that this is over the limit
				d.Block.Body.BlobKzgCommitments = append(commits, d.Block.Body.BlobKzgCommitments...)
				sb, err := blocks.NewSignedBeaconBlock(d)
				require.NoError(t, err)
				rb, err := blocks.NewROBlock(sb)
				require.NoError(t, err)
				c, err := rb.Block().Body().BlobKzgCommitments()
				require.NoError(t, err)
				require.Equal(t, true, len(c) > fieldparams.MaxBlobsPerBlock)
				return rb
			},
			slot: windowSlots + 1,
			err:  errIndexOutOfBounds,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := c.block(t)
			co, err := commitmentsToCheck(b, c.slot)
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, len(c.commits), len(co))
			for i := 0; i < len(c.commits); i++ {
				require.Equal(t, true, bytes.Equal(c.commits[i], co[i]))
			}
		})
	}
}

func daAlwaysSucceeds(_ [][]byte, _ []*ethpb.BlobSidecar) error {
	return nil
}

type mockDA struct {
	t   *testing.T
	scs []blocks.ROBlob
	err error
}

func TestLazilyPersistent_Missing(t *testing.T) {
	ctx := context.Background()
	store := filesystem.NewEphemeralBlobStorage(t)

	blk, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 3)

	mbv := &mockBlobBatchVerifier{t: t, scs: scs}
	as := NewLazilyPersistentStore(store, mbv)

	// Only one commitment persisted, should return error with other indices
	require.NoError(t, as.Persist(1, scs[2]))
	err := as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	missingErr := MissingIndicesError{}
	require.Equal(t, true, errors.As(err, &missingErr))
	require.DeepEqual(t, []uint64{0, 1}, missingErr.Missing())

	// All but one persisted, return missing idx
	require.NoError(t, as.Persist(1, scs[0]))
	err = as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	require.Equal(t, true, errors.As(err, &missingErr))
	require.DeepEqual(t, []uint64{1}, missingErr.Missing())

	// All persisted, return nil
	require.NoError(t, as.Persist(1, scs[1]))

	require.NoError(t, as.IsDataAvailable(ctx, 1, blk))
}

func TestLazilyPersistent_Mismatch(t *testing.T) {
	ctx := context.Background()
	store := filesystem.NewEphemeralBlobStorage(t)

	blk, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 3)

	mbv := &mockBlobBatchVerifier{t: t, err: errors.New("kzg check should not run")}
	scs[0].KzgCommitment = bytesutil.PadTo([]byte("nope"), 48)
	as := NewLazilyPersistentStore(store, mbv)

	// Only one commitment persisted, should return error with other indices
	require.NoError(t, as.Persist(1, scs[0]))
	err := as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	mismatchErr := CommitmentMismatchError{}
	require.Equal(t, true, errors.As(err, &mismatchErr))
	require.DeepEqual(t, []uint64{0}, mismatchErr.Mismatch())

	// the next time we call the DA check, the mismatched commitment should be evicted
	err = as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	missingErr := MissingIndicesError{}
	require.Equal(t, true, errors.As(err, &missingErr))
	require.DeepEqual(t, []uint64{0, 1, 2}, missingErr.Missing())
}

func TestPersisted(t *testing.T) {
	blk, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 3)
	store := filesystem.NewEphemeralBlobStorage(t)
	for i := range scs {
		require.NoError(t, store.Save(verification.FakeVerifyForTest(t, scs[i])))
	}
	mbv := &mockBlobBatchVerifier{t: t, err: errors.New("kzg check should not run")}
	as := NewLazilyPersistentStore(store, mbv)
	entry := &cacheEntry{}
	dbidx, err := as.persisted(blk.Root(), entry)
	require.NoError(t, err)
	for i := range scs {
		require.Equal(t, dbidx[i], true)
	}

	expectedCommitments, err := blk.Block().Body().BlobKzgCommitments()
	require.NoError(t, err)
	missing := dbidx.missing(len(expectedCommitments))
	require.Equal(t, 0, len(missing))

	// test that the caching is working by returning the wrong set of sidecars
	// and making sure that dbidx still thinks none are missing
	store = filesystem.NewEphemeralBlobStorage(t)
	for i := 1; i < len(scs); i++ {
		require.NoError(t, store.Save(verification.FakeVerifyForTest(t, scs[i])))
	}
	as = NewLazilyPersistentStore(store, &mockBlobBatchVerifier{})
	// note, using the same entry value
	dbidx, err = as.persisted(blk.Root(), entry)
	require.NoError(t, err)
	// same assertions should pass as when all sidecars returned by db
	missing = dbidx.missing(len(expectedCommitments))
	require.Equal(t, 0, len(missing))

	// do it again, but with a fresh cache entry - we should see a missing sidecar
	newEntry := &cacheEntry{}
	dbidx, err = as.persisted(blk.Root(), newEntry)
	require.NoError(t, err)
	missing = dbidx.missing(len(expectedCommitments))
	require.Equal(t, 1, len(missing))
	// only element in missing should be the zero index
	require.Equal(t, uint64(0), missing[0])
}

func TestLazilyPersistent_DBFallback(t *testing.T) {
	ctx := context.Background()

	blk, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 3)
	// Generate the same sidecars index 0 so we can mess with its commitment
	_, scscp := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 1)
	store := filesystem.NewEphemeralBlobStorage(t)
	for i := range scs {
		require.NoError(t, store.Save(verification.FakeVerifyForTest(t, scs[i])))
	}
	mbv := &mockBlobBatchVerifier{t: t, err: errors.New("kzg check should not run")}
	as := NewLazilyPersistentStore(store, mbv)

	// Set up the mismatched commit, but we don't expect this to error because
	// the db contains the sidecars.
	var err error
	scscp[0], err = blocks.NewROBlobWithRoot(scscp[0].BlobSidecar, scs[0].BlockRoot())
	require.NoError(t, err)
	scscp[0].KzgCommitment = bytesutil.PadTo([]byte("nope"), 48)
	require.NoError(t, as.Persist(1, scscp[0]))

	// This should pass since the db is giving us all the right sidecars
	require.NoError(t, as.IsDataAvailable(ctx, 1, blk))

	// now using an empty db, we should fail
	as.store = filesystem.NewEphemeralBlobStorage(t)

	// but we should have pruned, so we'll get a missing error, not mismatch
	err = as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	missingErr := MissingIndicesError{}
	require.Equal(t, true, errors.As(err, &missingErr))
	require.DeepEqual(t, []uint64{0, 1, 2}, missingErr.Missing())

	// put the bad value back in the cache
	require.NoError(t, as.Persist(1, scscp[0]))
	// now we'll get a mismatch error
	err = as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	mismatchErr := CommitmentMismatchError{}
	require.Equal(t, true, errors.As(err, &mismatchErr))
	require.DeepEqual(t, []uint64{0}, mismatchErr.Mismatch())
}

func TestLazyPersistOnceCommitted(t *testing.T) {
	_, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 6)
	as := NewLazilyPersistentStore(filesystem.NewEphemeralBlobStorage(t), &mockBlobBatchVerifier{})
	// stashes as expected
	require.NoError(t, as.Persist(1, scs...))
	// ignores duplicates
	require.ErrorIs(t, as.Persist(1, scs...), ErrDuplicateSidecar)

	// ignores index out of bound
	scs[0].Index = 6
	require.ErrorIs(t, as.Persist(1, scs[0]), errIndexOutOfBounds)

	_, more := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 4)
	// ignores sidecars before the retention period
	slotOOB, err := slots.EpochStart(params.BeaconConfig().MinEpochsForBlobsSidecarsRequest)
	require.NoError(t, err)
	require.NoError(t, as.Persist(32+slotOOB, more[0]))

	// doesn't ignore new sidecars with a different block root
	require.NoError(t, as.Persist(1, more...))
}

type mockBlobBatchVerifier struct {
	t        *testing.T
	scs      []blocks.ROBlob
	err      error
	verified map[[32]byte]primitives.Slot
}

var _ BlobBatchVerifier = &mockBlobBatchVerifier{}

func (m *mockBlobBatchVerifier) VerifiedROBlobs(_ context.Context, scs []blocks.ROBlob) ([]blocks.VerifiedROBlob, error) {
	require.Equal(m.t, len(scs), len(m.scs))
	for i := range m.scs {
		require.Equal(m.t, m.scs[i], scs[i])
	}
	vscs := verification.FakeVerifySliceForTest(m.t, scs)
	return vscs, m.err
}

func (m *mockBlobBatchVerifier) MarkVerified(root [32]byte, slot primitives.Slot) {
	if m.verified == nil {
		m.verified = make(map[[32]byte]primitives.Slot)
	}
	m.verified[root] = slot
}
