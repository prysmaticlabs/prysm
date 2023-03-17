package sync

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestBlobsByRangeValidation(t *testing.T) {
	cfg := params.BeaconConfig()
	repositionFutureEpochs(cfg)
	undo, err := params.SetActiveWithUndo(cfg)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, undo())
	}()
	denebSlot, err := slots.EpochStart(params.BeaconConfig().DenebForkEpoch)
	require.NoError(t, err)

	minReqEpochs := params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest
	minReqSlots, err := slots.EpochStart(minReqEpochs)
	require.NoError(t, err)
	// spec criteria for mix,max bound checking
	/*
		Clients MUST keep a record of signed blobs sidecars seen on the epoch range
		[max(current_epoch - MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS, DENEB_FORK_EPOCH), current_epoch]
		where current_epoch is defined by the current wall-clock time,
		and clients MUST support serving requests of blobs on this range.
	*/
	defaultCurrent := denebSlot + 100 + minReqSlots
	defaultMinStart, err := blobsByRangeMinStartSlot(defaultCurrent)
	require.NoError(t, err)
	cases := []struct {
		name    string
		current types.Slot
		req     *ethpb.BlobSidecarsByRangeRequest

		start types.Slot
		end   types.Slot
		batch uint64
		err   error
	}{
		{
			name:    "start at current",
			current: denebSlot + 100,
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: denebSlot + 100,
				Count:     10,
			},
			start: denebSlot + 100,
			end:   denebSlot + 100,
			batch: 10,
		},
		{
			name:    "start after current",
			current: denebSlot,
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: denebSlot + 100,
				Count:     10,
			},
			start: denebSlot,
			end:   denebSlot,
			batch: 0,
		},
		{
			name:    "start before current_epoch - MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS",
			current: defaultCurrent,
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: defaultMinStart - 100,
				Count:     10,
			},
			start: defaultMinStart,
			end:   defaultMinStart,
			batch: 10,
		},
		{
			name:    "start before current_epoch - MIN_EPOCHS_FOR_BLOB_SIDECARS_REQUESTS - end still valid",
			current: defaultCurrent,
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: defaultMinStart - 10,
				Count:     20,
			},
			start: defaultMinStart,
			end:   defaultMinStart + 9,
			batch: 20,
		},
		{
			name:    "count > MAX_REQUEST_BLOB_SIDECARS",
			current: defaultCurrent,
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: defaultMinStart - 10,
				Count:     1000,
			},
			start: defaultMinStart,
			end:   defaultMinStart - 10 + 999,
			// a large count is ok, we just limit the amount of actual responses
			batch: uint64(flags.Get().BlobBatchLimit),
		},
		{
			name:    "start + count > current",
			current: defaultCurrent,
			req: &ethpb.BlobSidecarsByRangeRequest{
				StartSlot: defaultCurrent + 100,
				Count:     100,
			},
			start: defaultCurrent,
			end:   defaultCurrent,
			batch: 0,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end, batch, err := validateBlobsByRange(c.req, c.current)
			if c.err != nil {
				require.ErrorIs(t, err, c.err)
				return
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, c.start, start)
			require.Equal(t, c.end, end)
			require.Equal(t, c.batch, batch)
		})
	}
}
