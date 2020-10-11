package beaconv1

import (
	"context"
	"reflect"
	"testing"

	ethereum_beacon_p2p_v1 "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

	"github.com/golang/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	mockPOW "github.com/prysmaticlabs/prysm/beacon-chain/powchain/testing"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func fillDBTestBlocks(t *testing.T, ctx context.Context, db db.Database) (*ethpb_alpha.SignedBeaconBlock, []*ethpb_alpha.BeaconBlockContainer) {
	// Should return the proper genesis block if it exists.
	parentRoot := [32]byte{1, 2, 3}
	genBlk := testutil.NewBeaconBlock()
	genBlk.Block.ParentRoot = parentRoot[:]
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, genBlk))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

	count := uint64(100)
	blks := make([]*ethpb_alpha.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb_alpha.BeaconBlockContainer, count)
	for i := uint64(0); i < count; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte{uint8(i) % 8}, 32)
		att1 := testutil.NewAttestation()
		att1.Data.Slot = i
		att1.Data.CommitteeIndex = i
		att2 := testutil.NewAttestation()
		att2.Data.Slot = i
		att2.Data.CommitteeIndex = i + 1
		b.Block.Body.Attestations = []*ethpb_alpha.Attestation{att1, att2}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i] = b
		blkContainers[i] = &ethpb_alpha.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
	}
	require.NoError(t, db.SaveBlocks(ctx, blks))
	headRoot := bytesutil.ToBytes32(blkContainers[len(blks)-1].BlockRoot)
	summary := &ethereum_beacon_p2p_v1.StateSummary{
		Root: headRoot[:],
		Slot: blkContainers[len(blks)-1].Block.Block.Slot,
	}
	require.NoError(t, db.SaveStateSummary(ctx, summary))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, headRoot))
	return genBlk, blkContainers
}

func TestServer_GetBlockHeader_Slot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	_, blkContainers := fillDBTestBlocks(t, ctx, db)

	header, err := bs.GetBlockHeader(ctx, &ethpb.BlockRequest{
		BlockId: bytesutil.ToBytes(30, 8),
	})
	require.NoError(t, err)

	blkHdr, err := blockutil.SignedBeaconBlockHeaderFromBlock(blkContainers[30].Block)
	require.NoError(t, err)
	marshaledBlkHdr, err := blkHdr.Marshal()
	require.NoError(t, err)

	v1BlockHdr := &ethpb.SignedBeaconBlockHeader{}
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(marshaledBlkHdr, v1BlockHdr))

	if !reflect.DeepEqual(header.Data.Header.Message, v1BlockHdr.Header) {
		t.Error("Expected blocks to equal")
	}
}

func TestServer_GetBlockHeader_Slot_Empty(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	_, _ = fillDBTestBlocks(t, ctx, db)

	// Should throw an error no block is found.
	_, err := bs.GetBlockHeader(ctx, &ethpb.BlockRequest{
		BlockId: bytesutil.ToBytes(105, 8),
	})
	require.ErrorContains(t, "Could not find", err)
}

func TestServer_ListBlockHeaders_Slot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	_, blkContainers := fillDBTestBlocks(t, ctx, db)
	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.Body.Graffiti = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, db.SaveBlock(ctx, b2))
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.Body.Graffiti = bytesutil.PadTo([]byte{3}, 32)
	require.NoError(t, db.SaveBlock(ctx, b3))

	// Should throw an error if more than one blk returned.
	headers, err := bs.ListBlockHeaders(ctx, &ethpb.BlockHeadersRequest{
		Slot: 30,
	})
	require.NoError(t, err)

	blocks := []*ethpb_alpha.SignedBeaconBlock{blkContainers[30].Block, b2, b3}
	require.Equal(t, len(blocks), len(headers.Data))
	for i, blk := range blocks {
		signedHdr, err := blockutil.SignedBeaconBlockHeaderFromBlock(blk)
		require.NoError(t, err)
		marshaledBlkHdr, err := signedHdr.Marshal()
		require.NoError(t, err)
		v1alpa1BlockHdr := &ethpb_alpha.SignedBeaconBlockHeader{}
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(marshaledBlkHdr, v1alpa1BlockHdr))
		v1BlockHdr := &ethpb.SignedBeaconBlockHeader{}
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(marshaledBlkHdr, v1BlockHdr))

		if !reflect.DeepEqual(headers.Data[i].Header.Message, v1BlockHdr.Header) {
			t.Error("Expected blocks to equal")
		}
	}
}

func TestServer_ListBlockHeaders_ParentRoot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	parentRoot := bytesutil.PadTo([]byte{5, 6}, 32)
	_, _ = fillDBTestBlocks(t, ctx, db)
	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 31
	b2.Block.ParentRoot = parentRoot
	require.NoError(t, db.SaveBlock(ctx, b2))
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 32
	b3.Block.ParentRoot = parentRoot
	require.NoError(t, db.SaveBlock(ctx, b3))
	b4 := testutil.NewBeaconBlock()
	b4.Block.Slot = 36
	b4.Block.ParentRoot = parentRoot
	require.NoError(t, db.SaveBlock(ctx, b4))

	// Should throw an error if more than one blk returned.
	headers, err := bs.ListBlockHeaders(ctx, &ethpb.BlockHeadersRequest{
		ParentRoot: parentRoot,
	})
	require.NoError(t, err)

	blocks := []*ethpb_alpha.SignedBeaconBlock{b2, b3, b4}
	require.Equal(t, len(blocks), len(headers.Data))
	for i, blk := range blocks {
		signedHdr, err := blockutil.SignedBeaconBlockHeaderFromBlock(blk)
		require.NoError(t, err)
		marshaledBlkHdr, err := signedHdr.Marshal()
		require.NoError(t, err)
		v1alpa1BlockHdr := &ethpb_alpha.SignedBeaconBlockHeader{}
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(marshaledBlkHdr, v1alpa1BlockHdr))
		v1BlockHdr := &ethpb.SignedBeaconBlockHeader{}
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(marshaledBlkHdr, v1BlockHdr))

		if !reflect.DeepEqual(headers.Data[i].Header.Message, v1BlockHdr.Header) {
			t.Error("Expected blocks to equal")
		}
	}
}

func TestServer_ProposeBlock_OK(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	genesis := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(context.Background(), genesis), "Could not save genesis block")

	numDeposits := uint64(64)
	beaconState, _ := testutil.DeterministicGenesisState(t, numDeposits)
	bsRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

	c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
	beaconChainServer := &Server{
		BeaconDB:          db,
		ChainStartFetcher: &mockPOW.POWChain{},
		BlockReceiver:     c,
		HeadFetcher:       c,
		BlockNotifier:     c.BlockNotifier(),
		Broadcaster:       mockp2p.NewTestP2P(t),
	}
	req := testutil.NewBeaconBlock()
	req.Block.Slot = 5
	req.Block.ParentRoot = bsRoot[:]
	v1Block, err := v1alpha1ToV1Block(req)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, req))
	blockReq := &ethpb.BeaconBlockContainer{
		Message:   v1Block.Block,
		Signature: v1Block.Signature,
	}
	_, err = beaconChainServer.SubmitBlock(context.Background(), blockReq)
	assert.NoError(t, err, "Could not propose block correctly")
}

func TestServer_GetBlock_GenesisRoot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}

	genBlk, _ := fillDBTestBlocks(t, ctx, db)
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)

	// Should throw an error if more than one blk returned.
	block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
		BlockId: root[:],
	})
	require.NoError(t, err)

	marshaledBlk, err := genBlk.Block.Marshal()
	require.NoError(t, err)
	v1Block := &ethpb.BeaconBlock{}
	require.NoError(t, proto.Unmarshal(marshaledBlk, v1Block))

	if !reflect.DeepEqual(block.Data.Message, v1Block) {
		t.Error("Expected blocks to equal")
	}
}

func TestServer_GetBlock_Genesis(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}

	genBlk, _ := fillDBTestBlocks(t, ctx, db)

	// Should throw an error if more than one blk returned.
	block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
		BlockId: []byte("genesis"),
	})
	require.NoError(t, err)

	marshaledBlk, err := genBlk.Block.Marshal()
	require.NoError(t, err)
	v1Block := &ethpb.BeaconBlock{}
	require.NoError(t, proto.Unmarshal(marshaledBlk, v1Block))

	if !reflect.DeepEqual(block.Data.Message, v1Block) {
		t.Error("Expected blocks to equal")
	}
}

func TestServer_GetBlock_Head(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}

	_, blkContainers := fillDBTestBlocks(t, ctx, db)

	headBlock := blkContainers[len(blkContainers)-1]
	bs = &Server{
		BeaconDB:    db,
		HeadFetcher: &mock.ChainService{DB: db, Block: headBlock.Block},
	}
	// Should throw an error if more than one blk returned.
	block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
		BlockId: []byte("head"),
	})
	require.NoError(t, err)

	marshaledBlk, err := headBlock.Block.Block.Marshal()
	require.NoError(t, err)
	v1Block := &ethpb.BeaconBlock{}
	require.NoError(t, proto.Unmarshal(marshaledBlk, v1Block))

	if !reflect.DeepEqual(block.Data.Message, v1Block) {
		t.Error("Expected blocks to equal")
	}
}

func TestServer_GetBlock_Root(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	_, blkContainers := fillDBTestBlocks(t, ctx, db)

	// Should throw an error if more than one blk returned.
	block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
		BlockId: blkContainers[50].BlockRoot,
	})
	require.NoError(t, err)

	marshaledBlk, err := blkContainers[50].Block.Block.Marshal()
	require.NoError(t, err)
	v1Block := &ethpb.BeaconBlock{}
	require.NoError(t, proto.Unmarshal(marshaledBlk, v1Block))

	if !reflect.DeepEqual(block.Data.Message, v1Block) {
		t.Error("Expected blocks to equal")
	}
}

func TestServer_GetBlock_Slot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	_, blkContainers := fillDBTestBlocks(t, ctx, db)

	// Should throw an error if more than one blk returned.
	block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
		BlockId: bytesutil.ToBytes(40, 8),
	})
	require.NoError(t, err)

	marshaledBlk, err := blkContainers[40].Block.Block.Marshal()
	require.NoError(t, err)
	v1Block := &ethpb.BeaconBlock{}
	require.NoError(t, proto.Unmarshal(marshaledBlk, v1Block))

	if !reflect.DeepEqual(block.Data.Message, v1Block) {
		t.Error("Expected blocks to equal")
	}
}

func TestServer_GetBlockAttestations_Slot(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	bs := &Server{
		BeaconDB: db,
	}
	_, blkContainers := fillDBTestBlocks(t, ctx, db)

	// Should throw an error if more than one blk returned.
	block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
		BlockId: bytesutil.ToBytes(40, 8),
	})
	require.NoError(t, err)

	marshaledBlk, err := blkContainers[40].Block.Block.Marshal()
	require.NoError(t, err)
	v1Block := &ethpb.BeaconBlock{}
	require.NoError(t, proto.Unmarshal(marshaledBlk, v1Block))

	if !reflect.DeepEqual(block.Data.Message, v1Block) {
		t.Error("Expected blocks to equal")
	}
}
