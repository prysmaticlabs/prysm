package rpc

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BeaconChainServer defines a server implementation of the gRPC Beacon Chain service,
// providing RPC endpoints to access data relevant to the Ethereum 2.0 phase 0
// beacon chain.
type BeaconChainServer struct {
	beaconDB *db.BeaconDB
}

// ListAttestations retrieves attestations by block root, slot, or epoch.
//
// The server may return an empty list when no attestations match the given
// filter criteria. This RPC should not return NOT_FOUND. Only one filter
// criteria should be used.
func (bs *BeaconChainServer) ListAttestations(
	ctx context.Context, req *ethpb.ListAttestationsRequest,
) (*ethpb.ListAttestationsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// AttestationPool retrieves pending attestations.
//
// The server returns a list of attestations that have been seen but not
// yet processed. Pending attestations eventually expire as the slot
// advances, so an attestation missing from this request does not imply
// that it was included in a block. The attestation may have expired.
// Refer to the ethereum 2.0 specification for more details on how
// attestations are processed and when they are no longer valid.
// https://github.com/ethereum/eth2.0-specs/blob/dev/specs/core/0_beacon-chain.md#attestation
func (bs *BeaconChainServer) AttestationPool(
	ctx context.Context, _ *ptypes.Empty,
) (*ethpb.AttestationPoolResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
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
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// GetChainHead retrieves information about the head of the beacon chain from
// the view of the beacon chain node.
//
// This includes the head block slot and root as well as information about
// the most recent finalized and justified slots.
func (bs *BeaconChainServer) GetChainHead(
	ctx context.Context, _ *ptypes.Empty,
) (*ethpb.ChainHead, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// ListValidatorBalances retrieves the validator balances for a given set of public key at
// a specific epoch in time.
//
// TODO(#3045): Implement balances for a specific epoch. Current implementation returns latest balances,
// this is blocked by DB refactor.
func (bs *BeaconChainServer) ListValidatorBalances(
	ctx context.Context,
	req *ethpb.GetValidatorBalancesRequest) (*ethpb.ValidatorBalances, error) {

	res := make([]*ethpb.ValidatorBalances_Balance, 0, len(req.PublicKeys)+len(req.Indices))
	filtered := map[uint64]bool{} // track filtered validators to prevent duplication in the response.

	balances, err := bs.beaconDB.Balances(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve validator balances: %v", err)
	}
	validators, err := bs.beaconDB.Validators(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve validators: %v", err)
	}

	for _, pubKey := range req.PublicKeys {
		index, err := bs.beaconDB.ValidatorIndex(pubKey)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
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
// TODO(#3045): Implement validator set for a specific epoch. Current implementation returns latest set,
// this is blocked by DB refactor.
func (bs *BeaconChainServer) GetValidators(
	ctx context.Context,
	req *ethpb.GetValidatorsRequest) (*ethpb.Validators, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested page size %d can not be greater than max size %d",
			req.PageSize, params.BeaconConfig().MaxPageSize)
	}

	validators, err := bs.beaconDB.Validators(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve validators: %v", err)
	}
	validatorCount := len(validators)

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), validatorCount)
	if err != nil {
		return nil, err
	}

	res := &ethpb.Validators{
		Validators:    validators[start:end],
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
		index, err := bs.beaconDB.ValidatorIndex(pubKey)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not retrieve validator index: %v", err)
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

// GetValidatorParticipation retrieves the validator participation information for a given epoch.
//
// This method returns information about the global participation of
// validator attestations.
func (bs *BeaconChainServer) GetValidatorParticipation(
	ctx context.Context, req *ethpb.GetValidatorParticipationRequest,
) (*ethpb.ValidatorParticipation, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
