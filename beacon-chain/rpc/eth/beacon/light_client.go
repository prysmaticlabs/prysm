package beacon

import (
	"bytes"
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// GetLightClientBootstrap - implements https://github.com/ethereum/beacon-APIs/blob/263f4ed6c263c967f13279c7a9f5629b51c5fc55/apis/beacon/light_client/bootstrap.yaml
func (bs *Server) GetLightClientBootstrap(ctx context.Context, req *ethpbv2.LightClientBootstrapRequest) (*ethpbv2.LightClientBootstrapResponse, error) {
	// Prepare
	ctx, span := trace.StartSpan(ctx, "beacon.GetLightClientBootstrap")
	defer span.End()

	// Get the block
	var blockRoot [32]byte
	copy(blockRoot[:], req.BlockRoot)

	blk, err := bs.BeaconDB.Block(ctx, blockRoot)
	err = handleGetBlockError(blk, err)
	if err != nil {
		return nil, err
	}

	// Get the state
	state, err := bs.StateFetcher.StateBySlot(ctx, blk.Block().Slot())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state by slot: %v", err)
	}

	bootstrap, err := createLightClientBootstrap(ctx, state)
	if err != nil {
		return nil, err
	}

	result := &ethpbv2.LightClientBootstrapResponse{
		Version: ethpbv2.Version(blk.Version()),
		Data:    bootstrap,
	}

	return result, nil
}

// In https://github.com/ethereum/consensus-specs/blob/3d235740e5f1e641d3b160c8688f26e7dc5a1894/specs/altair/light-client/full-node.md#create_light_client_bootstrap
// def create_light_client_bootstrap(state: BeaconState) -> LightClientBootstrap:
//
//	assert compute_epoch_at_slot(state.slot) >= ALTAIR_FORK_EPOCH
//	assert state.slot == state.latest_block_header.slot
//
//	return LightClientBootstrap(
//	    header=BeaconBlockHeader(
//	        slot=state.latest_block_header.slot,
//	        proposer_index=state.latest_block_header.proposer_index,
//	        parent_root=state.latest_block_header.parent_root,
//	        state_root=hash_tree_root(state),
//	        body_root=state.latest_block_header.body_root,
//	    ),
//	    current_sync_committee=state.current_sync_committee,
//	    current_sync_committee_branch=compute_merkle_proof_for_state(state, CURRENT_SYNC_COMMITTEE_INDEX)
//	)
func createLightClientBootstrap(ctx context.Context, state state.BeaconState) (*ethpbv2.LightClientBootstrap, error) {
	// assert compute_epoch_at_slot(state.slot) >= ALTAIR_FORK_EPOCH
	if slots.ToEpoch(state.Slot()) < params.BeaconConfig().AltairForkEpoch {
		return nil, status.Errorf(codes.Internal, "Invalid state slot: %d", state.Slot())
	}

	// assert state.slot == state.latest_block_header.slot
	if state.Slot() != state.LatestBlockHeader().Slot {
		return nil, status.Errorf(codes.Internal, "Invalid state slot: %d", state.Slot())
	}

	// Prepare data
	latestBlockHeader := state.LatestBlockHeader()

	currentSyncCommittee, err := state.CurrentSyncCommittee()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get current sync committee: %v", err)
	}

	committee := ethpbv2.SyncCommittee{
		Pubkeys:         currentSyncCommittee.GetPubkeys(),
		AggregatePubkey: currentSyncCommittee.GetAggregatePubkey(),
	}

	branch, err := state.CurrentSyncCommitteeProof(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get current sync committee proof: %v", err)
	}

	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state root: %v", err)
	}

	// Return result
	result := &ethpbv2.LightClientBootstrap{
		Header: &ethpbv1.BeaconBlockHeader{
			Slot:          latestBlockHeader.Slot,
			ProposerIndex: latestBlockHeader.ProposerIndex,
			ParentRoot:    latestBlockHeader.ParentRoot,
			StateRoot:     stateRoot[:],
			BodyRoot:      latestBlockHeader.BodyRoot,
		},
		CurrentSyncCommittee:       &committee,
		CurrentSyncCommitteeBranch: branch,
	}

	return result, nil
}

// GetLightClientUpdatesByRange - implements https://github.com/ethereum/beacon-APIs/blob/263f4ed6c263c967f13279c7a9f5629b51c5fc55/apis/beacon/light_client/updates.yaml
func (bs *Server) GetLightClientUpdatesByRange(ctx context.Context, req *ethpbv2.LightClientUpdatesByRangeRequest) (*ethpbv2.LightClientUpdatesByRangeResponse, error) {
	// Prepare
	ctx, span := trace.StartSpan(ctx, "beacon.GetLightClientUpdatesByRange")
	defer span.End()

	// Determine slots per period
	config := params.BeaconConfig()
	slotsPerPeriod := uint64(config.EpochsPerSyncCommitteePeriod) * uint64(config.SlotsPerEpoch)

	// Adjust count based on configuration
	count := uint64(req.Count)
	if count > config.MaxRequestLightClientUpdates {
		count = config.MaxRequestLightClientUpdates
	}

	// Determine the start and end periods
	startPeriod := req.StartPeriod
	endPeriod := startPeriod + count - 1

	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	lHeadSlot := uint64(headState.Slot())
	headPeriod := lHeadSlot / slotsPerPeriod
	if headPeriod < endPeriod {
		endPeriod = headPeriod
	}

	// Populate updates
	var updates []*ethpbv2.LightClientUpdateWithVersion
	for period := startPeriod; period <= endPeriod; period++ {
		// Get the last known state of the period,
		//    1. We wish the block has a parent in the same period if possible
		//	  2. We wish the block has a state in the same period
		lLastSlotInPeriod := period*slotsPerPeriod + slotsPerPeriod - 1
		if lLastSlotInPeriod > lHeadSlot {
			lLastSlotInPeriod = lHeadSlot
		}
		lFirstSlotInPeriod := period * slotsPerPeriod

		var state state.BeaconState
		for lSlot := lLastSlotInPeriod; lSlot >= lFirstSlotInPeriod; lSlot-- {
			state, err = bs.StateFetcher.StateBySlot(ctx, types.Slot(lSlot))
			if err == nil {
				break
			}
		}

		if state == nil {
			// No valid state found for the period
			continue
		}

		// Get the block
		slot := state.Slot()
		blocks, err := bs.BeaconDB.BlocksBySlot(ctx, slot)
		if err != nil || len(blocks) == 0 {
			continue
		}
		block := blocks[0]

		// Get attested state
		attestedRoot := block.Block().ParentRoot()
		attestedBlock, err := bs.BeaconDB.Block(ctx, attestedRoot)
		if err != nil || attestedBlock == nil {
			continue
		}

		attestedSlot := attestedBlock.Block().Slot()
		attestedState, err := bs.StateFetcher.StateBySlot(ctx, attestedSlot)
		if err != nil {
			continue
		}

		// Get finalized block
		var finalizedBlock interfaces.SignedBeaconBlock
		finalizedCheckPoint := attestedState.FinalizedCheckpoint()
		if finalizedCheckPoint != nil {
			finalizedRoot := bytesutil.ToBytes32(finalizedCheckPoint.Root)
			finalizedBlock, err = bs.BeaconDB.Block(ctx, finalizedRoot)
		}

		update, err := createLightClientUpdate(
			ctx,
			config,
			slotsPerPeriod,
			state,
			block,
			attestedState,
			finalizedBlock,
		)

		if err == nil {
			updates = append(updates, &ethpbv2.LightClientUpdateWithVersion{
				Version: ethpbv2.Version(attestedState.Version()),
				Data:    update,
			})
		}
	}

	if len(updates) == 0 {
		return nil, status.Errorf(codes.NotFound, "No updates found")
	}

	result := ethpbv2.LightClientUpdatesByRangeResponse{
		Updates: updates,
	}

	return &result, nil
}

// In https://github.com/ethereum/consensus-specs/blob/d70dcd9926a4bbe987f1b4e65c3e05bd029fcfb8/specs/altair/light-client/full-node.md#create_light_client_update
// def create_light_client_update(state: BeaconState,
//
//	                           block: SignedBeaconBlock,
//	                           attested_state: BeaconState,
//	                           finalized_block: Optional[SignedBeaconBlock]) -> LightClientUpdate:
//	assert compute_epoch_at_slot(attested_state.slot) >= ALTAIR_FORK_EPOCH
//	assert sum(block.message.body.sync_aggregate.sync_committee_bits) >= MIN_SYNC_COMMITTEE_PARTICIPANTS
//
//	assert state.slot == state.latest_block_header.slot
//	header = state.latest_block_header.copy()
//	header.state_root = hash_tree_root(state)
//	assert hash_tree_root(header) == hash_tree_root(block.message)
//	update_signature_period = compute_sync_committee_period(compute_epoch_at_slot(block.message.slot))
//
//	assert attested_state.slot == attested_state.latest_block_header.slot
//	attested_header = attested_state.latest_block_header.copy()
//	attested_header.state_root = hash_tree_root(attested_state)
//	assert hash_tree_root(attested_header) == block.message.parent_root
//	update_attested_period = compute_sync_committee_period(compute_epoch_at_slot(attested_header.slot))
//
//	# `next_sync_committee` is only useful if the message is signed by the current sync committee
//	if update_attested_period == update_signature_period:
//	    next_sync_committee = attested_state.next_sync_committee
//	    next_sync_committee_branch = compute_merkle_proof_for_state(attested_state, NEXT_SYNC_COMMITTEE_INDEX)
//	else:
//	    next_sync_committee = SyncCommittee()
//	    next_sync_committee_branch = [Bytes32() for _ in range(floorlog2(NEXT_SYNC_COMMITTEE_INDEX))]
//
//	# Indicate finality whenever possible
//	if finalized_block is not None:
//	    if finalized_block.message.slot != GENESIS_SLOT:
//	        finalized_header = BeaconBlockHeader(
//	            slot=finalized_block.message.slot,
//	            proposer_index=finalized_block.message.proposer_index,
//	            parent_root=finalized_block.message.parent_root,
//	            state_root=finalized_block.message.state_root,
//	            body_root=hash_tree_root(finalized_block.message.body),
//	        )
//	        assert hash_tree_root(finalized_header) == attested_state.finalized_checkpoint.root
//	    else:
//	        assert attested_state.finalized_checkpoint.root == Bytes32()
//	        finalized_header = BeaconBlockHeader()
//	    finality_branch = compute_merkle_proof_for_state(attested_state, FINALIZED_ROOT_INDEX)
//	else:
//	    finalized_header = BeaconBlockHeader()
//	    finality_branch = [Bytes32() for _ in range(floorlog2(FINALIZED_ROOT_INDEX))]
//
//	return LightClientUpdate(
//	    attested_header=attested_header,
//	    next_sync_committee=next_sync_committee,
//	    next_sync_committee_branch=next_sync_committee_branch,
//	    finalized_header=finalized_header,
//	    finality_branch=finality_branch,
//	    sync_aggregate=block.message.body.sync_aggregate,
//	    signature_slot=block.message.slot,
//	)
func createLightClientUpdate(
	ctx context.Context,
	config *params.BeaconChainConfig,
	slotsPerPeriod uint64,
	state state.BeaconState,
	block interfaces.SignedBeaconBlock,
	attestedState state.BeaconState,
	finalizedBlock interfaces.SignedBeaconBlock) (*ethpbv2.LightClientUpdate, error) {

	// assert compute_epoch_at_slot(attested_state.slot) >= ALTAIR_FORK_EPOCH
	attestedEpoch := types.Epoch(uint64(attestedState.Slot()) / uint64(config.SlotsPerEpoch))
	if attestedEpoch < types.Epoch(config.AltairForkEpoch) {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid attested epoch: %d", attestedEpoch)
	}

	// assert sum(block.message.body.sync_aggregate.sync_committee_bits) >= MIN_SYNC_COMMITTEE_PARTICIPANTS
	syncAggregate, err := block.Block().Body().SyncAggregate()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync aggregate: %v", err)
	}

	if syncAggregate.SyncCommitteeBits.Count() < config.MinSyncCommitteeParticipants {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid sync committee bits count: %d", syncAggregate.SyncCommitteeBits.Count())
	}

	// assert state.slot == state.latest_block_header.slot
	if state.Slot() != state.LatestBlockHeader().Slot {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid state slot: %d", state.Slot())
	}

	// assert hash_tree_root(header) == hash_tree_root(block.message)
	header := *state.LatestBlockHeader()
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get state root: %v", err)
	}
	header.StateRoot = stateRoot[:]

	headerRoot, err := header.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get header root: %v", err)
	}

	blockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block root: %v", err)
	}

	if headerRoot != blockRoot {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid header root: %v", headerRoot)
	}

	// update_signature_period = compute_sync_committee_period(compute_epoch_at_slot(block.message.slot))
	updateSignaturePeriod := uint64(block.Block().Slot()) / slotsPerPeriod

	// assert attested_state.slot == attested_state.latest_block_header.slot
	if attestedState.Slot() != attestedState.LatestBlockHeader().Slot {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid attested state slot: %d", attestedState.Slot())
	}

	// attested_header = attested_state.latest_block_header.copy()
	attestedHeader := *attestedState.LatestBlockHeader()

	// attested_header.state_root = hash_tree_root(attested_state)
	attestedStateRoot, err := attestedState.HashTreeRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get attested state root: %v", err)
	}
	attestedHeader.StateRoot = attestedStateRoot[:]

	// assert hash_tree_root(attested_header) == block.message.parent_root
	attestedHeaderRoot, err := attestedHeader.HashTreeRoot()
	if attestedHeaderRoot != block.Block().ParentRoot() {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid attested header root: %v", attestedHeaderRoot)
	}

	// update_attested_period = compute_sync_committee_period(compute_epoch_at_slot(attested_header.slot))
	updateAttestedPeriod := uint64(attestedHeader.Slot) / slotsPerPeriod

	// Generate next sync committee and proof
	var nextSyncCommittee *ethpbv2.SyncCommittee
	var nextSyncCommitteeBranch [][]byte
	if updateAttestedPeriod == updateSignaturePeriod {
		tempNextSyncCommittee, err := attestedState.NextSyncCommittee()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get next sync committee: %v", err)
		}

		nextSyncCommittee = &ethpbv2.SyncCommittee{
			Pubkeys:         tempNextSyncCommittee.Pubkeys,
			AggregatePubkey: tempNextSyncCommittee.AggregatePubkey,
		}

		nextSyncCommitteeBranch, err = attestedState.NextSyncCommitteeProof(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get next sync committee proof: %v", err)
		}
	} else {
		pubKeys := make([][]byte, config.SyncCommitteeSize)
		for i := 0; i < int(config.SyncCommitteeSize); i++ {
			pubKeys[i] = make([]byte, 48)
		}
		nextSyncCommittee = &ethpbv2.SyncCommittee{
			Pubkeys:         pubKeys,
			AggregatePubkey: make([]byte, 48),
		}

		nextSyncCommitteeBranch = make([][]byte, 5)
		for i := 0; i < 5; i++ {
			nextSyncCommitteeBranch[i] = make([]byte, 32)
		}
	}

	// Indicate finality whenever possible
	var finalizedHeader *ethpbv1.BeaconBlockHeader
	var finalityBranch [][]byte
	if finalizedBlock != nil {
		if finalizedBlock.Block().Slot() != 0 {
			tempFinalizedHeader, err := finalizedBlock.Header()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get finalized header: %v", err)
			}
			finalizedHeader = migration.V1Alpha1SignedHeaderToV1(tempFinalizedHeader).GetMessage()

			finalizedHeaderRoot, err := finalizedHeader.HashTreeRoot()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get finalized header root: %v", err)
			}

			if finalizedHeaderRoot != bytesutil.ToBytes32(attestedState.FinalizedCheckpoint().Root) {
				return nil, status.Errorf(codes.InvalidArgument, "Invalid finalized header root: %v", finalizedHeaderRoot)
			}
		} else {
			if !bytes.Equal(attestedState.FinalizedCheckpoint().Root, make([]byte, 32)) {
				return nil, status.Errorf(codes.InvalidArgument, "Invalid finalized header root: %v", attestedState.FinalizedCheckpoint().Root)
			}

			finalizedHeader = &ethpbv1.BeaconBlockHeader{
				Slot:          0,
				ProposerIndex: 0,
				ParentRoot:    make([]byte, 32),
				StateRoot:     make([]byte, 32),
				BodyRoot:      make([]byte, 32),
			}
		}

		finalityBranch, err = attestedState.FinalizedRootProof(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get finalized root proof: %v", err)
		}
	} else {
		finalizedHeader = &ethpbv1.BeaconBlockHeader{
			Slot:          0,
			ProposerIndex: 0,
			ParentRoot:    make([]byte, 32),
			StateRoot:     make([]byte, 32),
			BodyRoot:      make([]byte, 32),
		}

		finalityBranch = make([][]byte, 6)
		for i := 0; i < 6; i++ {
			finalityBranch[i] = make([]byte, 32)
		}
	}

	// Return result
	attestedHeaderResult := &ethpbv1.BeaconBlockHeader{
		Slot:          attestedHeader.Slot,
		ProposerIndex: attestedHeader.ProposerIndex,
		ParentRoot:    attestedHeader.ParentRoot,
		StateRoot:     attestedHeader.StateRoot,
		BodyRoot:      attestedHeader.BodyRoot,
	}

	syncAggregateResult := &ethpbv1.SyncAggregate{
		SyncCommitteeBits:      syncAggregate.SyncCommitteeBits,
		SyncCommitteeSignature: syncAggregate.SyncCommitteeSignature,
	}

	result := &ethpbv2.LightClientUpdate{
		AttestedHeader:          attestedHeaderResult,
		NextSyncCommittee:       nextSyncCommittee,
		NextSyncCommitteeBranch: nextSyncCommitteeBranch,
		FinalizedHeader:         finalizedHeader,
		FinalityBranch:          finalityBranch,
		SyncAggregate:           syncAggregateResult,
		SignatureSlot:           uint64(block.Block().Slot()),
	}

	return result, nil
}

// GetLightClientFinalityUpdate - implements https://github.com/ethereum/beacon-APIs/blob/263f4ed6c263c967f13279c7a9f5629b51c5fc55/apis/beacon/light_client/finality_update.yaml
func (bs *Server) GetLightClientFinalityUpdate(ctx context.Context, _ *empty.Empty) (*ethpbv2.LightClientFinalityUpdateResponse, error) {
	// Prepare
	ctx, span := trace.StartSpan(ctx, "beacon.GetLightClientFinalityUpdate")
	defer span.End()

	// Determine slots per period
	config := params.BeaconConfig()
	slotsPerPeriod := uint64(config.EpochsPerSyncCommitteePeriod) * uint64(config.SlotsPerEpoch)

	// Get the current state
	state, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Get the block
	latestBlockHeader := state.LatestBlockHeader()
	latestBlockHeaderRoot, err := latestBlockHeader.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get latest block header root: %v", err)
	}

	block, err := bs.BeaconDB.Block(ctx, latestBlockHeaderRoot)

	// Get attested state
	attestedRoot := block.Block().ParentRoot()
	attestedBlock, err := bs.BeaconDB.Block(ctx, attestedRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get attested block: %v", err)
	}

	attestedSlot := attestedBlock.Block().Slot()
	attestedState, err := bs.StateFetcher.StateBySlot(ctx, attestedSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get attested state: %v", err)
	}

	// Get finalized block
	var finalizedBlock interfaces.SignedBeaconBlock
	finalizedCheckPoint := attestedState.FinalizedCheckpoint()
	if finalizedCheckPoint != nil {
		finalizedRoot := bytesutil.ToBytes32(finalizedCheckPoint.Root)
		finalizedBlock, err = bs.BeaconDB.Block(ctx, finalizedRoot)
	}

	update, err := createLightClientUpdate(
		ctx,
		config,
		slotsPerPeriod,
		state,
		block,
		attestedState,
		finalizedBlock,
	)

	if err != nil {
		return nil, err
	}

	finalityUpdate := createLightClientFinalityUpdate(update)

	// Return the result
	result := &ethpbv2.LightClientFinalityUpdateResponse{
		Version: ethpbv2.Version(attestedState.Version()),
		Data:    finalityUpdate,
	}

	return result, nil
}

// In https://github.com/ethereum/consensus-specs/blob/3d235740e5f1e641d3b160c8688f26e7dc5a1894/specs/altair/light-client/full-node.md#create_light_client_finality_update
// def create_light_client_finality_update(update: LightClientUpdate) -> LightClientFinalityUpdate:
//
//	return LightClientFinalityUpdate(
//	    attested_header=update.attested_header,
//	    finalized_header=update.finalized_header,
//	    finality_branch=update.finality_branch,
//	    sync_aggregate=update.sync_aggregate,
//	    signature_slot=update.signature_slot,
//	)
func createLightClientFinalityUpdate(update *ethpbv2.LightClientUpdate) *ethpbv2.LightClientFinalityUpdate {
	return &ethpbv2.LightClientFinalityUpdate{
		AttestedHeader:  update.AttestedHeader,
		FinalizedHeader: update.FinalizedHeader,
		FinalityBranch:  update.FinalityBranch,
		SyncAggregate:   update.SyncAggregate,
		SignatureSlot:   update.SignatureSlot,
	}
}

// GetLightClientOptimisticUpdate - implements https://github.com/ethereum/beacon-APIs/blob/263f4ed6c263c967f13279c7a9f5629b51c5fc55/apis/beacon/light_client/optimistic_update.yaml
func (bs *Server) GetLightClientOptimisticUpdate(ctx context.Context, _ *empty.Empty) (*ethpbv2.LightClientOptimisticUpdateResponse, error) {
	// Prepare
	ctx, span := trace.StartSpan(ctx, "beacon.GetLightClientOptimisticUpdate")
	defer span.End()

	// Get head state
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Get the attestedHeader
	prevSlot := slots.PrevSlot(headState.Slot())
	prevBlockRoot := bytesutil.ToBytes32(headState.BlockRoots()[prevSlot%params.BeaconConfig().SlotsPerHistoricalRoot])
	prevBlock, err := bs.BeaconDB.Block(ctx, prevBlockRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block: %v", err)
	}
	prevBlockSignedHeader, err := prevBlock.Header()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block header: %v", err)
	}

	attestedHeader := migration.V1Alpha1SignedHeaderToV1(prevBlockSignedHeader).GetMessage()

	// Get head block
	headBlk, err := bs.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head block: %v", err)
	}

	// Get SyncAggregate
	syncAggregate, err := headBlk.Block().Body().SyncAggregate()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync aggregate: %v", err)
	}

	syncAggregateV1 := ethpbv1.SyncAggregate{
		SyncCommitteeBits:      syncAggregate.SyncCommitteeBits,
		SyncCommitteeSignature: syncAggregate.SyncCommitteeSignature,
	}

	data := ethpbv2.LightClientOptimisticUpdate{
		AttestedHeader: attestedHeader,
		SyncAggregate:  &syncAggregateV1,
		SignatureSlot:  prevSlot,
	}

	// Return the result
	result := &ethpbv2.LightClientOptimisticUpdateResponse{
		Version: ethpbv2.Version(headBlk.Version()),
		Data:    &data,
	}

	return result, nil
}
