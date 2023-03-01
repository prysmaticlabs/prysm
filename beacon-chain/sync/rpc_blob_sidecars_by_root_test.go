package sync

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	p2pTypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	leakybucket "github.com/prysmaticlabs/prysm/v3/container/leaky-bucket"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/network/forks"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

type sidecarsTestCase struct {
	name          string
	blocks        int
	indices       map[int][]int // allow test to specify indices that are present/missing
	slotOverride  map[int]types.Slot
	expectMissing map[int][]int
	chain         *mock.ChainService
}

func includeIndices(idxs ...int) []bool {
	included := make([]bool, params.BeaconConfig().MaxBlobsPerBlock)
	for _, idx := range idxs {
		if idx > len(included)-1 {
			panic(fmt.Sprintf("invalid includeIndices test setup, included index %d exceeds MaxBlobsPerBlock=%d", idx, len(included)))
		}
		included[idx] = true
	}
	return included
}

func generateTestBlock(t *testing.T, slot types.Slot) *ethpb.SignedBeaconBlockDeneb {
	// Start service with 160 as allowed blocks capacity (and almost zero capacity recovery).
	parent := bytesutil.PadTo([]byte("parentHash"), fieldparams.RootLength)
	stateRoot := bytesutil.PadTo([]byte("stateRoot"), fieldparams.RootLength)
	receiptsRoot := bytesutil.PadTo([]byte("receiptsRoot"), fieldparams.RootLength)
	logsBloom := bytesutil.PadTo([]byte("logs"), fieldparams.LogsBloomLength)
	tx := gethTypes.NewTransaction(
		0,
		common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
		big.NewInt(0), 0, big.NewInt(0),
		nil,
	)
	txs := []*gethTypes.Transaction{tx}
	encodedBinaryTxs := make([][]byte, 1)
	var err error
	encodedBinaryTxs[0], err = txs[0].MarshalBinary()
	require.NoError(t, err)
	blockHash := bytesutil.ToBytes32([]byte("foo"))
	payload := &enginev1.ExecutionPayloadDeneb{
		ParentHash:    parent,
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     stateRoot,
		ReceiptsRoot:  receiptsRoot,
		LogsBloom:     logsBloom,
		PrevRandao:    blockHash[:],
		BlockNumber:   0,
		GasLimit:      0,
		GasUsed:       0,
		Timestamp:     0,
		ExtraData:     make([]byte, 0),
		BaseFeePerGas: bytesutil.PadTo([]byte("baseFeePerGas"), fieldparams.RootLength),
		ExcessDataGas: bytesutil.PadTo([]byte("excessDataGas"), fieldparams.RootLength),
		BlockHash:     blockHash[:],
		Transactions:  encodedBinaryTxs,
	}
	block := util.NewBeaconBlockDeneb()
	block.Block.Body.ExecutionPayload = payload
	block.Block.Slot = slot
	return block
}

func generateTestSidecar(root [32]byte, block *ethpb.SignedBeaconBlockDeneb, index int) *ethpb.BlobSidecar {
	blob := &enginev1.Blob{
		Data: make([]byte, fieldparams.BlobSize),
	}
	binary.LittleEndian.PutUint64(blob.Data, uint64(index))
	return &ethpb.BlobSidecar{
		BlockRoot:       root[:],
		Index:           uint64(index),
		Slot:            block.Block.Slot,
		BlockParentRoot: block.Block.ParentRoot[:],
		ProposerIndex:   block.Block.ProposerIndex,
		Blob:            blob,
		KzgCommitment:   make([]byte, 48),
		KzgProof:        make([]byte, 48),
	}
}

func allIndices() []int {
	idxs := make([]int, params.BeaconConfig().MaxBlobsPerBlock)
	for i := 0; uint64(i) < params.BeaconConfig().MaxBlobsPerBlock; i++ {
		idxs[i] = i
	}
	return idxs
}

func TestSidecarByRootValidation(t *testing.T) {
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
	cases := []sidecarsTestCase{
		{
			name:          "before minimum_request_epoch",
			blocks:        1,
			slotOverride:  map[int]types.Slot{0: capellaSlot},
			expectMissing: map[int][]int{0: allIndices()},
			chain:         dmc,
		},
		{
			name:          "some before minimum_request_epoch",
			blocks:        2,
			slotOverride:  map[int]types.Slot{0: capellaSlot},
			expectMissing: map[int][]int{0: allIndices()},
			chain:         dmc,
		},
	}
	runN(t, cases)
}

func TestSidecarsByRootOK(t *testing.T) {
	cases := []sidecarsTestCase{
		{
			name:   "0 blob",
			blocks: 0,
		},
		{
			name:   "1 blob",
			blocks: 1,
		},
		{
			name:   "2 blob",
			blocks: 2,
		},
	}
	runN(t, cases)
}

func runN(t *testing.T, cases []sidecarsTestCase) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.run(t)
		})
	}
}

// we use max uints for future forks, but this causes overflows when computing slots
// so it is helpful in tests to temporarily reposition the epochs to give room for some math.
func repositionFutureEpochs(cfg *params.BeaconChainConfig) {
	if cfg.CapellaForkEpoch == math.MaxUint64 {
		cfg.CapellaForkEpoch = cfg.BellatrixForkEpoch + 100
	}
	if cfg.DenebForkEpoch == math.MaxUint64 {
		cfg.DenebForkEpoch = cfg.CapellaForkEpoch + 100
	}
}

func defaultMockChain(t *testing.T) *mock.ChainService {
	// TODO: set slot and FinalizedCheckpoint to satisfy conditions
	df, err := forks.Fork(params.BeaconConfig().DenebForkEpoch)
	require.NoError(t, err)
	return &mock.ChainService{ValidatorsRoot: [32]byte{}, Fork: df}
}

func (c sidecarsTestCase) expectedInResponse(num, idx int) bool {
	if missingIdx, ok := c.expectMissing[num]; ok {
		for x := range missingIdx {
			if x == idx {
				return false
			}
		}
		return true
	}
	return true
}

func (c sidecarsTestCase) run(t *testing.T) {
	cfg := params.BeaconConfig()
	repositionFutureEpochs(cfg)
	undo, err := params.SetActiveWithUndo(cfg)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, undo())
	}()
	maxBlobs := int(params.BeaconConfig().MaxBlobsPerBlock)
	if c.chain == nil {
		c.chain = defaultMockChain(t)
	}

	db := &MockBlobDB{}
	var req, expect p2pTypes.BlobSidecarsByRootReq
	de, err := slots.EpochStart(params.BeaconConfig().DenebForkEpoch)
	require.NoError(t, err)
	for i := 0; i < c.blocks; i++ {
		// check if there is a slot override for this index
		// ie to create a block outside the minimum_request_epoch
		bs := de
		if _, ovr := c.slotOverride[i]; ovr {
			bs = c.slotOverride[i]
		}
		block := generateTestBlock(t, bs+types.Slot(i))
		root, err := block.HashTreeRoot()
		require.NoError(t, err)
		binary.LittleEndian.PutUint64(root[:], uint64(i))
		indices, ok := c.indices[i]
		// if specific indices aren't requested, generate them for all
		// an empty list would mean to generate them for none
		if !ok {
			indices = allIndices()
		}
		idxMask := includeIndices(indices...)
		for bi := 0; bi < maxBlobs; bi++ {
			ubi := uint64(bi)
			sc := generateTestSidecar(root, block, bi)
			if idxMask[bi] {
				require.NoError(t, db.WriteBlobSidecar(root, ubi, sc))
				if c.expectedInResponse(i, bi) {
					expect = append(expect, &ethpb.BlobIdentifier{BlockRoot: root[:], Index: ubi})
				}
			}
			req = append(req, &ethpb.BlobIdentifier{BlockRoot: root[:], Index: ubi})
		}
	}
	rate := params.BeaconNetworkConfig().MaxRequestBlobsSidecars * params.BeaconConfig().MaxBlobsPerBlock
	client := p2ptest.NewTestP2P(t)
	s := &Service{
		cfg:         &config{p2p: client, chain: c.chain},
		blobs:       db,
		rateLimiter: newRateLimiter(client)}
	s.setRateCollector(p2p.RPCBlobSidecarsByRootTopicV1, leakybucket.NewCollector(0.000001, int64(rate), time.Second, false))

	rsc := make([]*ethpb.BlobSidecar, 0)
	nh := func(stream network.Stream) {
		if len(expect) == 0 {
			expectFailure(t, responseCodeResourceUnavailable, "", stream)
			return
		}
		for _, sid := range expect {
			expectSuccess(t, stream)
			sc := &ethpb.BlobSidecar{}
			require.NoError(t, s.cfg.p2p.Encoding().DecodeWithMaxLength(stream, sc))
			rsc = append(rsc, sc)
			require.Equal(t, bytesutil.ToBytes32(sc.BlockRoot), bytesutil.ToBytes32(sid.BlockRoot))
			require.Equal(t, sc.Index, sid.Index)
		}
	}
	rht := &rpcHandlerTest{t: t, client: client, topic: p2p.RPCBlocksByRootTopicV1, timeout: time.Second * 10}
	rht.testHandler(nh, s.blobSidecarByRootRPCHandler, &req)
	// The response is a list of BlobSidecar whose length is less than or equal to the number of requests.
	// It may be less in the case that the responding peer is missing blocks or sidecars.
	require.Equal(t, true, len(rsc) <= len(expect))
	for i := range rsc {
		require.Equal(t, bytesutil.ToBytes32(expect[i].BlockRoot), bytesutil.ToBytes32(rsc[i].BlockRoot))
		require.Equal(t, expect[i].Index, rsc[i].Index)
	}
}

type rpcHandlerTest struct {
	t       *testing.T
	client  *p2ptest.TestP2P
	topic   protocol.ID
	timeout time.Duration
}

func (rt *rpcHandlerTest) testHandler(nh network.StreamHandler, rh rpcHandler, rhi interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), rt.timeout)
	defer func() {
		cancel()
	}()

	w := util.NewWaiter()
	server := p2ptest.NewTestP2P(rt.t)
	rt.client.Connect(server)
	defer func() {
		require.NoError(rt.t, rt.client.Disconnect(server.PeerID()))
	}()
	require.Equal(rt.t, 1, len(rt.client.BHost.Network().Peers()), "Expected peers to be connected")
	h := func(stream network.Stream) {
		defer w.Done()
		nh(stream)
	}
	server.BHost.SetStreamHandler(protocol.ID(rt.topic), h)
	stream, err := rt.client.BHost.NewStream(ctx, server.BHost.ID(), protocol.ID(rt.topic))
	require.NoError(rt.t, err)
	require.NoError(rt.t, rh(ctx, rhi, stream))
	w.RequireDoneBeforeCancel(rt.t, ctx)
}
