package blockchain

import (
	"os"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	mockExecution "github.com/prysmaticlabs/prysm/v4/beacon-chain/execution/testing"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	state_native "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	consensus_blocks "github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type Unmarshaler interface {
	UnmarshalSSZ(buf []byte) error
}

// dataFetcher fetches and unmarshals data from file to provided data structure.
func dataFetcher(fPath string, data Unmarshaler) error {
	rawFile, err := os.ReadFile(fPath) // #nosec G304
	if err != nil {
		return err
	}
	return data.UnmarshalSSZ(rawFile)
}

// Test setup:
// Start with a head at slot 6653310
// We import a timely block 6653311 this should warm up the caches
// We import a late block 6653312 (epoch boundary). This should advance the NCS.
// We import an early block 6653313 (slot 1). This should cause a reorg.
//
// This test does not run a full node, but rather runs the full blockchain
// package without execution engine deadlocks. If you suspect that a delay comes
// from the EL you should not use this template. This test also does not fully
// cover the DB paths, it tricks the DB in thinking that we are doing a
// checkpoint sync from the justified state so that it will fail to migrate old
// states to cold.
func TestLateSlot1(t *testing.T) {
	hook := logTest.NewGlobal()
	slot32 := primitives.Slot(6653312)
	baseDir := "/home/heluani/Documents/code/ethereum/prysm/beacon-chain/blockchain/testing/scenario/"
	// finalizedState is the current finalized state at slot 30.
	finalizedStateData := &ethpb.BeaconStateCapella{}
	sszPathFinalizedState := baseDir + "state_finalized.ssz"
	require.NoError(t, dataFetcher(sszPathFinalizedState, finalizedStateData))
	finalizedState, err := state_native.InitializeFromProtoCapella(finalizedStateData)
	require.NoError(t, err)
	require.Equal(t, slot32, finalizedState.Slot()+96)

	// finalizedBlock is the beacon block with the root that was finalized
	// checkpoint root.
	finalizedBlockData := &ethpb.SignedBeaconBlockCapella{}
	sszPathFinalizedBlock := baseDir + "block_finalized.ssz"
	require.NoError(t, dataFetcher(sszPathFinalizedBlock, finalizedBlockData))
	finalizedBlock, err := consensus_blocks.NewSignedBeaconBlock(finalizedBlockData)
	require.NoError(t, err)
	require.Equal(t, slot32, finalizedBlock.Block().Slot()+96)

	// justifiedState is the current justified state at slot 30, we will
	// consider this to be our genesis. It is at slot32 - 64.
	justifiedStateData := &ethpb.BeaconStateCapella{}
	sszPathJustifiedState := baseDir + "state_justified.ssz"
	require.NoError(t, dataFetcher(sszPathJustifiedState, justifiedStateData))
	justifiedState, err := state_native.InitializeFromProtoCapella(justifiedStateData)
	require.NoError(t, err)
	require.Equal(t, slot32, justifiedState.Slot()+64)

	// justifiedBlock is the beacon block with the root that was justified
	// checkpoint root.
	justifiedBlockData := &ethpb.SignedBeaconBlockCapella{}
	sszPathJustifiedBlock := baseDir + "block_justified.ssz"
	require.NoError(t, dataFetcher(sszPathJustifiedBlock, justifiedBlockData))
	justifiedBlock, err := consensus_blocks.NewSignedBeaconBlock(justifiedBlockData)
	require.NoError(t, err)
	require.Equal(t, slot32, justifiedBlock.Block().Slot()+64)

	// finalizedJustifiedState is the beacon state at the justification
	// checkpoint of the finalized checkpoint of the state at slot 30,
	// It is needed to obtain the right balances when starting
	finalizedJustifiedStateData := &ethpb.BeaconStateCapella{}
	sszPathFinalizedJustifiedState := baseDir + "state_finalized_justified.ssz"
	require.NoError(t, dataFetcher(sszPathFinalizedJustifiedState, finalizedJustifiedStateData))
	finalizedJustifiedState, err := state_native.InitializeFromProtoCapella(finalizedJustifiedStateData)
	require.NoError(t, err)
	require.Equal(t, slot32, finalizedJustifiedState.Slot()+128)

	// finalizedJustifiedBlock is the beacon block with the root that was
	// the finalizedJustified checkpoint root.
	finalizedJustifiedBlockData := &ethpb.SignedBeaconBlockCapella{}
	sszPathFinalizedJustifiedBlock := baseDir + "block_finalized_justified.ssz"
	require.NoError(t, dataFetcher(sszPathFinalizedJustifiedBlock, finalizedJustifiedBlockData))
	finalizedJustifiedBlock, err := consensus_blocks.NewSignedBeaconBlock(finalizedJustifiedBlockData)
	require.NoError(t, err)
	require.Equal(t, slot32, finalizedJustifiedBlock.Block().Slot()+128)

	// postState30 is the state at slot 30 right after applying the block at
	// slot 30.
	postState30Data := &ethpb.BeaconStateCapella{}
	sszPathPostState30 := baseDir + "post_state_30.ssz"
	require.NoError(t, dataFetcher(sszPathPostState30, postState30Data))
	postState30, err := state_native.InitializeFromProtoCapella(postState30Data)
	require.NoError(t, err)
	require.Equal(t, slot32, postState30.Slot()+2)

	// block30 is the beacon block that arrived at slot 30. It's the
	// starting headRoot for the test
	beaconBlock30Data := &ethpb.SignedBeaconBlockCapella{}
	sszPathBlock30 := baseDir + "block30.ssz"
	require.NoError(t, dataFetcher(sszPathBlock30, beaconBlock30Data))
	block30, err := consensus_blocks.NewSignedBeaconBlock(beaconBlock30Data)
	require.NoError(t, err)
	require.Equal(t, slot32, block30.Block().Slot()+2)

	// block31 is the beacon block that arrived at slot 31.
	beaconBlock31Data := &ethpb.SignedBeaconBlockCapella{}
	sszPathBlock31 := baseDir + "block31.ssz"
	require.NoError(t, dataFetcher(sszPathBlock31, beaconBlock31Data))
	block31, err := consensus_blocks.NewSignedBeaconBlock(beaconBlock31Data)
	require.NoError(t, err)
	require.Equal(t, slot32, block31.Block().Slot()+1)

	// block32 is the beacon block that arrived at slot 32.
	beaconBlock32Data := &ethpb.SignedBeaconBlockCapella{}
	sszPathBlock32 := baseDir + "block32.ssz"
	require.NoError(t, dataFetcher(sszPathBlock32, beaconBlock32Data))
	block32, err := consensus_blocks.NewSignedBeaconBlock(beaconBlock32Data)
	require.NoError(t, err)
	require.Equal(t, slot32, block32.Block().Slot())

	// block33 is the beacon block that arrived at slot 33, this block is
	// reorged
	beaconBlock33Data := &ethpb.SignedBeaconBlockCapella{}
	sszPathBlock33 := baseDir + "block33.ssz"
	require.NoError(t, dataFetcher(sszPathBlock33, beaconBlock33Data))
	block33, err := consensus_blocks.NewSignedBeaconBlock(beaconBlock33Data)
	require.NoError(t, err)
	require.Equal(t, slot32+1, block33.Block().Slot())
	require.Equal(t, block32.Block().ParentRoot(), block33.Block().ParentRoot())

	// newJustifiedBlock is the beacon block that arrived at slot 0, it's the
	// block that becomes justified at the epoch boundary.
	newJustifiedBlockData := &ethpb.SignedBeaconBlockCapella{}
	newJustifiedBlockPath := baseDir + "new_justified_block.ssz"
	require.NoError(t, dataFetcher(newJustifiedBlockPath, newJustifiedBlockData))
	newJustifiedBlock, err := consensus_blocks.NewSignedBeaconBlock(newJustifiedBlockData)
	require.NoError(t, err)
	require.Equal(t, slot32, newJustifiedBlock.Block().Slot()+32)

	// newJustifiedState is the beacon State at slot 0. It is the state that
	// gets justified at the epoch boundary.
	newJustifiedStateData := &ethpb.BeaconStateCapella{}
	newJustifiedStatePath := baseDir + "new_justified_state.ssz"
	require.NoError(t, dataFetcher(newJustifiedStatePath, newJustifiedStateData))
	newJustifiedState, err := state_native.InitializeFromProtoCapella(newJustifiedStateData)
	require.NoError(t, err)
	require.Equal(t, slot32, newJustifiedState.Slot()+32)

	// Setup the service
	service, tr := minimalTestService(t)
	ctx, beaconDB, fcs := tr.ctx, tr.db, tr.fcs

	// Save the justified and finalized states and blocks.
	require.NoError(t, beaconDB.SaveBlock(ctx, finalizedBlock))
	finalizedRoot, err := finalizedBlock.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, finalizedState.Copy(), finalizedRoot))
	require.NoError(t, beaconDB.SaveBlock(ctx, justifiedBlock))
	justifiedRoot, err := justifiedBlock.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, justifiedState.Copy(), justifiedRoot))
	fcp := postState30.FinalizedCheckpoint()
	require.Equal(t, [32]byte(fcp.Root), finalizedRoot)
	jcp := postState30.CurrentJustifiedCheckpoint()
	require.Equal(t, [32]byte(jcp.Root), justifiedRoot)
	finalizedSummary := &ethpb.StateSummary{Slot: finalizedState.Slot(), Root: finalizedRoot[:]}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, finalizedSummary))
	justifiedSummary := &ethpb.StateSummary{Slot: justifiedState.Slot(), Root: justifiedRoot[:]}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, justifiedSummary))

	// Forkchoice requires justified balances,
	// For this we need to save the justified checkpoint of the
	// corresponding finalized state.
	require.NoError(t, beaconDB.SaveBlock(ctx, finalizedJustifiedBlock))
	finalizedJustifiedRoot, err := finalizedJustifiedBlock.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, finalizedJustifiedState.Copy(), finalizedJustifiedRoot))
	fjcs := finalizedState.CurrentJustifiedCheckpoint()
	require.Equal(t, [32]byte(fjcs.Root), finalizedJustifiedRoot)
	finalizedJustifiedSummary := &ethpb.StateSummary{Slot: finalizedJustifiedState.Slot(), Root: finalizedJustifiedRoot[:]}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, finalizedJustifiedSummary))

	// Setup forkchoice to have the two checkpoints
	execution, err := finalizedBlock.Block().Body().Execution()
	require.NoError(t, err)
	payloadHash := [32]byte(execution.BlockHash())
	// We use a fake finalized checkpoint for the finalized checkpoint node
	// it's self-referencing as if it was genesis
	st, _, err := prepareForkchoiceState(ctx, slot32-96, finalizedRoot, [32]byte{}, payloadHash, fjcs, fcp)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, st, finalizedRoot))

	// We fake the justified checkpoint to have as parent the finalized
	// checkpoint
	execution, err = justifiedBlock.Block().Body().Execution()
	require.NoError(t, err)
	payloadHash = [32]byte(execution.BlockHash())
	st, _, err = prepareForkchoiceState(ctx, slot32-64, justifiedRoot, finalizedRoot, payloadHash, fcp, fcp)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, st, justifiedRoot))

	require.NoError(t, fcs.UpdateFinalizedCheckpoint(&forkchoicetypes.Checkpoint{Epoch: fcp.Epoch, Root: [32]byte(fcp.Root)}))
	require.NoError(t, fcs.UpdateJustifiedCheckpoint(ctx, &forkchoicetypes.Checkpoint{Epoch: jcp.Epoch, Root: [32]byte(jcp.Root)}))

	// Save the new justified state/block to DB since it will be justified
	// in the epoch pre-compute, we need to insert this block as well to
	// forkchoice
	require.NoError(t, beaconDB.SaveBlock(ctx, newJustifiedBlock))
	newJustifiedRoot, err := newJustifiedBlock.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, newJustifiedState.Copy(), newJustifiedRoot))
	newJustifiedSummary := &ethpb.StateSummary{Slot: newJustifiedState.Slot(), Root: newJustifiedRoot[:]}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, newJustifiedSummary))
	st, _, err = prepareForkchoiceState(ctx, slot32-32, newJustifiedRoot, justifiedRoot, [32]byte{}, jcp, fcp)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, st, newJustifiedRoot))

	// We fake the forkchoice node at 30 since we won't be able to import
	// the block 31 otherwise. We need to use a full state to be able to
	// import later block 31. When importing this block we need to set
	// genesis time so that it is timely
	secondsSinceGenesis, err := (slot32 - 2).SafeMul(params.BeaconConfig().SecondsPerSlot)
	require.NoError(t, err)
	genesisTime := time.Now().Add(-time.Duration(uint64(secondsSinceGenesis)) * time.Second)
	service.SetGenesisTime(genesisTime)
	fcs.SetGenesisTime(uint64(genesisTime.Unix()))

	// Start the service that will run the late block tasks and advance
	// slots
	var vr [32]byte
	require.NoError(t, service.clockSetter.SetClock(startup.NewClock(time.Now(), vr)))
	go service.runLateBlockTasks()
	go service.spawnProcessAttestationsRoutine()

	block30Root := block31.Block().ParentRoot()
	fakeState30 := postState30.Copy()
	bh := fakeState30.LatestBlockHeader()
	copy(bh.ParentRoot, newJustifiedRoot[:])
	require.NoError(t, fakeState30.SetLatestBlockHeader(bh))
	require.NoError(t, fcs.InsertNode(ctx, fakeState30, block30Root))
	fcHead, err := fcs.Head(ctx)
	require.NoError(t, err)
	require.Equal(t, block30Root, fcHead)

	// Check blockchain package's checkpoints
	bfcp := service.FinalizedCheckpt()
	require.Equal(t, fcp.Epoch, bfcp.Epoch)
	require.Equal(t, [32]byte(bfcp.Root), [32]byte(fcp.Root))
	bjcp := service.CurrentJustifiedCheckpt()
	require.DeepEqual(t, jcp, bjcp)

	// Set up the node's head
	require.NoError(t, beaconDB.SaveBlock(ctx, block30))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, postState30.Copy(), block30Root))
	block30Summary := &ethpb.StateSummary{Slot: slot32 - 2, Root: block30Root[:]}
	require.NoError(t, beaconDB.SaveStateSummary(ctx, block30Summary))
	require.NoError(t, service.setHead(block30Root, block30, postState30))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, block30Root))
	headRoot, err := service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, block30Root, [32]byte(headRoot))
	headState, err := service.HeadState(ctx)
	require.NoError(t, err)
	require.Equal(t, headState.Slot(), slot32-2)

	// Update the node's next slot cache
	require.NoError(t, transition.UpdateNextSlotCache(ctx, block30Root[:], postState30))
	preState31 := transition.NextSlotState(block30Root[:], slot32-1)
	require.NotNil(t, preState31)

	// Mock the execution engine to accept any block as valid
	executionEngine := &mockExecution.EngineClient{}
	service.cfg.ExecutionEngineCaller = executionEngine

	// Save the origin checkpoint root to prevent the db from walking up the
	// ancestry. We are asserting that the justified checkpoint root (that
	// is at slot32 - 64 is the init checkpoint
	require.NoError(t, service.cfg.BeaconDB.SaveOriginCheckpointBlockRoot(ctx, justifiedRoot))

	// Wait until slot 31 to import it. We wait so that the late blocks
	// tasks are run.
	slot31Start := slots.StartTime(uint64(service.genesisTime.Unix()), slot32-1)
	time.Sleep(slot31Start.Add(time.Second).Sub(time.Now()))

	// Import block 31 check that headRoot and NSC are right
	block31Root := block32.Block().ParentRoot()
	require.NoError(t, service.ReceiveBlock(ctx, block31, block31Root))
	headRoot, err = service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, block30Root, block31.Block().ParentRoot())
	require.Equal(t, block31Root, [32]byte(headRoot))
	headState, err = service.HeadState(ctx)
	require.NoError(t, err)
	require.Equal(t, headState.Slot(), slot32-1)

	// Why do we need to sleep here to let the NSC to be updated? state copies take long it seems
	time.Sleep(200 * time.Millisecond)
	preState32 := transition.NextSlotState(block31Root[:], slot32)
	require.NotNil(t, preState32)
	require.Equal(t, slot32, preState32.Slot())

	// Wait until the start of slot 32 + 5 seconds so that slot 32 is late
	// Import block 32
	slot32Start := slots.StartTime(uint64(service.genesisTime.Unix()), slot32)
	time.Sleep(slot32Start.Add(5 * time.Second).Sub(time.Now()))

	block32Root, err := block32.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.ReceiveBlock(ctx, block32, block32Root))
	headRoot, err = service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, block32Root, [32]byte(headRoot))
	headState, err = service.HeadState(ctx)
	require.NoError(t, err)
	require.Equal(t, headState.Slot(), slot32)

	preState33 := transition.NextSlotState(block32Root[:], slot32+1)
	require.NotNil(t, preState33)
	require.Equal(t, slot32+1, preState33.Slot())

	// We should also have advanced the NSC for the blockroot of 31
	preState33Alt := transition.NextSlotState(block31Root[:], slot32+1)
	require.NotNil(t, preState33Alt)
	require.Equal(t, slot32+1, preState33Alt.Slot())

	// Wait until the start of slot 33 and import it early
	slot33Start := slots.StartTime(uint64(service.genesisTime.Unix()), slot32+1)
	time.Sleep(slot33Start.Add(time.Second).Sub(time.Now()))

	block33Root, err := block33.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.ReceiveBlock(ctx, block33, block33Root))
	headRoot, err = service.HeadRoot(ctx)
	require.NoError(t, err)
	require.Equal(t, block33Root, [32]byte(headRoot))
	headState, err = service.HeadState(ctx)
	require.NoError(t, err)
	require.Equal(t, headState.Slot(), slot32+1)

	preState34 := transition.NextSlotState(block33Root[:], slot32+2)
	require.NotNil(t, preState34)
	require.Equal(t, slot32+2, preState34.Slot())

	require.LogsContain(t, hook, "pingo")
}
