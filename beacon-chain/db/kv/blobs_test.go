package kv

import (
	"context"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestBlobsSidecar_Overwriting(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconNetworkConfig()
	// For purposes of testing, we only keep blobs around for 3 epochs. At fourth epoch, we will
	// wrap around and overwrite the oldest epoch's elements as the keys for blobs work as a rotating buffer.
	cfg.MinEpochsForBlobsSidecarsRequest = 3
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
}
