package blockchain

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/execution"
	mockExecution "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/protoarray"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	bstate "github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_NotifyForkchoiceUpdate(t *testing.T) {
	params.BeaconConfig().SafeSlotsToImportOptimistically = 0
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	altairBlk := util.SaveBlock(t, ctx, beaconDB, util.NewBeaconBlockAltair())
	altairBlkRoot, err := altairBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	bellatrixBlk := util.SaveBlock(t, ctx, beaconDB, util.NewBeaconBlockBellatrix())
	bellatrixBlkRoot, err := bellatrixBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	st, _ := util.DeterministicGenesisState(t, 10)
	service.head = &head{
		state: st,
	}
	require.NoError(t, err)
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	state, blkRoot, err := prepareForkchoiceState(ctx, 0, [32]byte{}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 1, altairBlkRoot, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, bellatrixBlkRoot, altairBlkRoot, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))

	tests := []struct {
		name             string
		blk              interfaces.BeaconBlock
		headRoot         [32]byte
		finalizedRoot    [32]byte
		justifiedRoot    [32]byte
		newForkchoiceErr error
		errString        string
	}{
		{
			name: "phase0 block",
			blk: func() interfaces.BeaconBlock {
				b, err := consensusblocks.NewBeaconBlock(&ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}})
				require.NoError(t, err)
				return b
			}(),
		},
		{
			name: "altair block",
			blk: func() interfaces.BeaconBlock {
				b, err := consensusblocks.NewBeaconBlock(&ethpb.BeaconBlockAltair{Body: &ethpb.BeaconBlockBodyAltair{}})
				require.NoError(t, err)
				return b
			}(),
		},
		{
			name: "not execution block",
			blk: func() interfaces.BeaconBlock {
				b, err := consensusblocks.NewBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{
							ParentHash:    make([]byte, fieldparams.RootLength),
							FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
							StateRoot:     make([]byte, fieldparams.RootLength),
							ReceiptsRoot:  make([]byte, fieldparams.RootLength),
							LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
							PrevRandao:    make([]byte, fieldparams.RootLength),
							BaseFeePerGas: make([]byte, fieldparams.RootLength),
							BlockHash:     make([]byte, fieldparams.RootLength),
						},
					},
				})
				require.NoError(t, err)
				return b
			}(),
		},
		{
			name: "happy case: finalized root is altair block",
			blk: func() interfaces.BeaconBlock {
				b, err := consensusblocks.NewBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{},
					},
				})
				require.NoError(t, err)
				return b
			}(),
			finalizedRoot: altairBlkRoot,
			justifiedRoot: altairBlkRoot,
		},
		{
			name: "happy case: finalized root is bellatrix block",
			blk: func() interfaces.BeaconBlock {
				b, err := consensusblocks.NewBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{},
					},
				})
				require.NoError(t, err)
				return b
			}(),
			finalizedRoot: bellatrixBlkRoot,
			justifiedRoot: bellatrixBlkRoot,
		},
		{
			name: "forkchoice updated with optimistic block",
			blk: func() interfaces.BeaconBlock {
				b, err := consensusblocks.NewBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{},
					},
				})
				require.NoError(t, err)
				return b
			}(),
			newForkchoiceErr: execution.ErrAcceptedSyncingPayloadStatus,
			finalizedRoot:    bellatrixBlkRoot,
			justifiedRoot:    bellatrixBlkRoot,
		},
		{
			name: "forkchoice updated with invalid block",
			blk: func() interfaces.BeaconBlock {
				b, err := consensusblocks.NewBeaconBlock(&ethpb.BeaconBlockBellatrix{
					Body: &ethpb.BeaconBlockBodyBellatrix{
						ExecutionPayload: &v1.ExecutionPayload{},
					},
				})
				require.NoError(t, err)
				return b
			}(),
			newForkchoiceErr: execution.ErrInvalidPayloadStatus,
			finalizedRoot:    bellatrixBlkRoot,
			justifiedRoot:    bellatrixBlkRoot,
			headRoot:         [32]byte{'a'},
			errString:        ErrInvalidPayload.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service.cfg.ExecutionEngineCaller = &mockExecution.EngineClient{ErrForkchoiceUpdated: tt.newForkchoiceErr}
			st, _ := util.DeterministicGenesisState(t, 1)
			require.NoError(t, beaconDB.SaveState(ctx, st, tt.finalizedRoot))
			require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, tt.finalizedRoot))
			arg := &notifyForkchoiceUpdateArg{
				headState: st,
				headRoot:  tt.headRoot,
				headBlock: tt.blk,
			}
			_, err = service.notifyForkchoiceUpdate(ctx, arg)
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
				if tt.errString == ErrInvalidPayload.Error() {
					require.Equal(t, true, IsInvalidBlock(err))
					require.Equal(t, tt.headRoot, InvalidBlockRoot(err)) // Head root should be invalid. Not block root!
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//
//
//  A <- B <- C <- D
//       \
//         ---------- E <- F
//                     \
//                       ------ G
// D is the current head, attestations for F and G come late, both are invalid.
// We switch recursively to F then G and finally to D.
//
// We test:
// 1. forkchoice removes blocks F and G from the forkchoice implementation
// 2. forkchoice removes the weights of these blocks
// 3. the blockchain package calls fcu to obtain heads G -> F -> D.

func Test_NotifyForkchoiceUpdateRecursive_Protoarray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	// Prepare blocks
	ba := util.NewBeaconBlockBellatrix()
	ba.Block.Body.ExecutionPayload.BlockNumber = 1
	wba := util.SaveBlock(t, ctx, beaconDB, ba)
	bra, err := wba.Block().HashTreeRoot()
	require.NoError(t, err)

	bb := util.NewBeaconBlockBellatrix()
	bb.Block.Body.ExecutionPayload.BlockNumber = 2
	wbb := util.SaveBlock(t, ctx, beaconDB, bb)
	brb, err := wbb.Block().HashTreeRoot()
	require.NoError(t, err)

	bc := util.NewBeaconBlockBellatrix()
	bc.Block.Body.ExecutionPayload.BlockNumber = 3
	wbc := util.SaveBlock(t, ctx, beaconDB, bc)
	brc, err := wbc.Block().HashTreeRoot()
	require.NoError(t, err)

	bd := util.NewBeaconBlockBellatrix()
	pd := [32]byte{'D'}
	bd.Block.Body.ExecutionPayload.BlockHash = pd[:]
	bd.Block.Body.ExecutionPayload.BlockNumber = 4
	wbd := util.SaveBlock(t, ctx, beaconDB, bd)
	brd, err := wbd.Block().HashTreeRoot()
	require.NoError(t, err)

	be := util.NewBeaconBlockBellatrix()
	pe := [32]byte{'E'}
	be.Block.Body.ExecutionPayload.BlockHash = pe[:]
	be.Block.Body.ExecutionPayload.BlockNumber = 5
	wbe := util.SaveBlock(t, ctx, beaconDB, be)
	bre, err := wbe.Block().HashTreeRoot()
	require.NoError(t, err)

	bf := util.NewBeaconBlockBellatrix()
	pf := [32]byte{'F'}
	bf.Block.Body.ExecutionPayload.BlockHash = pf[:]
	bf.Block.Body.ExecutionPayload.BlockNumber = 6
	bf.Block.ParentRoot = bre[:]
	wbf := util.SaveBlock(t, ctx, beaconDB, bf)
	brf, err := wbf.Block().HashTreeRoot()
	require.NoError(t, err)

	bg := util.NewBeaconBlockBellatrix()
	bg.Block.Body.ExecutionPayload.BlockNumber = 7
	pg := [32]byte{'G'}
	bg.Block.Body.ExecutionPayload.BlockHash = pg[:]
	bg.Block.ParentRoot = bre[:]
	wbg := util.SaveBlock(t, ctx, beaconDB, bg)
	brg, err := wbg.Block().HashTreeRoot()
	require.NoError(t, err)

	// Insert blocks into forkchoice
	service := setupBeaconChain(t, beaconDB)
	fcs := protoarray.New()
	service.cfg.ForkChoiceStore = fcs
	service.cfg.ProposerSlotIndexCache = cache.NewProposerPayloadIDsCache()

	service.justifiedBalances.balances = []uint64{50, 100, 200}
	require.NoError(t, err)
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, bra, [32]byte{}, [32]byte{'A'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, brb, bra, [32]byte{'B'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 3, brc, brb, [32]byte{'C'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 4, brd, brc, [32]byte{'D'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 5, bre, brb, [32]byte{'E'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 6, brf, bre, [32]byte{'F'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 7, brg, bre, [32]byte{'G'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))

	// Insert Attestations to D, F and G so that they have higher weight than D
	// Ensure G is head
	fcs.ProcessAttestation(ctx, []uint64{0}, brd, 1)
	fcs.ProcessAttestation(ctx, []uint64{1}, brf, 1)
	fcs.ProcessAttestation(ctx, []uint64{2}, brg, 1)
	jc := &forkchoicetypes.Checkpoint{Epoch: 0, Root: bra}
	require.NoError(t, fcs.UpdateJustifiedCheckpoint(jc))
	headRoot, err := fcs.Head(ctx, []uint64{50, 100, 200})
	require.NoError(t, err)
	require.Equal(t, brg, headRoot)

	// Prepare Engine Mock to return invalid unless head is D, LVH =  E
	service.cfg.ExecutionEngineCaller = &mockExecution.EngineClient{ErrForkchoiceUpdated: execution.ErrInvalidPayloadStatus, ForkChoiceUpdatedResp: pe[:], OverrideValidHash: [32]byte{'D'}}
	st, _ := util.DeterministicGenesisState(t, 1)
	service.head = &head{
		state: st,
		block: wba,
	}

	require.NoError(t, beaconDB.SaveState(ctx, st, bra))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, bra))
	a := &notifyForkchoiceUpdateArg{
		headState: st,
		headBlock: wbg.Block(),
		headRoot:  brg,
	}
	_, err = service.notifyForkchoiceUpdate(ctx, a)
	require.Equal(t, true, IsInvalidBlock(err))
	require.Equal(t, brf, InvalidBlockRoot(err))

	// Ensure Head is D
	headRoot, err = fcs.Head(ctx, service.justifiedBalances.balances)
	require.NoError(t, err)
	require.Equal(t, brd, headRoot)

	// Ensure F and G where removed but their parent E wasn't
	require.Equal(t, false, fcs.HasNode(brf))
	require.Equal(t, false, fcs.HasNode(brg))
	require.Equal(t, true, fcs.HasNode(bre))
}

func Test_NotifyForkchoiceUpdate_NIlLVH(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	// Prepare blocks
	ba := util.NewBeaconBlockBellatrix()
	ba.Block.Body.ExecutionPayload.BlockNumber = 1
	wba := util.SaveBlock(t, ctx, beaconDB, ba)
	bra, err := wba.Block().HashTreeRoot()
	require.NoError(t, err)

	bb := util.NewBeaconBlockBellatrix()
	bb.Block.Body.ExecutionPayload.BlockNumber = 2
	wbb := util.SaveBlock(t, ctx, beaconDB, bb)
	brb, err := wbb.Block().HashTreeRoot()
	require.NoError(t, err)

	bc := util.NewBeaconBlockBellatrix()
	pc := [32]byte{'C'}
	bc.Block.Body.ExecutionPayload.BlockHash = pc[:]
	bc.Block.Body.ExecutionPayload.BlockNumber = 3
	wbc := util.SaveBlock(t, ctx, beaconDB, bc)
	brc, err := wbc.Block().HashTreeRoot()
	require.NoError(t, err)

	bd := util.NewBeaconBlockBellatrix()
	pd := [32]byte{'D'}
	bd.Block.Body.ExecutionPayload.BlockHash = pd[:]
	bd.Block.Body.ExecutionPayload.BlockNumber = 4
	bd.Block.ParentRoot = brc[:]
	wbd := util.SaveBlock(t, ctx, beaconDB, bd)
	brd, err := wbd.Block().HashTreeRoot()
	require.NoError(t, err)

	// Insert blocks into forkchoice
	service := setupBeaconChain(t, beaconDB)
	fcs := doublylinkedtree.New()
	service.cfg.ForkChoiceStore = fcs
	service.cfg.ProposerSlotIndexCache = cache.NewProposerPayloadIDsCache()

	service.justifiedBalances.balances = []uint64{50, 100, 200}
	require.NoError(t, err)
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, bra, [32]byte{}, [32]byte{'A'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, brb, bra, [32]byte{'B'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 3, brc, brb, [32]byte{'C'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 4, brd, brc, [32]byte{'D'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))

	// Prepare Engine Mock to return invalid LVH =  nil
	service.cfg.ExecutionEngineCaller = &mockExecution.EngineClient{ErrForkchoiceUpdated: execution.ErrInvalidPayloadStatus, OverrideValidHash: [32]byte{'C'}}
	st, _ := util.DeterministicGenesisState(t, 1)
	service.head = &head{
		state: st,
		block: wba,
	}

	require.NoError(t, beaconDB.SaveState(ctx, st, bra))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, bra))
	a := &notifyForkchoiceUpdateArg{
		headState: st,
		headBlock: wbd.Block(),
		headRoot:  brd,
	}
	_, err = service.notifyForkchoiceUpdate(ctx, a)
	require.Equal(t, true, IsInvalidBlock(err))
	require.Equal(t, brd, InvalidBlockRoot(err))
	require.Equal(t, brd, InvalidAncestorRoots(err)[0])
	require.Equal(t, 1, len(InvalidAncestorRoots(err)))
}

//
//
//  A <- B <- C <- D
//       \
//         ---------- E <- F
//                     \
//                       ------ G
// D is the current head, attestations for F and G come late, both are invalid.
// We switch recursively to F then G and finally to D.
//
// We test:
// 1. forkchoice removes blocks F and G from the forkchoice implementation
// 2. forkchoice removes the weights of these blocks
// 3. the blockchain package calls fcu to obtain heads G -> F -> D.

func Test_NotifyForkchoiceUpdateRecursive_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	// Prepare blocks
	ba := util.NewBeaconBlockBellatrix()
	ba.Block.Body.ExecutionPayload.BlockNumber = 1
	wba := util.SaveBlock(t, ctx, beaconDB, ba)
	bra, err := wba.Block().HashTreeRoot()
	require.NoError(t, err)

	bb := util.NewBeaconBlockBellatrix()
	bb.Block.Body.ExecutionPayload.BlockNumber = 2
	wbb := util.SaveBlock(t, ctx, beaconDB, bb)
	brb, err := wbb.Block().HashTreeRoot()
	require.NoError(t, err)

	bc := util.NewBeaconBlockBellatrix()
	bc.Block.Body.ExecutionPayload.BlockNumber = 3
	wbc := util.SaveBlock(t, ctx, beaconDB, bc)
	brc, err := wbc.Block().HashTreeRoot()
	require.NoError(t, err)

	bd := util.NewBeaconBlockBellatrix()
	pd := [32]byte{'D'}
	bd.Block.Body.ExecutionPayload.BlockHash = pd[:]
	bd.Block.Body.ExecutionPayload.BlockNumber = 4
	wbd := util.SaveBlock(t, ctx, beaconDB, bd)
	brd, err := wbd.Block().HashTreeRoot()
	require.NoError(t, err)

	be := util.NewBeaconBlockBellatrix()
	pe := [32]byte{'E'}
	be.Block.Body.ExecutionPayload.BlockHash = pe[:]
	be.Block.Body.ExecutionPayload.BlockNumber = 5
	wbe := util.SaveBlock(t, ctx, beaconDB, be)
	bre, err := wbe.Block().HashTreeRoot()
	require.NoError(t, err)

	bf := util.NewBeaconBlockBellatrix()
	pf := [32]byte{'F'}
	bf.Block.Body.ExecutionPayload.BlockHash = pf[:]
	bf.Block.Body.ExecutionPayload.BlockNumber = 6
	bf.Block.ParentRoot = bre[:]
	wbf := util.SaveBlock(t, ctx, beaconDB, bf)
	brf, err := wbf.Block().HashTreeRoot()
	require.NoError(t, err)

	bg := util.NewBeaconBlockBellatrix()
	bg.Block.Body.ExecutionPayload.BlockNumber = 7
	pg := [32]byte{'G'}
	bg.Block.Body.ExecutionPayload.BlockHash = pg[:]
	bg.Block.ParentRoot = bre[:]
	wbg := util.SaveBlock(t, ctx, beaconDB, bg)
	brg, err := wbg.Block().HashTreeRoot()
	require.NoError(t, err)

	// Insert blocks into forkchoice
	service := setupBeaconChain(t, beaconDB)
	fcs := doublylinkedtree.New()
	service.cfg.ForkChoiceStore = fcs
	service.cfg.ProposerSlotIndexCache = cache.NewProposerPayloadIDsCache()

	service.justifiedBalances.balances = []uint64{50, 100, 200}
	require.NoError(t, err)
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	state, blkRoot, err := prepareForkchoiceState(ctx, 1, bra, [32]byte{}, [32]byte{'A'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, brb, bra, [32]byte{'B'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 3, brc, brb, [32]byte{'C'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 4, brd, brc, [32]byte{'D'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 5, bre, brb, [32]byte{'E'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 6, brf, bre, [32]byte{'F'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 7, brg, bre, [32]byte{'G'}, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))

	// Insert Attestations to D, F and G so that they have higher weight than D
	// Ensure G is head
	fcs.ProcessAttestation(ctx, []uint64{0}, brd, 1)
	fcs.ProcessAttestation(ctx, []uint64{1}, brf, 1)
	fcs.ProcessAttestation(ctx, []uint64{2}, brg, 1)
	jc := &forkchoicetypes.Checkpoint{Epoch: 0, Root: bra}
	require.NoError(t, fcs.UpdateJustifiedCheckpoint(jc))
	headRoot, err := fcs.Head(ctx, []uint64{50, 100, 200})
	require.NoError(t, err)
	require.Equal(t, brg, headRoot)

	// Prepare Engine Mock to return invalid unless head is D, LVH =  E
	service.cfg.ExecutionEngineCaller = &mockExecution.EngineClient{ErrForkchoiceUpdated: execution.ErrInvalidPayloadStatus, ForkChoiceUpdatedResp: pe[:], OverrideValidHash: [32]byte{'D'}}
	st, _ := util.DeterministicGenesisState(t, 1)
	service.head = &head{
		state: st,
		block: wba,
	}

	require.NoError(t, beaconDB.SaveState(ctx, st, bra))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, bra))
	a := &notifyForkchoiceUpdateArg{
		headState: st,
		headBlock: wbg.Block(),
		headRoot:  brg,
	}
	_, err = service.notifyForkchoiceUpdate(ctx, a)
	require.Equal(t, true, IsInvalidBlock(err))
	require.Equal(t, brf, InvalidBlockRoot(err))

	// Ensure Head is D
	headRoot, err = fcs.Head(ctx, service.justifiedBalances.balances)
	require.NoError(t, err)
	require.Equal(t, brd, headRoot)

	// Ensure F and G where removed but their parent E wasn't
	require.Equal(t, false, fcs.HasNode(brf))
	require.Equal(t, false, fcs.HasNode(brg))
	require.Equal(t, true, fcs.HasNode(bre))
}

func Test_NotifyNewPayload(t *testing.T) {
	cfg := params.BeaconConfig()
	cfg.TerminalTotalDifficulty = "2"
	params.OverrideBeaconConfig(cfg)

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	phase0State, _ := util.DeterministicGenesisState(t, 1)
	altairState, _ := util.DeterministicGenesisStateAltair(t, 1)
	bellatrixState, _ := util.DeterministicGenesisStateBellatrix(t, 2)
	a := &ethpb.SignedBeaconBlockAltair{
		Block: &ethpb.BeaconBlockAltair{
			Body: &ethpb.BeaconBlockBodyAltair{},
		},
	}
	altairBlk, err := consensusblocks.NewSignedBeaconBlock(a)
	require.NoError(t, err)
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot: 1,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				ExecutionPayload: &v1.ExecutionPayload{
					BlockNumber:   1,
					ParentHash:    make([]byte, fieldparams.RootLength),
					FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
					StateRoot:     make([]byte, fieldparams.RootLength),
					ReceiptsRoot:  make([]byte, fieldparams.RootLength),
					LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
					PrevRandao:    make([]byte, fieldparams.RootLength),
					BaseFeePerGas: make([]byte, fieldparams.RootLength),
					BlockHash:     make([]byte, fieldparams.RootLength),
				},
			},
		},
	}
	bellatrixBlk, err := consensusblocks.NewSignedBeaconBlock(util.HydrateSignedBeaconBlockBellatrix(blk))
	require.NoError(t, err)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	st := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(epochsSinceFinalitySaveHotStateDB))
	service.genesisTime = time.Now().Add(time.Duration(-1*int64(st)*int64(params.BeaconConfig().SecondsPerSlot)) * time.Second)
	r, err := bellatrixBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	state, blkRoot, err := prepareForkchoiceState(ctx, 0, [32]byte{}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 1, r, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))

	tests := []struct {
		postState      bstate.BeaconState
		invalidBlock   bool
		isValidPayload bool
		blk            interfaces.SignedBeaconBlock
		newPayloadErr  error
		errString      string
		name           string
	}{
		{
			name:           "phase 0 post state",
			postState:      phase0State,
			isValidPayload: true,
		},
		{
			name:           "altair post state",
			postState:      altairState,
			isValidPayload: true,
		},
		{
			name:           "nil beacon block",
			postState:      bellatrixState,
			errString:      "signed beacon block can't be nil",
			isValidPayload: false,
		},
		{
			name:           "new payload with optimistic block",
			postState:      bellatrixState,
			blk:            bellatrixBlk,
			newPayloadErr:  execution.ErrAcceptedSyncingPayloadStatus,
			isValidPayload: false,
		},
		{
			name:           "new payload with invalid block",
			postState:      bellatrixState,
			blk:            bellatrixBlk,
			newPayloadErr:  execution.ErrInvalidPayloadStatus,
			errString:      ErrInvalidPayload.Error(),
			isValidPayload: false,
			invalidBlock:   true,
		},
		{
			name:           "altair pre state, altair block",
			postState:      bellatrixState,
			blk:            altairBlk,
			isValidPayload: true,
		},
		{
			name:      "altair pre state, happy case",
			postState: bellatrixState,
			blk: func() interfaces.SignedBeaconBlock {
				blk := &ethpb.SignedBeaconBlockBellatrix{
					Block: &ethpb.BeaconBlockBellatrix{
						Body: &ethpb.BeaconBlockBodyBellatrix{
							ExecutionPayload: &v1.ExecutionPayload{
								ParentHash: bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength),
							},
						},
					},
				}
				b, err := consensusblocks.NewSignedBeaconBlock(blk)
				require.NoError(t, err)
				return b
			}(),
			isValidPayload: true,
		},
		{
			name:      "not at merge transition",
			postState: bellatrixState,
			blk: func() interfaces.SignedBeaconBlock {
				blk := &ethpb.SignedBeaconBlockBellatrix{
					Block: &ethpb.BeaconBlockBellatrix{
						Body: &ethpb.BeaconBlockBodyBellatrix{
							ExecutionPayload: &v1.ExecutionPayload{
								ParentHash:    make([]byte, fieldparams.RootLength),
								FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
								StateRoot:     make([]byte, fieldparams.RootLength),
								ReceiptsRoot:  make([]byte, fieldparams.RootLength),
								LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
								PrevRandao:    make([]byte, fieldparams.RootLength),
								BaseFeePerGas: make([]byte, fieldparams.RootLength),
								BlockHash:     make([]byte, fieldparams.RootLength),
							},
						},
					},
				}
				b, err := consensusblocks.NewSignedBeaconBlock(blk)
				require.NoError(t, err)
				return b
			}(),
			isValidPayload: true,
		},
		{
			name:      "happy case",
			postState: bellatrixState,
			blk: func() interfaces.SignedBeaconBlock {
				blk := &ethpb.SignedBeaconBlockBellatrix{
					Block: &ethpb.BeaconBlockBellatrix{
						Body: &ethpb.BeaconBlockBodyBellatrix{
							ExecutionPayload: &v1.ExecutionPayload{
								ParentHash: bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength),
							},
						},
					},
				}
				b, err := consensusblocks.NewSignedBeaconBlock(blk)
				require.NoError(t, err)
				return b
			}(),
			isValidPayload: true,
		},
		{
			name:      "undefined error from ee",
			postState: bellatrixState,
			blk: func() interfaces.SignedBeaconBlock {
				blk := &ethpb.SignedBeaconBlockBellatrix{
					Block: &ethpb.BeaconBlockBellatrix{
						Body: &ethpb.BeaconBlockBodyBellatrix{
							ExecutionPayload: &v1.ExecutionPayload{
								ParentHash: bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength),
							},
						},
					},
				}
				b, err := consensusblocks.NewSignedBeaconBlock(blk)
				require.NoError(t, err)
				return b
			}(),
			newPayloadErr: ErrUndefinedExecutionEngineError,
			errString:     ErrUndefinedExecutionEngineError.Error(),
		},
		{
			name:      "invalid block hash error from ee",
			postState: bellatrixState,
			blk: func() interfaces.SignedBeaconBlock {
				blk := &ethpb.SignedBeaconBlockBellatrix{
					Block: &ethpb.BeaconBlockBellatrix{
						Body: &ethpb.BeaconBlockBodyBellatrix{
							ExecutionPayload: &v1.ExecutionPayload{
								ParentHash: bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength),
							},
						},
					},
				}
				b, err := consensusblocks.NewSignedBeaconBlock(blk)
				require.NoError(t, err)
				return b
			}(),
			newPayloadErr: ErrInvalidBlockHashPayloadStatus,
			errString:     ErrInvalidBlockHashPayloadStatus.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &mockExecution.EngineClient{ErrNewPayload: tt.newPayloadErr, BlockByHashMap: map[[32]byte]*v1.ExecutionBlock{}}
			e.BlockByHashMap[[32]byte{'a'}] = &v1.ExecutionBlock{
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("b")),
				},
				TotalDifficulty: "0x2",
			}
			e.BlockByHashMap[[32]byte{'b'}] = &v1.ExecutionBlock{
				Header: gethtypes.Header{
					ParentHash: common.BytesToHash([]byte("3")),
				},
				TotalDifficulty: "0x1",
			}
			service.cfg.ExecutionEngineCaller = e
			root := [32]byte{'a'}
			state, blkRoot, err := prepareForkchoiceState(ctx, 0, root, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
			require.NoError(t, err)
			require.NoError(t, service.cfg.ForkChoiceStore.InsertNode(ctx, state, blkRoot))
			postVersion, postHeader, err := getStateVersionAndPayload(tt.postState)
			require.NoError(t, err)
			isValidPayload, err := service.notifyNewPayload(ctx, postVersion, postHeader, tt.blk)
			if tt.errString != "" {
				require.ErrorContains(t, tt.errString, err)
				if tt.invalidBlock {
					require.Equal(t, true, IsInvalidBlock(err))
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.isValidPayload, isValidPayload)
				require.Equal(t, false, IsInvalidBlock(err))
			}
		})
	}
}

func Test_NotifyNewPayload_SetOptimisticToValid(t *testing.T) {
	cfg := params.BeaconConfig()
	cfg.TerminalTotalDifficulty = "2"
	params.OverrideBeaconConfig(cfg)
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	bellatrixState, _ := util.DeterministicGenesisStateBellatrix(t, 2)
	blk := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Body: &ethpb.BeaconBlockBodyBellatrix{
				ExecutionPayload: &v1.ExecutionPayload{
					ParentHash: bytesutil.PadTo([]byte{'a'}, fieldparams.RootLength),
				},
			},
		},
	}
	bellatrixBlk, err := consensusblocks.NewSignedBeaconBlock(blk)
	require.NoError(t, err)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	e := &mockExecution.EngineClient{BlockByHashMap: map[[32]byte]*v1.ExecutionBlock{}}
	e.BlockByHashMap[[32]byte{'a'}] = &v1.ExecutionBlock{
		Header: gethtypes.Header{
			ParentHash: common.BytesToHash([]byte("b")),
		},
		TotalDifficulty: "0x2",
	}
	e.BlockByHashMap[[32]byte{'b'}] = &v1.ExecutionBlock{
		Header: gethtypes.Header{
			ParentHash: common.BytesToHash([]byte("3")),
		},
		TotalDifficulty: "0x1",
	}
	service.cfg.ExecutionEngineCaller = e
	postVersion, postHeader, err := getStateVersionAndPayload(bellatrixState)
	require.NoError(t, err)
	validated, err := service.notifyNewPayload(ctx, postVersion, postHeader, bellatrixBlk)
	require.NoError(t, err)
	require.Equal(t, true, validated)
}

func Test_IsOptimisticCandidateBlock(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	params.BeaconConfig().SafeSlotsToImportOptimistically = 128
	service.genesisTime = time.Now().Add(-time.Second * 12 * 2 * 128)

	parentBlk := util.NewBeaconBlockBellatrix()
	wrappedParentBlock, err := consensusblocks.NewSignedBeaconBlock(parentBlk)
	require.NoError(t, err)
	parentRoot, err := wrappedParentBlock.Block().HashTreeRoot()
	require.NoError(t, err)

	tests := []struct {
		name      string
		blk       interfaces.BeaconBlock
		justified interfaces.SignedBeaconBlock
		err       error
	}{
		{
			name: "deep block",
			blk: func(tt *testing.T) interfaces.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 1
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := consensusblocks.NewBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) interfaces.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := consensusblocks.NewSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			err: nil,
		},
		{
			name: "shallow block, Altair justified chkpt",
			blk: func(tt *testing.T) interfaces.BeaconBlock {
				blk := util.NewBeaconBlockAltair()
				blk.Block.Slot = 200
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := consensusblocks.NewBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) interfaces.SignedBeaconBlock {
				blk := util.NewBeaconBlockAltair()
				blk.Block.Slot = 32
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := consensusblocks.NewSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			err: errNotOptimisticCandidate,
		},
		{
			name: "shallow block, Bellatrix justified chkpt without execution",
			blk: func(tt *testing.T) interfaces.BeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 200
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := consensusblocks.NewBeaconBlock(blk.Block)
				require.NoError(tt, err)
				return wr
			}(t),
			justified: func(tt *testing.T) interfaces.SignedBeaconBlock {
				blk := util.NewBeaconBlockBellatrix()
				blk.Block.Slot = 32
				blk.Block.ParentRoot = parentRoot[:]
				wr, err := consensusblocks.NewSignedBeaconBlock(blk)
				require.NoError(tt, err)
				return wr
			}(t),
			err: errNotOptimisticCandidate,
		},
	}
	for _, tt := range tests {
		require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, tt.justified))
		require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wrappedParentBlock))

		err = service.optimisticCandidateBlock(ctx, tt.blk)
		if tt.err != nil {
			require.Equal(t, tt.err.Error(), err.Error())
		} else {
			require.NoError(t, err)
		}
	}
}

func Test_IsOptimisticShallowExecutionParent(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	params.BeaconConfig().SafeSlotsToImportOptimistically = 128
	service.genesisTime = time.Now().Add(-time.Second * 12 * 2 * 128)
	payload := &v1.ExecutionPayload{
		ParentHash:    make([]byte, 32),
		FeeRecipient:  make([]byte, 20),
		StateRoot:     make([]byte, 32),
		ReceiptsRoot:  make([]byte, 32),
		LogsBloom:     make([]byte, 256),
		PrevRandao:    make([]byte, 32),
		BaseFeePerGas: bytesutil.PadTo([]byte{1, 2, 3, 4}, fieldparams.RootLength),
		BlockHash:     make([]byte, 32),
		BlockNumber:   100,
	}
	body := &ethpb.BeaconBlockBodyBellatrix{ExecutionPayload: payload}
	b := &ethpb.BeaconBlockBellatrix{Body: body, Slot: 200}
	rawSigned := &ethpb.SignedBeaconBlockBellatrix{Block: b}
	blk := util.HydrateSignedBeaconBlockBellatrix(rawSigned)
	wr := util.SaveBlock(t, ctx, service.cfg.BeaconDB, blk)
	blkRoot, err := wr.Block().HashTreeRoot()
	require.NoError(t, err)

	childBlock := util.NewBeaconBlockBellatrix()
	childBlock.Block.ParentRoot = blkRoot[:]
	// shallow block
	childBlock.Block.Slot = 201
	wrappedChild := util.SaveBlock(t, ctx, service.cfg.BeaconDB, childBlock)
	err = service.optimisticCandidateBlock(ctx, wrappedChild.Block())
	require.NoError(t, err)
}

func Test_GetPayloadAttribute(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}

	// Cache miss
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	hasPayload, _, vId, err := service.getPayloadAttribute(ctx, nil, 0)
	require.NoError(t, err)
	require.Equal(t, false, hasPayload)
	require.Equal(t, types.ValidatorIndex(0), vId)

	// Cache hit, advance state, no fee recipient
	suggestedVid := types.ValidatorIndex(1)
	slot := types.Slot(1)
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(slot, suggestedVid, [8]byte{}, [32]byte{})
	st, _ := util.DeterministicGenesisState(t, 1)
	hook := logTest.NewGlobal()
	hasPayload, attr, vId, err := service.getPayloadAttribute(ctx, st, slot)
	require.NoError(t, err)
	require.Equal(t, true, hasPayload)
	require.Equal(t, suggestedVid, vId)
	require.Equal(t, params.BeaconConfig().EthBurnAddressHex, common.BytesToAddress(attr.SuggestedFeeRecipient).String())
	require.LogsContain(t, hook, "Fee recipient is currently using the burn address")

	// Cache hit, advance state, has fee recipient
	suggestedAddr := common.HexToAddress("123")
	require.NoError(t, service.cfg.BeaconDB.SaveFeeRecipientsByValidatorIDs(ctx, []types.ValidatorIndex{suggestedVid}, []common.Address{suggestedAddr}))
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(slot, suggestedVid, [8]byte{}, [32]byte{})
	hasPayload, attr, vId, err = service.getPayloadAttribute(ctx, st, slot)
	require.NoError(t, err)
	require.Equal(t, true, hasPayload)
	require.Equal(t, suggestedVid, vId)
	require.Equal(t, suggestedAddr, common.BytesToAddress(attr.SuggestedFeeRecipient))
}

func Test_UpdateLastValidatedCheckpoint(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())

	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	stateGen := stategen.New(beaconDB)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stateGen),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	genesisStateRoot := [32]byte{}
	genesisBlk := blocks.NewGenesisBlock(genesisStateRoot[:])
	util.SaveBlock(t, ctx, beaconDB, genesisBlk)
	genesisRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	assert.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, genesisRoot))
	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	fjc := &forkchoicetypes.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash}
	require.NoError(t, fcs.UpdateJustifiedCheckpoint(fjc))
	require.NoError(t, fcs.UpdateFinalizedCheckpoint(fjc))
	state, blkRoot, err := prepareForkchoiceState(ctx, 0, genesisRoot, params.BeaconConfig().ZeroHash, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	fcs.SetOriginRoot(genesisRoot)
	genesisSummary := &ethpb.StateSummary{
		Root: genesisStateRoot[:],
		Slot: 0,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, genesisSummary))

	// Get last validated checkpoint
	origCheckpoint, err := service.cfg.BeaconDB.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveLastValidatedCheckpoint(ctx, origCheckpoint))

	// Optimistic finalized checkpoint
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 320
	blk.Block.ParentRoot = genesisRoot[:]
	util.SaveBlock(t, ctx, beaconDB, blk)
	opRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	opCheckpoint := &ethpb.Checkpoint{
		Root:  opRoot[:],
		Epoch: 10,
	}
	opStateSummary := &ethpb.StateSummary{
		Root: opRoot[:],
		Slot: 320,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, opStateSummary))
	tenjc := &ethpb.Checkpoint{Epoch: 10, Root: genesisRoot[:]}
	tenfc := &ethpb.Checkpoint{Epoch: 10, Root: genesisRoot[:]}
	state, blkRoot, err = prepareForkchoiceState(ctx, 320, opRoot, genesisRoot, params.BeaconConfig().ZeroHash, tenjc, tenfc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	assert.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, opRoot))
	require.NoError(t, service.updateFinalized(ctx, opCheckpoint))
	cp, err := service.cfg.BeaconDB.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)
	require.DeepEqual(t, origCheckpoint.Root, cp.Root)
	require.Equal(t, origCheckpoint.Epoch, cp.Epoch)

	// Validated finalized checkpoint
	blk = util.NewBeaconBlock()
	blk.Block.Slot = 640
	blk.Block.ParentRoot = opRoot[:]
	util.SaveBlock(t, ctx, beaconDB, blk)
	validRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)

	validCheckpoint := &ethpb.Checkpoint{
		Root:  validRoot[:],
		Epoch: 20,
	}
	validSummary := &ethpb.StateSummary{
		Root: validRoot[:],
		Slot: 640,
	}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, validSummary))
	twentyjc := &ethpb.Checkpoint{Epoch: 20, Root: validRoot[:]}
	twentyfc := &ethpb.Checkpoint{Epoch: 20, Root: validRoot[:]}
	state, blkRoot, err = prepareForkchoiceState(ctx, 640, validRoot, genesisRoot, params.BeaconConfig().ZeroHash, twentyjc, twentyfc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	require.NoError(t, fcs.SetOptimisticToValid(ctx, validRoot))
	assert.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, validRoot))
	require.NoError(t, service.updateFinalized(ctx, validCheckpoint))
	cp, err = service.cfg.BeaconDB.LastValidatedCheckpoint(ctx)
	require.NoError(t, err)

	optimistic, err := service.IsOptimisticForRoot(ctx, validRoot)
	require.NoError(t, err)
	require.Equal(t, false, optimistic)
	require.DeepEqual(t, validCheckpoint.Root, cp.Root)
	require.Equal(t, validCheckpoint.Epoch, cp.Epoch)
}

func TestService_removeInvalidBlockAndState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	// Deleting unknown block should not error.
	require.NoError(t, service.removeInvalidBlockAndState(ctx, [][32]byte{{'a'}, {'b'}, {'c'}}))

	// Happy case
	b1 := util.NewBeaconBlock()
	b1.Block.Slot = 1
	blk1 := util.SaveBlock(t, ctx, service.cfg.BeaconDB, b1)
	r1, err := blk1.Block().HashTreeRoot()
	require.NoError(t, err)
	st, _ := util.DeterministicGenesisStateBellatrix(t, 1)
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: 1,
		Root: r1[:],
	}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, r1))

	b2 := util.NewBeaconBlock()
	b2.Block.Slot = 2
	blk2 := util.SaveBlock(t, ctx, service.cfg.BeaconDB, b2)
	r2, err := blk2.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
		Slot: 2,
		Root: r2[:],
	}))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, st, r2))

	require.NoError(t, service.removeInvalidBlockAndState(ctx, [][32]byte{r1, r2}))

	require.Equal(t, false, service.hasBlock(ctx, r1))
	require.Equal(t, false, service.hasBlock(ctx, r2))
	require.Equal(t, false, service.cfg.BeaconDB.HasStateSummary(ctx, r1))
	require.Equal(t, false, service.cfg.BeaconDB.HasStateSummary(ctx, r2))
	has, err := service.cfg.StateGen.HasState(ctx, r1)
	require.NoError(t, err)
	require.Equal(t, false, has)
	has, err = service.cfg.StateGen.HasState(ctx, r2)
	require.NoError(t, err)
	require.Equal(t, false, has)
}

func TestService_getPayloadHash(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(doublylinkedtree.New()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	_, err = service.getPayloadHash(ctx, []byte{})
	require.ErrorIs(t, errBlockNotFoundInCacheOrDB, err)

	b := util.NewBeaconBlock()
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err := consensusblocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, service.saveInitSyncBlock(ctx, r, wsb))

	h, err := service.getPayloadHash(ctx, r[:])
	require.NoError(t, err)
	require.DeepEqual(t, params.BeaconConfig().ZeroHash, h)

	bb := util.NewBeaconBlockBellatrix()
	h = [32]byte{'a'}
	bb.Block.Body.ExecutionPayload.BlockHash = h[:]
	r, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	wsb, err = consensusblocks.NewSignedBeaconBlock(bb)
	require.NoError(t, err)
	require.NoError(t, service.saveInitSyncBlock(ctx, r, wsb))

	h, err = service.getPayloadHash(ctx, r[:])
	require.NoError(t, err)
	require.DeepEqual(t, [32]byte{'a'}, h)
}
