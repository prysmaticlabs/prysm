package beacon

import (
	"context"
	"reflect"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/proto/interfaces"
	"github.com/prysmaticlabs/prysm/proto/migration"
	ethpb_alpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func fillDBTestBlocks(ctx context.Context, t *testing.T, beaconDB db.Database) (*ethpb_alpha.SignedBeaconBlock, []*ethpb_alpha.BeaconBlockContainer) {
	parentRoot := [32]byte{1, 2, 3}
	genBlk := testutil.NewBeaconBlock()
	genBlk.Block.ParentRoot = parentRoot[:]
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genBlk)))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, root))

	count := types.Slot(100)
	blks := make([]interfaces.SignedBeaconBlock, count)
	blkContainers := make([]*ethpb_alpha.BeaconBlockContainer, count)
	for i := types.Slot(0); i < count; i++ {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = i
		b.Block.ParentRoot = bytesutil.PadTo([]byte{uint8(i)}, 32)
		att1 := testutil.NewAttestation()
		att1.Data.Slot = i
		att1.Data.CommitteeIndex = types.CommitteeIndex(i)
		att2 := testutil.NewAttestation()
		att2.Data.Slot = i
		att2.Data.CommitteeIndex = types.CommitteeIndex(i + 1)
		b.Block.Body.Attestations = []*ethpb_alpha.Attestation{att1, att2}
		root, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		blks[i] = wrapper.WrappedPhase0SignedBeaconBlock(b)
		blkContainers[i] = &ethpb_alpha.BeaconBlockContainer{Block: b, BlockRoot: root[:]}
	}
	require.NoError(t, beaconDB.SaveBlocks(ctx, blks))
	headRoot := bytesutil.ToBytes32(blkContainers[len(blks)-1].BlockRoot)
	summary := &p2ppb.StateSummary{
		Root: headRoot[:],
		Slot: blkContainers[len(blks)-1].Block.Block.Slot,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, summary))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, headRoot))
	return genBlk, blkContainers
}

func TestServer_GetBlockHeader(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	genBlk, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	headBlock := blkContainers[len(blkContainers)-1]

	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))

	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
	}

	tests := []struct {
		name    string
		blockID []byte
		want    *ethpb_alpha.SignedBeaconBlock
		wantErr bool
	}{
		{
			name:    "slot",
			blockID: []byte("30"),
			want:    blkContainers[30].Block,
		},
		{
			name:    "root",
			blockID: blkContainers[20].BlockRoot,
			want:    blkContainers[20].Block,
		},
		{
			name:    "canonical",
			blockID: []byte("30"),
			want:    blkContainers[30].Block,
		},
		{
			name:    "genesis",
			blockID: []byte("genesis"),
			want:    genBlk,
		},
		{
			name:    "genesis root",
			blockID: root[:],
			want:    genBlk,
		},
		{
			name:    "head",
			blockID: []byte("head"),
			want:    headBlock.Block,
		},
		{
			name:    "finalized",
			blockID: []byte("finalized"),
			want:    blkContainers[64].Block,
		},
		{
			name:    "no block",
			blockID: []byte("105"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := bs.GetBlockHeader(ctx, &ethpb.BlockRequest{
				BlockId: tt.blockID,
			})
			if !tt.wantErr {
				require.NoError(t, err)
			} else {
				require.NotEqual(t, err, nil)
				return
			}

			blkHdr, err := migration.V1Alpha1BlockToV1BlockHeader(tt.want)
			require.NoError(t, err)

			if !reflect.DeepEqual(header.Data.Header.Message, blkHdr.Message) {
				t.Error("Expected blocks to equal")
			}
		})
	}
}

func TestServer_ListBlockHeaders(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]
	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
	}

	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))
	b4 := testutil.NewBeaconBlock()
	b4.Block.Slot = 31
	b4.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b4)))
	b5 := testutil.NewBeaconBlock()
	b5.Block.Slot = 28
	b5.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b5)))

	tests := []struct {
		name       string
		slot       types.Slot
		parentRoot []byte
		want       []*ethpb_alpha.SignedBeaconBlock
		wantErr    bool
	}{
		{
			name: "slot",
			slot: types.Slot(30),
			want: []*ethpb_alpha.SignedBeaconBlock{
				blkContainers[30].Block,
				b2,
				b3,
			},
		},
		{
			name:       "parent root",
			parentRoot: b2.Block.ParentRoot,
			want: []*ethpb_alpha.SignedBeaconBlock{
				blkContainers[1].Block,
				b2,
				b4,
				b5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers, err := bs.ListBlockHeaders(ctx, &ethpb.BlockHeadersRequest{
				Slot:       &tt.slot,
				ParentRoot: tt.parentRoot,
			})
			require.NoError(t, err)

			require.Equal(t, len(tt.want), len(headers.Data))
			for i, blk := range tt.want {
				signedHdr, err := migration.V1Alpha1BlockToV1BlockHeader(blk)
				require.NoError(t, err)

				if !reflect.DeepEqual(headers.Data[i].Header.Message, signedHdr.Message) {
					t.Error("Expected blocks to equal")
				}
			}
		})
	}
}

func TestServer_ProposeBlock_OK(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	genesis := testutil.NewBeaconBlock()
	require.NoError(t, beaconDB.SaveBlock(context.Background(), wrapper.WrappedPhase0SignedBeaconBlock(genesis)), "Could not save genesis block")

	numDeposits := uint64(64)
	beaconState, _ := testutil.DeterministicGenesisState(t, numDeposits)
	bsRoot, err := beaconState.HashTreeRoot(ctx)
	require.NoError(t, err)
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(ctx, beaconState, genesisRoot), "Could not save genesis state")

	c := &mock.ChainService{Root: bsRoot[:], State: beaconState}
	beaconChainServer := &Server{
		BeaconDB:         beaconDB,
		BlockReceiver:    c,
		ChainInfoFetcher: c,
		BlockNotifier:    c.BlockNotifier(),
		Broadcaster:      mockp2p.NewTestP2P(t),
	}
	req := testutil.NewBeaconBlock()
	req.Block.Slot = 5
	req.Block.ParentRoot = bsRoot[:]
	v1Block, err := migration.V1Alpha1ToV1SignedBlock(req)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(req)))
	blockReq := &ethpb.BeaconBlockContainer{
		Message:   v1Block.Block,
		Signature: v1Block.Signature,
	}
	_, err = beaconChainServer.SubmitBlock(context.Background(), blockReq)
	assert.NoError(t, err, "Could not propose block correctly")
}

func TestServer_GetBlock(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]

	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))

	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
	}

	genBlk, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		name    string
		blockID []byte
		want    *ethpb_alpha.SignedBeaconBlock
		wantErr bool
	}{
		{
			name:    "slot",
			blockID: []byte("30"),
			want:    blkContainers[30].Block,
		},
		{
			name:    "bad formatting",
			blockID: []byte("3bad0"),
			wantErr: true,
		},
		{
			name:    "canonical",
			blockID: []byte("30"),
			want:    blkContainers[30].Block,
		},
		{
			name:    "head",
			blockID: []byte("head"),
			want:    headBlock.Block,
		},
		{
			name:    "finalized",
			blockID: []byte("finalized"),
			want:    blkContainers[64].Block,
		},
		{
			name:    "genesis",
			blockID: []byte("genesis"),
			want:    genBlk,
		},
		{
			name:    "genesis root",
			blockID: root[:],
			want:    genBlk,
		},
		{
			name:    "root",
			blockID: blkContainers[20].BlockRoot,
			want:    blkContainers[20].Block,
		},
		{
			name:    "non-existent root",
			blockID: bytesutil.PadTo([]byte("hi there"), 32),
			wantErr: true,
		},
		{
			name:    "slot",
			blockID: []byte("40"),
			want:    blkContainers[40].Block,
		},
		{
			name:    "no block",
			blockID: []byte("105"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := bs.GetBlock(ctx, &ethpb.BlockRequest{
				BlockId: tt.blockID,
			})
			if tt.wantErr {
				require.NotEqual(t, err, nil)
				return
			}
			require.NoError(t, err)

			v1Block, err := migration.V1Alpha1ToV1SignedBlock(tt.want)
			require.NoError(t, err)

			if !reflect.DeepEqual(block.Data.Message, v1Block.Block) {
				t.Error("Expected blocks to equal")
			}
		})
	}
}

func TestServer_GetBlockSSZ(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]

	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))

	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
	}

	ok, blocks, err := beaconDB.BlocksBySlot(ctx, 30)
	require.Equal(t, true, ok)
	require.NoError(t, err)
	sszBlock, err := blocks[0].MarshalSSZ()
	require.NoError(t, err)

	resp, err := bs.GetBlockSSZ(ctx, &ethpb.BlockRequest{BlockId: []byte("30")})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.DeepEqual(t, sszBlock, resp.Data)
}

func TestServer_GetBlockRoot(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	genBlk, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]
	b2 := testutil.NewBeaconBlock()
	b2.Block.Slot = 30
	b2.Block.ParentRoot = bytesutil.PadTo([]byte{1}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b2)))
	b3 := testutil.NewBeaconBlock()
	b3.Block.Slot = 30
	b3.Block.ParentRoot = bytesutil.PadTo([]byte{4}, 32)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(b3)))

	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
	}

	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		name    string
		blockID []byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "bad formatting",
			blockID: []byte("3bad0"),
			wantErr: true,
		},
		{
			name:    "canonical slot",
			blockID: []byte("30"),
			want:    blkContainers[30].BlockRoot,
		},
		{
			name:    "head",
			blockID: []byte("head"),
			want:    headBlock.BlockRoot,
		},
		{
			name:    "finalized",
			blockID: []byte("finalized"),
			want:    blkContainers[64].BlockRoot,
		},
		{
			name:    "genesis",
			blockID: []byte("genesis"),
			want:    root[:],
		},
		{
			name:    "genesis root",
			blockID: root[:],
			want:    root[:],
		},
		{
			name:    "root",
			blockID: blkContainers[20].BlockRoot,
			want:    blkContainers[20].BlockRoot,
		},
		{
			name:    "non-existent root",
			blockID: bytesutil.PadTo([]byte("hi there"), 32),
			wantErr: true,
		},
		{
			name:    "slot",
			blockID: []byte("40"),
			want:    blkContainers[40].BlockRoot,
		},
		{
			name:    "no block",
			blockID: []byte("105"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockRootResp, err := bs.GetBlockRoot(ctx, &ethpb.BlockRequest{
				BlockId: tt.blockID,
			})
			if tt.wantErr {
				require.NotEqual(t, err, nil)
				return
			}
			require.NoError(t, err)
			assert.DeepEqual(t, tt.want, blockRootResp.Data.Root)
		})
	}
}

func TestServer_ListBlockAttestations(t *testing.T) {
	beaconDB := dbTest.SetupDB(t)
	ctx := context.Background()

	_, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	headBlock := blkContainers[len(blkContainers)-1]
	bs := &Server{
		BeaconDB: beaconDB,
		ChainInfoFetcher: &mock.ChainService{
			DB:                  beaconDB,
			Block:               wrapper.WrappedPhase0SignedBeaconBlock(headBlock.Block),
			Root:                headBlock.BlockRoot,
			FinalizedCheckPoint: &ethpb_alpha.Checkpoint{Root: blkContainers[64].BlockRoot},
		},
	}

	genBlk, blkContainers := fillDBTestBlocks(ctx, t, beaconDB)
	root, err := genBlk.Block.HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		name    string
		blockID []byte
		want    *ethpb_alpha.SignedBeaconBlock
		wantErr bool
	}{
		{
			name:    "slot",
			blockID: []byte("30"),
			want:    blkContainers[30].Block,
		},
		{
			name:    "bad formatting",
			blockID: []byte("3bad0"),
			wantErr: true,
		},
		{
			name:    "head",
			blockID: []byte("head"),
			want:    headBlock.Block,
		},
		{
			name:    "finalized",
			blockID: []byte("finalized"),
			want:    blkContainers[64].Block,
		},
		{
			name:    "genesis",
			blockID: []byte("genesis"),
			want:    genBlk,
		},
		{
			name:    "genesis root",
			blockID: root[:],
			want:    genBlk,
		},
		{
			name:    "root",
			blockID: blkContainers[20].BlockRoot,
			want:    blkContainers[20].Block,
		},
		{
			name:    "non-existent root",
			blockID: bytesutil.PadTo([]byte("hi there"), 32),
			wantErr: true,
		},
		{
			name:    "slot",
			blockID: []byte("40"),
			want:    blkContainers[40].Block,
		},
		{
			name:    "no block",
			blockID: []byte("105"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := bs.ListBlockAttestations(ctx, &ethpb.BlockRequest{
				BlockId: tt.blockID,
			})
			if tt.wantErr {
				require.NotEqual(t, err, nil)
				return
			}
			require.NoError(t, err)

			v1Block, err := migration.V1Alpha1ToV1SignedBlock(tt.want)
			require.NoError(t, err)

			if !reflect.DeepEqual(block.Data, v1Block.Block.Body.Attestations) {
				t.Error("Expected attestations to equal")
			}
		})
	}
}
