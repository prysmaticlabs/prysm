package beacon

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/api/grpc"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	butil "github.com/prysmaticlabs/prysm/encoding/bytes"
	"github.com/prysmaticlabs/prysm/proto/eth/v2"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListSyncCommittees retrieves the sync committees for the given epoch.
// If the epoch is not passed in, then the sync committees for the epoch of the state will be obtained.
func (bs *Server) ListSyncCommittees(ctx context.Context, req *ethpbv2.StateSyncCommitteesRequest) (*ethpbv2.StateSyncCommitteesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListSyncCommittees")
	defer span.End()

	st, err := bs.stateFromRequest(ctx, &stateRequest{
		epoch:   req.Epoch,
		stateId: req.StateId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not fetch beacon state using request: %v", err)
	}

	// Get the current sync committee and sync committee indices from the state.
	committeeIndices, committee, err := currentCommitteeIndicesFromState(st)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync committee indices from state: %v", err)
	}
	subcommittees, err := extractSyncSubcommittees(st, committee)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not extract sync subcommittees: %v", err)
	}

	return &ethpbv2.StateSyncCommitteesResponse{
		Data: &ethpbv2.SyncCommitteeValidators{
			Validators:          committeeIndices,
			ValidatorAggregates: subcommittees,
		},
	}, nil
}

func currentCommitteeIndicesFromState(st state.BeaconState) ([]types.ValidatorIndex, *ethpbalpha.SyncCommittee, error) {
	committee, err := st.CurrentSyncCommittee()
	if err != nil {
		return nil, nil, fmt.Errorf(
			"could not get sync committee: %v", err,
		)
	}

	committeeIndices := make([]types.ValidatorIndex, len(committee.Pubkeys))
	for i, key := range committee.Pubkeys {
		index, ok := st.ValidatorIndexByPubkey(butil.ToBytes48(key))
		if !ok {
			return nil, nil, fmt.Errorf(
				"validator index not found for pubkey %#x",
				butil.Trunc(key),
			)
		}
		committeeIndices[i] = index
	}
	return committeeIndices, committee, nil
}

func extractSyncSubcommittees(st state.BeaconState, committee *ethpbalpha.SyncCommittee) ([]*eth.SyncSubcommitteeValidators, error) {
	subcommitteeCount := params.BeaconConfig().SyncCommitteeSubnetCount
	subcommittees := make([]*ethpbv2.SyncSubcommitteeValidators, subcommitteeCount)
	for i := uint64(0); i < subcommitteeCount; i++ {
		pubkeys, err := altair.SyncSubCommitteePubkeys(committee, types.CommitteeIndex(i))
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get subcommittee pubkeys: %v", err,
			)
		}
		subcommittee := &ethpbv2.SyncSubcommitteeValidators{Validators: make([]types.ValidatorIndex, len(pubkeys))}
		for j, key := range pubkeys {
			index, ok := st.ValidatorIndexByPubkey(butil.ToBytes48(key))
			if !ok {
				return nil, fmt.Errorf(
					"validator index not found for pubkey %#x",
					butil.Trunc(key),
				)
			}
			subcommittee.Validators[j] = index
		}
		subcommittees[i] = subcommittee
	}
	return subcommittees, nil
}

// SubmitPoolSyncCommitteeSignatures submits sync committee signature objects to the node.
func (bs *Server) SubmitPoolSyncCommitteeSignatures(ctx context.Context, req *ethpbv2.SubmitPoolSyncCommitteeSignatures) (*empty.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitPoolSyncCommitteeSignatures")
	defer span.End()

	var validMessages []*ethpbalpha.SyncCommitteeMessage
	var msgFailures []*helpers.SingleIndexedVerificationFailure
	for i, msg := range req.Data {
		if err := validateSyncCommitteeMessage(msg); err != nil {
			msgFailures = append(msgFailures, &helpers.SingleIndexedVerificationFailure{
				Index:   i,
				Message: err.Error(),
			})
			continue
		}

		v1alpha1Msg := &ethpbalpha.SyncCommitteeMessage{
			Slot:           msg.Slot,
			BlockRoot:      msg.BeaconBlockRoot,
			ValidatorIndex: msg.ValidatorIndex,
			Signature:      msg.Signature,
		}
		validMessages = append(validMessages, v1alpha1Msg)
	}

	for _, msg := range validMessages {
		_, err := bs.V1Alpha1ValidatorServer.SubmitSyncMessage(ctx, msg)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not submit message: %v", err)
		}
	}

	if len(msgFailures) > 0 {
		failuresContainer := &helpers.IndexedVerificationFailure{Failures: msgFailures}
		err := grpc.AppendCustomErrorHeader(ctx, failuresContainer)
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"One or more messages failed validation. Could not prepare detailed failure information: %v",
				err,
			)
		}
		return nil, status.Errorf(codes.InvalidArgument, "One or more messages failed validation")
	}

	return &empty.Empty{}, nil
}

func validateSyncCommitteeMessage(msg *ethpbv2.SyncCommitteeMessage) error {
	if !bytes.IsHexOfLen(msg.BeaconBlockRoot, 64) {
		return errors.New("invalid block root format")
	}
	if !bytes.IsHexOfLen(msg.Signature, 192) {
		return errors.New("invalid signature format")
	}
	return nil
}
