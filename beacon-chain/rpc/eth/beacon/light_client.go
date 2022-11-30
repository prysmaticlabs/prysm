package beacon

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// GetLightClientBootstrap - implements https://ethereum.github.io/beacon-APIs/?urls.primaryName=dev#/Beacon/getLightClientBootstrap
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

	// Prepare header
	signedBeaconHeader, err := blk.Header()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get block header: %v", err)
	}
	header := migration.V1Alpha1SignedHeaderToV1(signedBeaconHeader).GetMessage()

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

	data := ethpbv2.LightClientBootstrap{
		Header:                     header,
		CurrentSyncCommittee:       &committee,
		CurrentSyncCommitteeBranch: branch,
	}

	result := &ethpbv2.LightClientBootstrapResponse{
		Version: ethpbv2.Version(blk.Version()),
		Data:    &data,
	}

	return result, nil
}

// GetLightClientUpdatesByRange -
func (bs *Server) GetLightClientUpdatesByRange(ctx context.Context, req *ethpbv2.LightClientUpdatesByRangeRequest) (*ethpbv2.LightClientUpdatesByRangeResponse, error) {
	update1 := ethpbv2.LightClientUpdateWithVersion{
		Version: 1,
		Data: &ethpbv2.LightClientUpdate{
			AttestedHeader:          &ethpbv1.BeaconBlockHeader{},
			NextSyncCommittee:       &ethpbv2.SyncCommittee{},
			NextSyncCommitteeBranch: [][]byte{},
			FinalizedHeader:         &ethpbv1.BeaconBlockHeader{},
			FinalityBranch:          [][]byte{},
			SyncAggregate:           &ethpbv1.SyncAggregate{},
			SignatureSlot:           2,
		},
	}

	update2 := ethpbv2.LightClientUpdateWithVersion{
		Version: 1,
		Data: &ethpbv2.LightClientUpdate{
			AttestedHeader:          &ethpbv1.BeaconBlockHeader{},
			NextSyncCommittee:       &ethpbv2.SyncCommittee{},
			NextSyncCommitteeBranch: [][]byte{},
			FinalizedHeader:         &ethpbv1.BeaconBlockHeader{},
			FinalityBranch:          [][]byte{},
			SyncAggregate:           &ethpbv1.SyncAggregate{},
			SignatureSlot:           3,
		},
	}

	var updates []*ethpbv2.LightClientUpdateWithVersion
	updates = append(updates, &update1)
	updates = append(updates, &update2)

	result := ethpbv2.LightClientUpdatesByRangeResponse{
		Updates: updates,
	}

	return &result, nil
}

// GetLightClientFinalityUpdate - implements https://ethereum.github.io/beacon-APIs/?urls.primaryName=dev#/Beacon/getLightClientFinalityUpdate
func (bs *Server) GetLightClientFinalityUpdate(ctx context.Context, _ *empty.Empty) (*ethpbv2.LightClientFinalityUpdateResponse, error) {
	// Prepare
	ctx, span := trace.StartSpan(ctx, "beacon.GetLightClientFinalityUpdate")
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

	// Get finalized block
	var finalizedRoot [32]byte
	copy(finalizedRoot[:], headState.FinalizedCheckpoint().GetRoot())

	finalizedBlock, err := bs.BeaconDB.Block(ctx, finalizedRoot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get finalized block: %v", err)
	}
	if finalizedBlock == nil || finalizedBlock.IsNil() {
		return nil, status.Errorf(codes.Internal, "No finalized block yet")
	}

	// Get finalized header
	signedFinalizedHeader, err := finalizedBlock.Header()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get finalized block header: %v", err)
	}

	finalizedHeader := migration.V1Alpha1SignedHeaderToV1(signedFinalizedHeader).GetMessage()

	// Get finalityBranch
	finalityBranch, err := headState.FinalizedRootProof(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get finalized root proof: %v", err)
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

	data := ethpbv2.LightClientFinalityUpdate{
		AttestedHeader:  attestedHeader,
		FinalizedHeader: finalizedHeader,
		FinalityBranch:  finalityBranch,
		SyncAggregate:   &syncAggregateV1,
		SignatureSlot:   prevSlot,
	}

	// Return the result
	result := &ethpbv2.LightClientFinalityUpdateResponse{
		Version: ethpbv2.Version(headBlk.Version()),
		Data:    &data,
	}

	return result, nil
}

// GetLightClientOptimisticUpdate - implements https://ethereum.github.io/beacon-APIs/?urls.primaryName=dev#/Beacon/getLightClientOptimisticUpdate
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
