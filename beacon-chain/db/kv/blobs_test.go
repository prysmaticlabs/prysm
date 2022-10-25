package kv

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	bolt "go.etcd.io/bbolt"
)

func TestBlobsSidecar_Overwriting(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconNetworkConfig()
	// For purposes of testing, we only keep blob sidecars around for 2 epochs. At third epoch, we will
	// wrap around and overwrite the oldest epoch's elements as the keys for blobs work as a rotating buffer.
	cfg.MinEpochsForBlobsSidecarsRequest = 2
	params.OverrideBeaconNetworkConfig(cfg)
	db := setupDB(t)

	sidecars := make([]*ethpb.BlobsSidecar, 0)
	numSlots := uint64(cfg.MinEpochsForBlobsSidecarsRequest) * uint64(params.BeaconConfig().SlotsPerEpoch)
	for i := uint64(0); i < numSlots; i++ {
		// There can be multiple blobs per slot with different block roots, so we create some
		// in order to have a thorough test.
		root1 := bytesutil.ToBytes32([]byte(fmt.Sprintf("foo-%d", i)))
		root2 := bytesutil.ToBytes32([]byte(fmt.Sprintf("bar-%d", i)))
		sidecars = append(sidecars, &ethpb.BlobsSidecar{
			BeaconBlockRoot: root1[:],
			BeaconBlockSlot: types.Slot(i),
			Blobs:           make([]*enginev1.Blob, 0),
			AggregatedProof: make([]byte, 48),
		})
		sidecars = append(sidecars, &ethpb.BlobsSidecar{
			BeaconBlockRoot: root2[:],
			BeaconBlockSlot: types.Slot(i),
			Blobs:           make([]*enginev1.Blob, 0),
			AggregatedProof: make([]byte, 48),
		})
	}
	ctx := context.Background()
	for _, blobSidecar := range sidecars {
		require.NoError(t, db.SaveBlobsSidecar(ctx, blobSidecar))
		require.Equal(t, true, db.HasBlobsSidecar(ctx, bytesutil.ToBytes32(blobSidecar.BeaconBlockRoot)))
	}

	// We check there are only two blob sidecars stored at slot 0, as an example.
	keyPrefix := append(bytesutil.SlotToBytesBigEndian(0), bytesutil.SlotToBytesBigEndian(0)...)
	numBlobs := countBlobsWithPrefix(t, db, keyPrefix)
	require.Equal(t, 2, numBlobs)

	// Attempting to save another blob sidecar with slot 0 and a new block root should result
	// in three blob sidecars stored at slot 0. This means we are NOT overwriting old data.
	root := bytesutil.ToBytes32([]byte("baz-0"))
	sidecar := &ethpb.BlobsSidecar{
		BeaconBlockRoot: root[:],
		BeaconBlockSlot: types.Slot(0),
		Blobs:           make([]*enginev1.Blob, 0),
		AggregatedProof: make([]byte, 48),
	}
	require.NoError(t, db.SaveBlobsSidecar(ctx, sidecar))
	require.Equal(t, true, db.HasBlobsSidecar(ctx, bytesutil.ToBytes32(sidecar.BeaconBlockRoot)))

	numBlobs = countBlobsWithPrefix(t, db, keyPrefix)
	require.Equal(t, 3, numBlobs)

	// Now, we attempt to save a blob sidecar with slot = MAX_SLOTS_TO_PERSIST_BLOBS. This SHOULD cause us to
	// overwrite ALL old data at slot 0, as slot % MAX_SLOTS_TO_PERSIST_BLOBS will be equal to 0.
	// We should expect a single blob sidecar to exist at slot 0 after this operation.
	root = bytesutil.ToBytes32([]byte(fmt.Sprintf("foo-%d", numSlots)))
	sidecar = &ethpb.BlobsSidecar{
		BeaconBlockRoot: root[:],
		BeaconBlockSlot: types.Slot(numSlots),
		Blobs:           make([]*enginev1.Blob, 0),
		AggregatedProof: make([]byte, 48),
	}
	require.NoError(t, db.SaveBlobsSidecar(ctx, sidecar))
	require.Equal(t, true, db.HasBlobsSidecar(ctx, bytesutil.ToBytes32(sidecar.BeaconBlockRoot)))

	keyPrefix = append(bytesutil.SlotToBytesBigEndian(0), bytesutil.SlotToBytesBigEndian(64)...)
	numBlobs = countBlobsWithPrefix(t, db, keyPrefix)
	require.Equal(t, 1, numBlobs)
}

func countBlobsWithPrefix(t *testing.T, db *Store, prefix []byte) int {
	numBlobSidecars := 0
	require.NoError(t, db.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(blobsBucket).Cursor()
		for k, _ := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			numBlobSidecars++
		}
		return nil
	}))
	return numBlobSidecars
}
