package sync

import (
	"context"
	"encoding/binary"
	"io"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	ssz "github.com/prysmaticlabs/fastssz"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/encoder"
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
	name    string
	nblocks int                  // how many blocks to loop through in setting up test fixtures & requests
	missing map[int]map[int]bool // skip this blob index, so that we can test different custody scenarios
	expired map[int]bool         // mark block expired to test scenarios where requests are outside retention window
	chain   *mock.ChainService   // allow tests to control retention window via current slot and finalized checkpoint
	total   *int                 // allow a test to specify the total number of responses received
	err     error
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

type expectedResponse struct {
	sidecar *ethpb.BlobSidecar
	code    uint8
	message string
	skipped bool
}

type streamDecoder func(io.Reader, ssz.Unmarshaler) error

func (r *expectedResponse) requireExpected(t *testing.T, d streamDecoder, stream network.Stream) {
	if r.skipped {
		return
	}
	code, _, err := ReadStatusCode(stream, &encoder.SszNetworkEncoder{})
	require.NoError(t, err)
	require.Equal(t, r.code, code, "unexpected response code")
	//require.Equal(t, r.message, msg, "unexpected error message")
	if r.sidecar == nil {
		return
	}
	sc := &ethpb.BlobSidecar{}
	require.NoError(t, d(stream, sc))
	require.Equal(t, bytesutil.ToBytes32(sc.BlockRoot), bytesutil.ToBytes32(r.sidecar.BlockRoot))
	require.Equal(t, sc.Index, r.sidecar.Index)
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
	var req p2pTypes.BlobSidecarsByRootReq
	var expect []*expectedResponse
	oldest, err := slots.EpochStart(minimumRequestEpoch(c.chain.FinalizedCheckPoint.Epoch, slots.ToEpoch(c.chain.CurrentSlot())))
	require.NoError(t, err)
	streamTerminated := false
	for i := 0; i < c.nblocks; i++ {
		// check if there is a slot override for this index
		// ie to create a block outside the minimum_request_epoch
		var bs types.Slot
		if c.expired[i] {
			// the lowest possible bound of the retention period is the deneb epoch, so make sure
			// the slot of an expired block is at least one slot less than the deneb epoch.
			bs = oldest - 1 - types.Slot(i)
		} else {
			bs = oldest + types.Slot(i)
		}
		block := generateTestBlock(t, bs+types.Slot(i))
		root, err := block.HashTreeRoot()
		require.NoError(t, err)
		binary.LittleEndian.PutUint64(root[:], uint64(i))
		for bi := 0; bi < maxBlobs; bi++ {
			ubi := uint64(bi)
			req = append(req, &ethpb.BlobIdentifier{BlockRoot: root[:], Index: ubi})

			if streamTerminated {
				// once we know there is a bad response in the sequence, we want to filter out any subsequent
				// expected responses, because an error response terminates the stream.
				continue
			}
			// skip sidecars that are supposed to be missing
			if missed, ok := c.missing[i]; ok && missed[bi] {
				continue
			}
			sc := generateTestSidecar(root, block, bi)
			require.NoError(t, db.WriteBlobSidecar(root, ubi, sc))
			// if a sidecar is expired, we'll expect an error for the *first* index, and after that
			// we'll expect no further chunks in the stream, so filter out any further expected responses.
			// we don't need to check what index this is because we work through them in order and the first one
			// will set streamTerminated = true and skip everything else in the test case.
			if c.expired[i] {
				expect = append(expect, &expectedResponse{
					code:    responseCodeResourceUnavailable,
					message: p2pTypes.ErrBlobLTMinRequest.Error(),
				})
				streamTerminated = true
				continue
			}

			expect = append(expect, &expectedResponse{
				sidecar: sc,
				code:    responseCodeSuccess,
				message: "",
			})
		}
	}
	rate := params.BeaconNetworkConfig().MaxRequestBlobsSidecars * params.BeaconConfig().MaxBlobsPerBlock
	client := p2ptest.NewTestP2P(t)
	s := &Service{
		cfg:         &config{p2p: client, chain: c.chain},
		blobs:       db,
		rateLimiter: newRateLimiter(client)}
	s.setRateCollector(p2p.RPCBlobSidecarsByRootTopicV1, leakybucket.NewCollector(0.000001, int64(rate), time.Second, false))

	dec := s.cfg.p2p.Encoding().DecodeWithMaxLength
	if c.total != nil {
		require.Equal(t, *c.total, len(expect))
	}
	nh := func(stream network.Stream) {
		for _, ex := range expect {
			ex.requireExpected(t, dec, stream)
		}
	}
	rht := &rpcHandlerTest{t: t, client: client, topic: p2p.RPCBlocksByRootTopicV1, timeout: time.Second * 10, err: c.err}
	rht.testHandler(nh, s.blobSidecarByRootRPCHandler, &req)
}

type rpcHandlerTest struct {
	t       *testing.T
	client  *p2ptest.TestP2P
	topic   protocol.ID
	timeout time.Duration
	err     error
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

	err = rh(ctx, rhi, stream)
	if rt.err == nil {
		require.NoError(rt.t, err)
	} else {
		require.ErrorIs(rt.t, err, rt.err)
	}

	w.RequireDoneBeforeCancel(rt.t, ctx)
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
	df, err := forks.Fork(params.BeaconConfig().DenebForkEpoch)
	require.NoError(t, err)
	ce := params.BeaconConfig().DenebForkEpoch + params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest
	fe := ce - 2
	cs, err := slots.EpochStart(ce)
	require.NoError(t, err)

	return &mock.ChainService{
		ValidatorsRoot:      [32]byte{},
		Slot:                &cs,
		FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: fe},
		Fork:                df}
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
			c.run(t)
		})
	}
}

func TestSidecarsByRootOK(t *testing.T) {
	cases := []sidecarsTestCase{
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
			c.run(t)
		})
	}
}
