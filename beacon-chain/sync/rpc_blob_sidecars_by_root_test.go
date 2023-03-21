package sync

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	p2pTypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func TestBlobsByRootValidation(t *testing.T) {
	cfg := params.BeaconConfig()
	repositionFutureEpochs(cfg)
	undo, err := params.SetActiveWithUndo(cfg)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, undo())
	}()
	capellaSlot, err := slots.EpochStart(params.BeaconConfig().CapellaForkEpoch)
	require.NoError(t, err)
	dmc := defaultMockChain(t)
	dmc.Slot = &capellaSlot
	dmc.FinalizedCheckPoint = &ethpb.Checkpoint{Epoch: params.BeaconConfig().CapellaForkEpoch}
	cases := []*blobsTestCase{
		{
			name:    "block before minimum_request_epoch",
			nblocks: 1,
			expired: map[int]bool{0: true},
			chain:   dmc,
			err:     p2pTypes.ErrBlobLTMinRequest,
		},
		{
			name:    "blocks before and after minimum_request_epoch",
			nblocks: 2,
			expired: map[int]bool{0: true},
			chain:   dmc,
			err:     p2pTypes.ErrBlobLTMinRequest,
		},
		{
			name:    "one after minimum_request_epoch then one before",
			nblocks: 2,
			expired: map[int]bool{1: true},
			chain:   dmc,
			err:     p2pTypes.ErrBlobLTMinRequest,
		},
		{
			name:    "one missing index, one after minimum_request_epoch then one before",
			nblocks: 3,
			missing: map[int]map[int]bool{0: map[int]bool{0: true}},
			expired: map[int]bool{1: true},
			chain:   dmc,
			err:     p2pTypes.ErrBlobLTMinRequest,
		},
		{
			name:    "2 missing indices from 2 different blocks",
			nblocks: 3,
			missing: map[int]map[int]bool{0: map[int]bool{0: true}, 2: map[int]bool{3: true}},
			total:   func(i int) *int { return &i }(3*int(params.BeaconConfig().MaxBlobsPerBlock) - 2), // aka 10
		},
		{
			name:    "all indices missing",
			nblocks: 1,
			missing: map[int]map[int]bool{0: map[int]bool{0: true, 1: true, 2: true, 3: true}},
			total:   func(i int) *int { return &i }(0), // aka 10
		},
		{
			name:    "block with all indices missing between 2 full blocks",
			nblocks: 3,
			missing: map[int]map[int]bool{1: map[int]bool{0: true, 1: true, 2: true, 3: true}},
			total:   func(i int) *int { return &i }(8), // aka 10
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.run(t, p2p.RPCBlobSidecarsByRootTopicV1)
		})
	}
}

func TestBlobsByRootOK(t *testing.T) {
	cases := []*blobsTestCase{
		{
			name:    "0 blob",
			nblocks: 0,
		},
		{
			name:    "1 blob",
			nblocks: 1,
		},
		{
			name:    "2 blob",
			nblocks: 2,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.run(t, p2p.RPCBlobSidecarsByRootTopicV1)
		})
	}
}

func TestBlobsByRootMinReqEpoch(t *testing.T) {
	winMin := params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest
	cases := []struct {
		name      string
		finalized types.Epoch
		current   types.Epoch
		deneb     types.Epoch
		expected  types.Epoch
	}{
		{
			name:      "testnet genesis",
			deneb:     100,
			current:   0,
			finalized: 0,
			expected:  100,
		},
		{
			name:      "underflow averted",
			deneb:     100,
			current:   winMin - 1,
			finalized: 0,
			expected:  100,
		},
		{
			name:      "underflow averted - finalized is higher",
			deneb:     100,
			current:   winMin - 1,
			finalized: winMin - 2,
			expected:  winMin - 2,
		},
		{
			name:      "underflow averted - genesis at deneb",
			deneb:     0,
			current:   winMin - 1,
			finalized: 0,
			expected:  0,
		},
		{
			name:      "max is finalized",
			deneb:     100,
			current:   99 + winMin,
			finalized: 101,
			expected:  101,
		},
		{
			name:      "reqWindow > finalized, reqWindow < deneb",
			deneb:     100,
			current:   99 + winMin,
			finalized: 98,
			expected:  100,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := params.BeaconConfig()
			repositionFutureEpochs(cfg)
			cfg.DenebForkEpoch = c.deneb
			undo, err := params.SetActiveWithUndo(cfg)
			require.NoError(t, err)
			defer func() {
				require.NoError(t, undo())
			}()
			ep := blobMinReqEpoch(c.finalized, c.current)
			require.Equal(t, c.expected, ep)
		})
	}
}
