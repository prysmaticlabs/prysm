package validator

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestServer_SetSyncAggregate_EmptyCase(t *testing.T) {
	b, err := blocks.NewBeaconBlock(util.NewBeaconBlockAltair().Block)
	require.NoError(t, err)
	s := &Server{} // Sever is not initialized with sync committee pool.
	s.setSyncAggregate(context.Background(), b)
	agg, err := b.Body().SyncAggregate()
	require.NoError(t, err)

	emptySig := [96]byte{0xC0}
	want := &ethpb.SyncAggregate{
		SyncCommitteeBits:      make([]byte, params.BeaconConfig().SyncCommitteeSize),
		SyncCommitteeSignature: emptySig[:],
	}
	require.DeepEqual(t, want, agg)
}
