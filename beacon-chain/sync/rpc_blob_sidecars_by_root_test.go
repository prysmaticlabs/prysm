package sync

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	p2pTypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func (c *blobsTestCase) defaultOldestSlotByRoot(t *testing.T) types.Slot {
	oldest, err := slots.EpochStart(blobMinReqEpoch(c.chain.FinalizedCheckPoint.Epoch, slots.ToEpoch(c.chain.CurrentSlot())))
	require.NoError(t, err)
	return oldest
}

func blobRootRequestFromSidecars(scs []*ethpb.BlobSidecar) interface{} {
	req := make(p2pTypes.BlobSidecarsByRootReq, 0)
	for _, sc := range scs {
		req = append(req, &ethpb.BlobIdentifier{BlockRoot: sc.BlockRoot, Index: sc.Index})
	}
	return &req
}

func (c *blobsTestCase) filterExpectedByRoot(t *testing.T, scs []*ethpb.BlobSidecar, _ interface{}) []*expectedBlobChunk {
	var expect []*expectedBlobChunk
	blockOffset := 0
	if len(scs) == 0 {
		return expect
	}
	lastRoot := bytesutil.ToBytes32(scs[0].BlockRoot)
	for _, sc := range scs {
		root := bytesutil.ToBytes32(sc.BlockRoot)
		if root != lastRoot {
			blockOffset += 1
		}
		lastRoot = root

		// skip sidecars that are supposed to be missing
		if missed, ok := c.missing[blockOffset]; ok && missed[int(sc.Index)] {
			continue
		}
		// if a sidecar is expired, we'll expect an error for the *first* index, and after that
		// we'll expect no further chunks in the stream, so filter out any further expected responses.
		// we don't need to check what index this is because we work through them in order and the first one
		// will set streamTerminated = true and skip everything else in the test case.
		if c.expired[blockOffset] {
			return append(expect, &expectedBlobChunk{
				sidecar: sc,
				code:    responseCodeResourceUnavailable,
				message: p2pTypes.ErrBlobLTMinRequest.Error(),
			})
		}

		expect = append(expect, &expectedBlobChunk{
			sidecar: sc,
			code:    responseCodeSuccess,
			message: "",
		})
	}
	return expect
}

func (c *blobsTestCase) runTestBlobSidecarsByRoot(t *testing.T) {
	if c.serverHandle == nil {
		c.serverHandle = func(s *Service) rpcHandler { return s.blobSidecarByRootRPCHandler }
	}
	if c.defineExpected == nil {
		c.defineExpected = c.filterExpectedByRoot
	}
	if c.requestFromSidecars == nil {
		c.requestFromSidecars = blobRootRequestFromSidecars
	}
	if c.topic == "" {
		c.topic = p2p.RPCBlobSidecarsByRootTopicV1
	}
	if c.oldestSlot == nil {
		c.oldestSlot = c.defaultOldestSlotByRoot
	}
	if c.streamReader == nil {
		c.streamReader = defaultExpectedRequirer
	}
	c.run(t)
}

func TestReadChunkEncodedBlobs(t *testing.T) {
	dmc := defaultMockChain(t)
	cases := []*blobsTestCase{
		{
			name:         "test successful read via requester",
			nblocks:      1,
			chain:        dmc,
			streamReader: readChunkEncodedBlobsAsStreamReader,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.runTestBlobSidecarsByRoot(t)
		})
	}
}

func readChunkEncodedBlobsAsStreamReader(t *testing.T, s *Service, expect []*expectedBlobChunk) func(network.Stream) {
	ff := s.cfg.chain
	encoding := s.cfg.p2p.Encoding()
	return func(stream network.Stream) {
		scs, err := readChunkEncodedBlobs(stream, ff, encoding)
		require.NoError(t, err)
		require.Equal(t, len(expect), len(scs))
		for i, sc := range scs {
			esc := expect[i].sidecar
			require.Equal(t, esc.Slot, sc.Slot)
			require.Equal(t, esc.Index, sc.Index)
			require.Equal(t, bytesutil.ToBytes32(esc.BlockRoot), bytesutil.ToBytes32(sc.BlockRoot))
		}
	}
}

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
			total:   func(i int) *int { return &i }(0),
		},
		{
			name:    "block with all indices missing between 2 full blocks",
			nblocks: 3,
			missing: map[int]map[int]bool{1: map[int]bool{0: true, 1: true, 2: true, 3: true}},
			total:   func(i int) *int { return &i }(8),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.runTestBlobSidecarsByRoot(t)
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
			c.runTestBlobSidecarsByRoot(t)
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
