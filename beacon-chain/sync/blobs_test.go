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
	db "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
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

type blobsTestCase struct {
	name    string
	nblocks int                  // how many blocks to loop through in setting up test fixtures & requests
	missing map[int]map[int]bool // skip this blob index, so that we can test different custody scenarios
	expired map[int]bool         // mark block expired to test scenarios where requests are outside retention window
	chain   *mock.ChainService   // allow tests to control retention window via current slot and finalized checkpoint
	total   *int                 // allow a test to specify the total number of responses received
	err     error
}

func generateTestBlockWithSidecars(t *testing.T, parent [32]byte, slot types.Slot, nblobs int) (*ethpb.SignedBeaconBlockDeneb, []*ethpb.BlobSidecar) {
	// Start service with 160 as allowed blocks capacity (and almost zero capacity recovery).
	stateRoot := bytesutil.PadTo([]byte("stateRoot"), fieldparams.RootLength)
	receiptsRoot := bytesutil.PadTo([]byte("receiptsRoot"), fieldparams.RootLength)
	logsBloom := bytesutil.PadTo([]byte("logs"), fieldparams.LogsBloomLength)
	parentHash := bytesutil.PadTo([]byte("parentHash"), fieldparams.RootLength)
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
		ParentHash:    parentHash,
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
	block.Block.ParentRoot = parent[:]
	commitments := make([][48]byte, nblobs)
	block.Block.Body.BlobKzgCommitments = make([][]byte, nblobs)
	for i := range commitments {
		binary.LittleEndian.PutUint64(commitments[i][:], uint64(i))
		block.Block.Body.BlobKzgCommitments[i] = commitments[i][:]
	}

	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)

	sidecars := make([]*ethpb.BlobSidecar, len(commitments))
	for i, c := range block.Block.Body.BlobKzgCommitments {
		sidecars[i] = generateTestSidecar(root, block, i, c)
	}
	return block, sidecars
}

func generateTestBlock(t *testing.T, parent [32]byte, slot types.Slot) *ethpb.SignedBeaconBlockDeneb {
	// Start service with 160 as allowed blocks capacity (and almost zero capacity recovery).
	stateRoot := bytesutil.PadTo([]byte("stateRoot"), fieldparams.RootLength)
	receiptsRoot := bytesutil.PadTo([]byte("receiptsRoot"), fieldparams.RootLength)
	logsBloom := bytesutil.PadTo([]byte("logs"), fieldparams.LogsBloomLength)
	parentHash := bytesutil.PadTo([]byte("parentHash"), fieldparams.RootLength)
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
		ParentHash:    parentHash,
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
	block.Block.Body.BlobKzgCommitments = make([][]byte, 0)
	block.Block.ParentRoot = parent[:]
	return block
}

func generateTestSidecar(root [32]byte, block *ethpb.SignedBeaconBlockDeneb, index int, commitment []byte) *ethpb.BlobSidecar {
	blob := &enginev1.Blob{
		Data: make([]byte, fieldparams.BlobSize),
	}
	binary.LittleEndian.PutUint64(blob.Data, uint64(index))
	sc := &ethpb.BlobSidecar{
		BlockRoot:       root[:],
		Index:           uint64(index),
		Slot:            block.Block.Slot,
		BlockParentRoot: block.Block.ParentRoot,
		ProposerIndex:   block.Block.ProposerIndex,
		Blob:            blob,
		KzgCommitment:   commitment[:],
		KzgProof:        commitment[:],
	}
	return sc
}

type blobsByRootExpected struct {
	code    uint8
	sidecar *ethpb.BlobSidecar
	message string
}

type streamDecoder func(io.Reader, ssz.Unmarshaler) error

func (r *blobsByRootExpected) requireExpected(t *testing.T, s *Service, stream network.Stream) {
	d := s.cfg.p2p.Encoding().DecodeWithMaxLength

	code, _, err := ReadStatusCode(stream, &encoder.SszNetworkEncoder{})
	require.NoError(t, err)
	require.Equal(t, r.code, code, "unexpected response code")
	if r.sidecar == nil {
		return
	}

	c, err := readContextFromStream(stream, s.cfg.chain)
	require.NoError(t, err)

	valRoot := s.cfg.chain.GenesisValidatorsRoot()
	ctxBytes, err := forks.ForkDigestFromEpoch(slots.ToEpoch(r.sidecar.GetSlot()), valRoot[:])
	require.NoError(t, err)
	require.Equal(t, ctxBytes, bytesutil.ToBytes4(c))

	//require.Equal(t, r.message, msg, "unexpected error message")
	sc := &ethpb.BlobSidecar{}
	require.NoError(t, d(stream, sc))
	require.Equal(t, bytesutil.ToBytes32(sc.BlockRoot), bytesutil.ToBytes32(r.sidecar.BlockRoot))
	require.Equal(t, sc.Index, r.sidecar.Index)
}

func (c *blobsTestCase) setup(t *testing.T) (*Service, []*ethpb.BlobSidecar, []*blobsByRootExpected, func()) {
	cfg := params.BeaconConfig()
	repositionFutureEpochs(cfg)
	undo, err := params.SetActiveWithUndo(cfg)
	require.NoError(t, err)
	cleanup := func() {
		require.NoError(t, undo())
	}
	maxBlobs := int(params.BeaconConfig().MaxBlobsPerBlock)
	if c.chain == nil {
		c.chain = defaultMockChain(t)
	}
	d := db.SetupDB(t)

	bdb := &MockBlobDB{}
	sidecars := make([]*ethpb.BlobSidecar, 0)
	var expect []*blobsByRootExpected
	oldest, err := slots.EpochStart(blobMinReqEpoch(c.chain.FinalizedCheckPoint.Epoch, slots.ToEpoch(c.chain.CurrentSlot())))
	require.NoError(t, err)
	streamTerminated := false
	var parentRoot [32]byte
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
		block, bsc := generateTestBlockWithSidecars(t, parentRoot, bs, maxBlobs)
		root, err := block.Block.HashTreeRoot()
		require.NoError(t, err)
		for bi, sc := range bsc {
			ubi := uint64(bi)
			sidecars = append(sidecars, sc)

			if streamTerminated {
				// once we know there is a bad response in the sequence, we want to filter out any subsequent
				// expected responses, because an error response terminates the stream.
				continue
			}
			// skip sidecars that are supposed to be missing
			if missed, ok := c.missing[i]; ok && missed[bi] {
				continue
			}
			require.NoError(t, bdb.WriteBlobSidecar(root, ubi, sc))
			// if a sidecar is expired, we'll expect an error for the *first* index, and after that
			// we'll expect no further chunks in the stream, so filter out any further expected responses.
			// we don't need to check what index this is because we work through them in order and the first one
			// will set streamTerminated = true and skip everything else in the test case.
			if c.expired[i] {
				expect = append(expect, &blobsByRootExpected{
					code:    responseCodeResourceUnavailable,
					message: p2pTypes.ErrBlobLTMinRequest.Error(),
				})
				streamTerminated = true
				continue
			}

			expect = append(expect, &blobsByRootExpected{
				sidecar: sc,
				code:    responseCodeSuccess,
				message: "",
			})
		}
		util.SaveBlock(t, context.Background(), d, block)
		parentRoot = root
	}

	client := p2ptest.NewTestP2P(t)
	s := &Service{
		cfg:         &config{p2p: client, chain: c.chain, beaconDB: d},
		blobs:       bdb,
		rateLimiter: newRateLimiter(client)}
	//s.registerRPCHandlersDeneb()

	byRootRate := params.BeaconNetworkConfig().MaxRequestBlobsSidecars * params.BeaconConfig().MaxBlobsPerBlock
	byRangeRate := params.BeaconNetworkConfig().MaxRequestBlobsSidecars * params.BeaconConfig().MaxBlobsPerBlock
	s.setRateCollector(p2p.RPCBlobSidecarsByRootTopicV1, leakybucket.NewCollector(0.000001, int64(byRootRate), time.Second, false))
	s.setRateCollector(p2p.RPCBlobSidecarsByRangeTopicV1, leakybucket.NewCollector(0.000001, int64(byRangeRate), time.Second, false))

	return s, sidecars, expect, cleanup
}

func (c *blobsTestCase) run(t *testing.T, topic protocol.ID) {
	s, sidecars, expect, cleanup := c.setup(t)
	defer cleanup()

	if c.total != nil {
		require.Equal(t, *c.total, len(expect))
	}
	nh := func(stream network.Stream) {
		for _, ex := range expect {
			ex.requireExpected(t, s, stream)
		}
	}
	client := s.cfg.p2p.(*p2ptest.TestP2P)
	rht := &rpcHandlerTest{t: t, client: client, topic: topic, timeout: time.Second * 10, err: c.err}
	switch topic {
	case p2p.RPCBlobSidecarsByRootTopicV1:
		req := blobRootRequestFromSidecars(sidecars)
		rht.testHandler(nh, s.blobSidecarByRootRPCHandler, req)
	case p2p.RPCBlobSidecarsByRangeTopicV1:
		req := blobRangeRequestFromSidecars(sidecars)
		rht.testHandler(nh, s.blobSidecarsByRangeRPCHandler, req)
	}
}

func blobRangeRequestFromSidecars(scs []*ethpb.BlobSidecar) *ethpb.BlobSidecarsByRangeRequest {
	maxBlobs := params.BeaconConfig().MaxBlobsPerBlock
	count := uint64(len(scs)) / maxBlobs
	return &ethpb.BlobSidecarsByRangeRequest{
		StartSlot: scs[0].Slot,
		Count:     count,
	}
}

func blobRootRequestFromSidecars(scs []*ethpb.BlobSidecar) *p2pTypes.BlobSidecarsByRootReq {
	req := make(p2pTypes.BlobSidecarsByRootReq, 0)
	for _, sc := range scs {
		req = append(req, &ethpb.BlobIdentifier{BlockRoot: sc.BlockRoot, Index: sc.Index})
	}
	return &req
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
	ce := params.BeaconConfig().DenebForkEpoch + params.BeaconNetworkConfig().MinEpochsForBlobsSidecarsRequest + 1000
	fe := ce - 2
	cs, err := slots.EpochStart(ce)
	require.NoError(t, err)

	return &mock.ChainService{
		ValidatorsRoot:      [32]byte{},
		Slot:                &cs,
		FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: fe},
		Fork:                df}
}

func TestTestcaseSetup_BlocksAndBlobs(t *testing.T) {
	ctx := context.Background()
	c := blobsTestCase{nblocks: 10}
	s, sidecars, expect, cleanup := c.setup(t)
	defer cleanup()
	require.Equal(t, 40, len(sidecars))
	require.Equal(t, 40, len(expect))
	for _, sc := range sidecars {
		blk, err := s.cfg.beaconDB.Block(ctx, bytesutil.ToBytes32(sc.BlockRoot))
		require.NoError(t, err)
		var found *int
		comms, err := blk.Block().Body().BlobKzgCommitments()
		require.NoError(t, err)
		for i, cm := range comms {
			if bytesutil.ToBytes48(sc.KzgCommitment) == bytesutil.ToBytes48(cm) {
				found = &i
			}
		}
		require.Equal(t, true, found != nil)
	}
}

func TestRoundTripDenebSave(t *testing.T) {
	ctx := context.Background()
	cfg := params.BeaconConfig()
	repositionFutureEpochs(cfg)
	undo, err := params.SetActiveWithUndo(cfg)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, undo())
	}()
	parentRoot := [32]byte{}
	c := blobsTestCase{nblocks: 10}
	c.chain = defaultMockChain(t)
	oldest, err := slots.EpochStart(blobMinReqEpoch(c.chain.FinalizedCheckPoint.Epoch, slots.ToEpoch(c.chain.CurrentSlot())))
	require.NoError(t, err)
	maxBlobs := int(params.BeaconConfig().MaxBlobsPerBlock)
	block, bsc := generateTestBlockWithSidecars(t, parentRoot, oldest, maxBlobs)
	require.Equal(t, len(block.Block.Body.BlobKzgCommitments), len(bsc))
	require.Equal(t, maxBlobs, len(bsc))
	for i := range bsc {
		require.DeepEqual(t, block.Block.Body.BlobKzgCommitments[i], bsc[i].KzgCommitment)
	}
	d := db.SetupDB(t)
	util.SaveBlock(t, ctx, d, block)
	root, err := block.Block.HashTreeRoot()
	require.NoError(t, err)
	dbBlock, err := d.Block(ctx, root)
	require.NoError(t, err)
	comms, err := dbBlock.Block().Body().BlobKzgCommitments()
	require.NoError(t, err)
	require.Equal(t, maxBlobs, len(comms))
	for i := range bsc {
		require.DeepEqual(t, comms[i], bsc[i].KzgCommitment)
	}
}
