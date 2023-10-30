package das

import (
	"bytes"
	"context"
	"testing"

	errors "github.com/pkg/errors"
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
	windowSlots, err := slots.EpochEnd(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest)
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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := c.block(t)
			co := commitmentsToCheck(b, c.slot)
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
	t    *testing.T
	cmts [][]byte
	scs  []*ethpb.BlobSidecar
	err  error
}

func (m *mockDA) expectedArguments(gotC [][]byte, gotSC []*ethpb.BlobSidecar) error {
	require.Equal(m.t, len(m.cmts), len(gotC))
	require.Equal(m.t, len(gotSC), len(m.scs))
	for i := range m.cmts {
		require.Equal(m.t, true, bytes.Equal(m.cmts[i], gotC[i]))
	}
	for i := range m.scs {
		require.Equal(m.t, m.scs[i], gotSC[i])
	}
	return m.err
}

func TestLazilyPersistent_Missing(t *testing.T) {
	ctx := context.Background()
	db := &mockBlobsDB{}
	as := NewLazilyPersistentStore(db)

	blk, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 3)

	expectedCommitments, err := blk.Block().Body().BlobKzgCommitments()
	require.NoError(t, err)
	vf := &mockDA{
		t:    t,
		cmts: expectedCommitments,
		scs:  scs,
	}
	as.verifyKZG = vf.expectedArguments

	// Only one commitment persisted, should return error with other indices
	as.PersistOnceCommitted(ctx, 1, scs[2])
	err = as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	missingErr := MissingIndicesError{}
	require.Equal(t, true, errors.As(err, &missingErr))
	require.DeepEqual(t, []uint64{0, 1}, missingErr.Missing())

	// All but one persisted, return missing idx
	as.PersistOnceCommitted(ctx, 1, scs[0])
	err = as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	require.Equal(t, true, errors.As(err, &missingErr))
	require.DeepEqual(t, []uint64{1}, missingErr.Missing())

	// All persisted, return nil
	as.PersistOnceCommitted(ctx, 1, scs[1])
	require.NoError(t, as.IsDataAvailable(ctx, 1, blk))
}

func TestLazilyPersistent_Mismatch(t *testing.T) {
	ctx := context.Background()
	db := &mockBlobsDB{}
	as := NewLazilyPersistentStore(db)

	blk, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 3)

	vf := &mockDA{
		t:   t,
		err: errors.New("kzg check should not run"),
	}
	as.verifyKZG = vf.expectedArguments
	scs[0].KzgCommitment = bytesutil.PadTo([]byte("nope"), 48)

	// Only one commitment persisted, should return error with other indices
	as.PersistOnceCommitted(ctx, 1, scs[0])
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
	ctx := context.Background()

	blk, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 3)
	db := &mockBlobsDB{
		BlobSidecarsByRootCallback: func(ctx context.Context, beaconBlockRoot [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error) {
			return scs, nil
		},
	}
	vf := &mockDA{
		t:   t,
		err: errors.New("kzg check should not run"),
	}
	as := NewLazilyPersistentStore(db)
	as.verifyKZG = vf.expectedArguments
	entry := &cacheEntry{}
	dbidx, err := as.persisted(ctx, blk.Root(), entry)
	require.NoError(t, err)
	require.Equal(t, false, dbidx[0] == nil)
	for i := range scs {
		require.Equal(t, *dbidx[i], bytesutil.ToBytes48(scs[i].KzgCommitment))
	}

	expectedCommitments, err := blk.Block().Body().BlobKzgCommitments()
	require.NoError(t, err)
	missing, err := dbidx.missing(expectedCommitments)
	require.NoError(t, err)
	require.Equal(t, 0, len(missing))

	// test that the caching is working by returning the wrong set of sidecars
	// and making sure that dbidx still thinks none are missing
	db = &mockBlobsDB{
		BlobSidecarsByRootCallback: func(ctx context.Context, beaconBlockRoot [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error) {
			return scs[1:], nil
		},
	}
	as = NewLazilyPersistentStore(db)
	// note, using the same entry value
	dbidx, err = as.persisted(ctx, blk.Root(), entry)
	require.NoError(t, err)
	// same assertions should pass as when all sidecars returned by db
	missing, err = dbidx.missing(expectedCommitments)
	require.NoError(t, err)
	require.Equal(t, 0, len(missing))

	// do it again, but with a fresh cache entry - we should see a missing sidecar
	newEntry := &cacheEntry{}
	dbidx, err = as.persisted(ctx, blk.Root(), newEntry)
	require.NoError(t, err)
	missing, err = dbidx.missing(expectedCommitments)
	require.NoError(t, err)
	require.Equal(t, 1, len(missing))
	// only element in missing should be the zero index
	require.Equal(t, uint64(0), missing[0])
}

func TestLazilyPersistent_DBFallback(t *testing.T) {
	ctx := context.Background()

	blk, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 3)
	// Generate the same sidecars index 0 so we can mess with its commitment
	_, scscp := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 1)
	db := &mockBlobsDB{
		BlobSidecarsByRootCallback: func(ctx context.Context, beaconBlockRoot [32]byte, indices ...uint64) ([]*ethpb.BlobSidecar, error) {
			return scs, nil
		},
	}
	vf := &mockDA{
		t:   t,
		err: errors.New("kzg check should not run"),
	}
	as := NewLazilyPersistentStore(db)
	as.verifyKZG = vf.expectedArguments

	// Set up the mismatched commit, but we don't expect this to error because
	// the db contains the sidecars.
	scscp[0].BlockRoot = scs[0].BlockRoot
	scscp[0].KzgCommitment = bytesutil.PadTo([]byte("nope"), 48)
	as.PersistOnceCommitted(ctx, 1, scscp[0])
	// This should pass since the db is giving us all the right sidecars
	require.NoError(t, as.IsDataAvailable(ctx, 1, blk))

	// now using an empty db, we should fail
	as.db = &mockBlobsDB{}

	// but we should have pruned, so we'll get a missing error, not mismatch
	err := as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	missingErr := MissingIndicesError{}
	require.Equal(t, true, errors.As(err, &missingErr))
	require.DeepEqual(t, []uint64{0, 1, 2}, missingErr.Missing())

	// put the bad value back in the cache
	persisted := as.PersistOnceCommitted(ctx, 1, scscp[0])
	require.Equal(t, 1, len(persisted))
	// now we'll get a mismatch error
	err = as.IsDataAvailable(ctx, 1, blk)
	require.NotNil(t, err)
	mismatchErr := CommitmentMismatchError{}
	require.Equal(t, true, errors.As(err, &mismatchErr))
	require.DeepEqual(t, []uint64{0}, mismatchErr.Mismatch())
}

func TestLazyPersistOnceCommitted(t *testing.T) {
	ctx := context.Background()
	_, scs := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 6)
	as := NewLazilyPersistentStore(&mockBlobsDB{})
	// stashes as expected
	require.Equal(t, 6, len(as.PersistOnceCommitted(ctx, 1, scs...)))
	// ignores duplicates
	require.Equal(t, 0, len(as.PersistOnceCommitted(ctx, 1, scs...)))

	// ignores index out of bound
	scs[0].Index = 6
	require.Equal(t, 0, len(as.PersistOnceCommitted(ctx, 1, scs[0])))

	_, more := util.GenerateTestDenebBlockWithSidecar(t, [32]byte{}, 1, 4)
	// ignores sidecars before the retention period
	slotOOB, err := slots.EpochStart(params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest)
	require.NoError(t, err)
	require.Equal(t, 0, len(as.PersistOnceCommitted(ctx, 32+slotOOB, more[0])))

	// doesn't ignore new sidecars with a different block root
	require.Equal(t, 4, len(as.PersistOnceCommitted(ctx, 1, more...)))
}
