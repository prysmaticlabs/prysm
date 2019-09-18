package rpc

import (
	"context"
	"sort"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BeaconChainServer defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type BeaconChainServer struct {
	beaconDB            db.Database
	ctx                 context.Context
	chainStartFetcher   powchain.ChainStartFetcher
	headFetcher         blockchain.HeadFetcher
	stateFeedListener   blockchain.ChainFeeds
	pool                operations.Pool
	incomingAttestation chan *ethpb.Attestation
	canonicalStateChan  chan *pbp2p.BeaconState
	chainStartChan      chan time.Time
}

// sortableAttestations implements the Sort interface to sort attestations
// by shard as the canonical sorting attribute.
type sortableAttestations []*ethpb.Attestation

func (s sortableAttestations) Len() int      { return len(s) }
func (s sortableAttestations) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableAttestations) Less(i, j int) bool {
	return s[i].Data.Crosslink.Shard < s[j].Data.Crosslink.Shard
}

// ListAttestations retrieves attestations by block root, slot, or epoch.
// Attestations are sorted by crosslink shard by default.
//
// The server may return an empty list when no attestations match the given
// filter criteria. This RPC should not return NOT_FOUND. Only one filter
// criteria should be used.
//
// TODO(#3064): Filtering blocked by DB refactor for easier access to
// fetching data by attributes efficiently.
func (bs *BeaconChainServer) ListAttestations(
	ctx context.Context, req *ethpb.ListAttestationsRequest,
) (*ethpb.ListAttestationsResponse, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}
	switch req.QueryFilter.(type) {
	case *ethpb.ListAttestationsRequest_BlockRoot:
		return nil, status.Error(codes.Unimplemented, "not implemented")
	case *ethpb.ListAttestationsRequest_Slot:
		return nil, status.Error(codes.Unimplemented, "not implemented")
	case *ethpb.ListAttestationsRequest_Epoch:
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}
	atts, err := bs.beaconDB.Attestations(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not fetch attestations: %v", err)
	}
	// We sort attestations according to the Sortable interface.
	sort.Sort(sortableAttestations(atts))
	numAttestations := len(atts)

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numAttestations)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not paginate attestations: %v", err)
	}
	return &ethpb.ListAttestationsResponse{
		Attestations:  atts[start:end],
		TotalSize:     int32(numAttestations),
		NextPageToken: nextPageToken,
	}, nil
}

// AttestationPool retrieves pending attestations.
//
// The server returns a list of attestations that have been seen but not
// yet processed. Pool attestations eventually expire as the slot
// advances, so an attestation missing from this request does not imply
// that it was included in a block. The attestation may have expired.
// Refer to the ethereum 2.0 specification for more details on how
// attestations are processed and when they are no longer valid.
// https://github.com/ethereum/eth2.0-specs/blob/dev/specs/core/0_beacon-chain.md#attestations
func (bs *BeaconChainServer) AttestationPool(
	ctx context.Context, _ *ptypes.Empty,
) (*ethpb.AttestationPoolResponse, error) {
	headBlock, err := bs.beaconDB.(*kv.Store).HeadBlock(ctx)
	if err != nil {
		return nil, err
	}
	if headBlock == nil {
		return nil, status.Error(codes.Internal, "no head block found in db")
	}
	atts, err := bs.pool.AttestationPool(ctx, headBlock.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not fetch attestations: %v", err)
	}
	return &ethpb.AttestationPoolResponse{
		Attestations: atts,
	}, nil
}

// ListBlocks retrieves blocks by root, slot, or epoch.
//
// The server may return multiple blocks in the case that a slot or epoch is
// provided as the filter criteria. The server may return an empty list when
// no blocks in their database match the filter criteria. This RPC should
// not return NOT_FOUND. Only one filter criteria should be used.
func (bs *BeaconChainServer) ListBlocks(
	ctx context.Context, req *ethpb.ListBlocksRequest,
) (*ethpb.ListBlocksResponse, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	switch q := req.QueryFilter.(type) {
	case *ethpb.ListBlocksRequest_Epoch:
		startSlot := q.Epoch * params.BeaconConfig().SlotsPerEpoch
		endSlot := startSlot + params.BeaconConfig().SlotsPerEpoch - 1

		blks, err := bs.beaconDB.Blocks(ctx, filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get blocks: %v", err)
		}

		numBlks := len(blks)
		if numBlks == 0 {
			return &ethpb.ListBlocksResponse{Blocks: make([]*ethpb.BeaconBlock, 0), TotalSize: 0}, nil
		}

		start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not paginate blocks: %v", err)
		}

		return &ethpb.ListBlocksResponse{
			Blocks:        blks[start:end],
			TotalSize:     int32(numBlks),
			NextPageToken: nextPageToken,
		}, nil

	case *ethpb.ListBlocksRequest_Root:
		blk, err := bs.beaconDB.Block(ctx, bytesutil.ToBytes32(q.Root))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve block: %v", err)
		}

		if blk == nil {
			return &ethpb.ListBlocksResponse{Blocks: []*ethpb.BeaconBlock{}, TotalSize: 0}, nil
		}

		return &ethpb.ListBlocksResponse{
			Blocks:    []*ethpb.BeaconBlock{blk},
			TotalSize: 1,
		}, nil

	case *ethpb.ListBlocksRequest_Slot:
		blks, err := bs.beaconDB.Blocks(ctx, filters.NewFilter().SetStartSlot(q.Slot).SetEndSlot(q.Slot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve blocks for slot %d: %v", q.Slot, err)
		}

		numBlks := len(blks)
		if numBlks == 0 {
			return &ethpb.ListBlocksResponse{Blocks: []*ethpb.BeaconBlock{}, TotalSize: 0}, nil
		}

		start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numBlks)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not paginate blocks: %v", err)
		}

		return &ethpb.ListBlocksResponse{
			Blocks:        blks[start:end],
			TotalSize:     int32(numBlks),
			NextPageToken: nextPageToken,
		}, nil
	}

	return nil, status.Errorf(codes.InvalidArgument, "must satisfy one of the filter requirement")
}

// GetChainHead retrieves information about the head of the beacon chain from
// the view of the beacon chain node.
//
// This includes the head block slot and root as well as information about
// the most recent finalized and justified slots.
func (bs *BeaconChainServer) GetChainHead(ctx context.Context, _ *ptypes.Empty) (*ethpb.ChainHead, error) {
	finalizedCheckpoint := bs.headFetcher.HeadState().FinalizedCheckpoint
	justifiedCheckpoint := bs.headFetcher.HeadState().CurrentJustifiedCheckpoint
	prevJustifiedCheckpoint := bs.headFetcher.HeadState().PreviousJustifiedCheckpoint

	return &ethpb.ChainHead{
		BlockRoot:                  bs.headFetcher.HeadRoot(),
		BlockSlot:                  bs.headFetcher.HeadSlot(),
		FinalizedBlockRoot:         finalizedCheckpoint.Root,
		FinalizedSlot:              finalizedCheckpoint.Epoch * params.BeaconConfig().SlotsPerEpoch,
		JustifiedBlockRoot:         justifiedCheckpoint.Root,
		JustifiedSlot:              justifiedCheckpoint.Epoch * params.BeaconConfig().SlotsPerEpoch,
		PreviousJustifiedBlockRoot: prevJustifiedCheckpoint.Root,
		PreviousJustifiedSlot:      prevJustifiedCheckpoint.Epoch * params.BeaconConfig().SlotsPerEpoch,
	}, nil
}

// ListValidatorBalances retrieves the validator balances for a given set of public key at
// a specific epoch in time.
//
// TODO(#3064): Implement balances for a specific epoch. Current implementation returns latest balances,
// this is blocked by DB refactor.
func (bs *BeaconChainServer) ListValidatorBalances(
	ctx context.Context,
	req *ethpb.GetValidatorBalancesRequest) (*ethpb.ValidatorBalances, error) {

	res := make([]*ethpb.ValidatorBalances_Balance, 0, len(req.PublicKeys)+len(req.Indices))
	filtered := map[uint64]bool{} // track filtered validators to prevent duplication in the response.

	headState, err := bs.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve head state: %v", err)
	}
	balances := headState.Balances
	validators := headState.Validators

	for _, pubKey := range req.PublicKeys {
		// Skip empty public key
		if len(pubKey) == 0 {
			continue
		}

		index, ok, err := bs.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
		}
		if !ok {
			return nil, status.Errorf(codes.Internal, "could not find validator index for public key  %#x not found", pubKey)
		}

		filtered[index] = true

		if int(index) >= len(balances) {
			return nil, status.Errorf(codes.InvalidArgument, "validator index %d >= balance list %d",
				index, len(balances))
		}

		res = append(res, &ethpb.ValidatorBalances_Balance{
			PublicKey: pubKey,
			Index:     index,
			Balance:   balances[index],
		})
	}

	for _, index := range req.Indices {
		if int(index) >= len(balances) {
			return nil, status.Errorf(codes.InvalidArgument, "validator index %d >= balance list %d",
				index, len(balances))
		}

		if !filtered[index] {
			res = append(res, &ethpb.ValidatorBalances_Balance{
				PublicKey: validators[index].PublicKey,
				Index:     index,
				Balance:   balances[index],
			})
		}
	}
	return &ethpb.ValidatorBalances{Balances: res}, nil
}

// GetValidators retrieves the current list of active validators with an optional historical epoch flag to
// to retrieve validator set in time.
//
// TODO(#3064): Implement validator set for a specific epoch. Current implementation returns latest set,
// this is blocked by DB refactor.
func (bs *BeaconChainServer) GetValidators(
	ctx context.Context,
	req *ethpb.GetValidatorsRequest) (*ethpb.Validators, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	headState, err := bs.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get head state %v", err)
	}

	validatorCount := len(headState.Validators)
	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), validatorCount)
	if err != nil {
		return nil, err
	}

	res := &ethpb.Validators{
		Validators:    headState.Validators[start:end],
		TotalSize:     int32(validatorCount),
		NextPageToken: nextPageToken,
	}
	return res, nil
}

// GetValidatorActiveSetChanges retrieves the active set changes for a given epoch.
//
// This data includes any activations, voluntary exits, and involuntary
// ejections.
func (bs *BeaconChainServer) GetValidatorActiveSetChanges(
	ctx context.Context, req *ethpb.GetValidatorActiveSetChangesRequest,
) (*ethpb.ActiveSetChanges, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// GetValidatorQueue retrieves the current validator queue information.
func (bs *BeaconChainServer) GetValidatorQueue(
	ctx context.Context, _ *ptypes.Empty,
) (*ethpb.ValidatorQueue, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// ListValidatorAssignments retrieves the validator assignments for a given epoch,
// optional validator indices or public keys may be included to filter validator assignments.
//
// TODO(#3045): Implement validator set for a specific epoch. Current implementation returns latest set,
// this is blocked by DB refactor.
func (bs *BeaconChainServer) ListValidatorAssignments(
	ctx context.Context, req *ethpb.ListValidatorAssignmentsRequest,
) (*ethpb.ValidatorAssignments, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	e := req.Epoch
	s, err := bs.beaconDB.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve current state: %v", err)
	}

	var res []*ethpb.ValidatorAssignments_CommitteeAssignment
	filtered := map[uint64]bool{} // track filtered validators to prevent duplication in the response.

	// Filter out assignments by public keys.
	for _, pubKey := range req.PublicKeys {
		index, ok, err := bs.beaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
		}
		if !ok {
			return nil, status.Errorf(codes.Internal, "could not find validator index for public key  %#x not found", pubKey)
		}

		filtered[index] = true

		if int(index) >= len(s.Validators) {
			return nil, status.Errorf(codes.InvalidArgument, "validator index %d >= validator count %d",
				index, len(s.Validators))
		}

		committee, shard, slot, isProposer, err := helpers.CommitteeAssignment(s, e, index)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve assignment for validator %d: %v", index, err)
		}

		res = append(res, &ethpb.ValidatorAssignments_CommitteeAssignment{
			CrosslinkCommittees: committee,
			Shard:               shard,
			Slot:                slot,
			Proposer:            isProposer,
			PublicKey:           pubKey,
		})
	}

	// Filter out assignments by validator indices.
	for _, index := range req.Indices {
		if int(index) >= len(s.Validators) {
			return nil, status.Errorf(codes.InvalidArgument, "validator index %d >= validator count %d",
				index, len(s.Validators))
		}

		if !filtered[index] {
			committee, shard, slot, isProposer, err := helpers.CommitteeAssignment(s, e, index)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "could not retrieve assignment for validator %d: %v", index, err)
			}

			res = append(res, &ethpb.ValidatorAssignments_CommitteeAssignment{
				CrosslinkCommittees: committee,
				Shard:               shard,
				Slot:                slot,
				Proposer:            isProposer,
				PublicKey:           s.Validators[index].PublicKey,
			})
		}
	}

	// Return filtered assignments with pagination.
	if len(res) > 0 {
		start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), len(res))
		if err != nil {
			return nil, err
		}

		return &ethpb.ValidatorAssignments{
			Epoch:         e,
			Assignments:   res[start:end],
			NextPageToken: nextPageToken,
			TotalSize:     int32(len(res)),
		}, nil
	}

	// If no filter was specified, return assignments from active validator indices with pagination.
	activeIndices, err := helpers.ActiveValidatorIndices(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve active validator indices: %v", err)
	}

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), len(activeIndices))
	if err != nil {
		return nil, err
	}

	for _, index := range activeIndices[start:end] {
		committee, shard, slot, isProposer, err := helpers.CommitteeAssignment(s, e, index)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve assignment for validator %d: %v", index, err)
		}

		res = append(res, &ethpb.ValidatorAssignments_CommitteeAssignment{
			CrosslinkCommittees: committee,
			Shard:               shard,
			Slot:                slot,
			Proposer:            isProposer,
			PublicKey:           s.Validators[index].PublicKey,
		})
	}

	return &ethpb.ValidatorAssignments{
		Epoch:         e,
		Assignments:   res,
		NextPageToken: nextPageToken,
		TotalSize:     int32(len(res)),
	}, nil
}

// GetValidatorParticipation retrieves the validator participation information for a given epoch,
// it returns the information about validator's participation rate in voting on the proof of stake
// rules based on their balance compared to the total active validator balance.
func (bs *BeaconChainServer) GetValidatorParticipation(
	ctx context.Context, req *ethpb.GetValidatorParticipationRequest,
) (*ethpb.ValidatorParticipation, error) {
	headState := bs.headFetcher.HeadState()
	currentEpoch := helpers.SlotToEpoch(headState.Slot)
	if req.Epoch > helpers.SlotToEpoch(headState.Slot) {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"cannot request data from an epoch in the future: req.Epoch %d, currentEpoch %d", req.Epoch, currentEpoch,
		)
	}
	// If the request is from a past epoch, we look into our archived
	// data to find it and return it if it exists.
	if req.Epoch < helpers.SlotToEpoch(headState.Slot) {
		participation, err := bs.beaconDB.ArchivedValidatorParticipation(ctx, req.Epoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not fetch archived participation: %v", err)
		}
		return participation, nil
	}
	// Else if the request is for the current epoch, we compute validator participation
	// right away and return the result based on the head state.
	participation, err := epoch.ComputeValidatorParticipation(headState)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not compute participation: %v", err)
	}
	return participation, nil
}
