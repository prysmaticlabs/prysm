package sync

import (
	"bytes"
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/protolambda/zssz"
	"github.com/protolambda/zssz/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestRecentBeaconBlocksRPCHandler_ReturnsBlocks(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
	d, _ := db.SetupDB(t)

	var blkRoots [][32]byte
	// Populate the database with blocks that would match the request.
	for i := 1; i < 11; i++ {
		blk := &ethpb.BeaconBlock{
			Slot: uint64(i),
		}
		root, err := ssz.HashTreeRoot(blk)
		require.NoError(t, err)
		require.NoError(t, d.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: blk}))
		blkRoots = append(blkRoots, root)
	}

	r := &Service{p2p: p1, db: d, rateLimiter: newRateLimiter(p1)}
	pcl := protocol.ID("/testing")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(10000, 10000, false)

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		for i := range blkRoots {
			expectSuccess(t, r, stream)
			res := &ethpb.SignedBeaconBlock{}
			assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, &res))
			if res.Block.Slot != uint64(i+1) {
				t.Errorf("Received unexpected block slot %d but wanted %d", res.Block.Slot, i+1)
			}
		}
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	require.NoError(t, err)
	err = r.beaconBlocksRootRPCHandler(context.Background(), blkRoots, stream1)
	assert.NoError(t, err)

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}

func TestRecentBeaconBlocks_RPCRequestSent(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.DelaySend = true

	blockA := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 111}}
	blockB := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 40}}
	// Set up a head state with data we expect.
	blockARoot, err := stateutil.BlockRoot(blockA.Block)
	require.NoError(t, err)
	blockBRoot, err := stateutil.BlockRoot(blockB.Block)
	require.NoError(t, err)
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	require.NoError(t, err)
	require.NoError(t, genesisState.SetSlot(111))
	require.NoError(t, genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, blockARoot))
	finalizedCheckpt := &ethpb.Checkpoint{
		Epoch: 5,
		Root:  blockBRoot[:],
	}

	expectedRoots := [][32]byte{blockBRoot, blockARoot}

	r := &Service{
		p2p: p1,
		chain: &mock.ChainService{
			State:               genesisState,
			FinalizedCheckPoint: finalizedCheckpt,
			Root:                blockARoot[:],
		},
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
		ctx:                 context.Background(),
		rateLimiter:         newRateLimiter(p1),
	}

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/beacon_blocks_by_root/1/ssz_snappy")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(10000, 10000, false)

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := [][32]byte{}
		assert.NoError(t, p2.Encoding().DecodeWithMaxLength(stream, &out))
		assert.DeepEqual(t, expectedRoots, out, "Did not receive expected message")
		response := []*ethpb.SignedBeaconBlock{blockB, blockA}
		for _, blk := range response {
			_, err := stream.Write([]byte{responseCodeSuccess})
			assert.NoError(t, err, "Failed to write to stream")
			_, err = p2.Encoding().EncodeWithMaxLength(stream, blk)
			assert.NoError(t, err, "Could not send response back")
		}
	})

	p1.Connect(p2)
	require.NoError(t, r.sendRecentBeaconBlocksRequest(context.Background(), expectedRoots, p2.PeerID()))

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}

type testList [][32]byte

func (*testList) Limit() uint64 {
	return 2 << 10
}

func TestSSZCompatibility(t *testing.T) {
	rootA := [32]byte{'a'}
	rootB := [32]byte{'B'}
	rootC := [32]byte{'C'}
	list := testList{rootA, rootB, rootC}
	writer := bytes.NewBuffer([]byte{})
	sszType, err := types.SSZFactory(reflect.TypeOf(list))
	assert.NoError(t, err)
	n, err := zssz.Encode(writer, list, sszType)
	assert.NoError(t, err)
	encodedPart := writer.Bytes()[:n]
	fastSSZ, err := ssz.Marshal(list)
	assert.NoError(t, err)
	if !bytes.Equal(fastSSZ, encodedPart) {
		t.Errorf("Wanted the same result as ZSSZ of %#x but got %#X", encodedPart, fastSSZ)
	}
}
