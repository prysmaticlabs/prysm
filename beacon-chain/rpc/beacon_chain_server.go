package rpc

import (
	"context"
	"strconv"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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

	validators, err := bs.beaconDB.Validators(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve validators: %v", err)
	}

	if req.PageToken == "" {
		req.PageToken = "0"
	}
	if req.PageSize == 0 {
		req.PageSize = int32(params.BeaconConfig().DefaultPageSize)
	}

	pageSize := int(req.PageSize)
	// Input page size can't be greater than MaxPageSize.
	if pageSize > params.BeaconConfig().MaxPageSize {
		pageSize = params.BeaconConfig().MaxPageSize
	}

	pageToken, err := strconv.Atoi(req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not convert page token: %v", err)
	}

	// Start page can not be greater than validator size.
	start := pageToken * pageSize
	totalSize := len(validators)
	if start >= totalSize {
		return nil, status.Errorf(codes.InvalidArgument, "page start %d >= validator list %d",
			start, totalSize)
	}

	// End page can not go out of bound.
	end := start + pageSize
	if end > totalSize {
		end = totalSize
	}

	res := &ethpb.Validators{
		Validators:    validators[start:end],
		TotalSize:     int32(totalSize),
		NextPageToken: strconv.Itoa(pageToken + 1),
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

// ListValidatorAssignments retrieves the validator assignments for a given epoch.
//
// This request may specify optional validator indices or public keys to
// filter validator assignments.
func (bs *BeaconChainServer) ListValidatorAssignments(
	ctx context.Context, req *ethpb.ListValidatorAssignmentsRequest,
) (*ethpb.ValidatorAssignments, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
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
