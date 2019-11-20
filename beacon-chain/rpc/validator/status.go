package validator

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errPubkeyDoesNotExist = errors.New("pubkey does not exist")

// ValidatorStatus returns the validator status of the current epoch.
// The status response can be one of the following:
//	PENDING_ACTIVE - validator is waiting to get activated.
//	ACTIVE - validator is active.
//	INITIATED_EXIT - validator has initiated an an exit request.
//	WITHDRAWABLE - validator's deposit can be withdrawn after lock up period.
//	EXITED - validator has exited, means the deposit has been withdrawn.
//	EXITED_SLASHED - validator was forcefully exited due to slashing.
func (vs *Server) ValidatorStatus(
	ctx context.Context,
	req *pb.ValidatorIndexRequest) (*pb.ValidatorStatusResponse, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	return vs.validatorStatus(ctx, req.PublicKey, headState), nil
}

// multipleValidatorStatus returns the validator status response for the set of validators
// requested by their pub keys.
func (vs *Server) multipleValidatorStatus(
	ctx context.Context,
	pubkeys [][]byte,
) (bool, []*pb.ValidatorActivationResponse_Status, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return false, nil, err
	}
	activeValidatorExists := false
	statusResponses := make([]*pb.ValidatorActivationResponse_Status, len(pubkeys))
	for i, key := range pubkeys {
		if ctx.Err() != nil {
			return false, nil, ctx.Err()
		}
		status := vs.validatorStatus(ctx, key, headState)
		if status == nil {
			continue
		}
		resp := &pb.ValidatorActivationResponse_Status{
			Status:    status,
			PublicKey: key,
		}
		statusResponses[i] = resp
		if status.Status == pb.ValidatorStatus_ACTIVE {
			activeValidatorExists = true
		}
	}

	return activeValidatorExists, statusResponses, nil
}

func (vs *Server) validatorStatus(ctx context.Context, pubKey []byte, headState *pbp2p.BeaconState) *pb.ValidatorStatusResponse {
	ctx, span := trace.StartSpan(ctx, "validatorServer.validatorStatus")
	defer span.End()

	resp := &pb.ValidatorStatusResponse{
		Status:          pb.ValidatorStatus_UNKNOWN_STATUS,
		ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
	}
	vStatus, idx, err := vs.retrieveStatusFromState(ctx, pubKey, headState)
	if err != nil && err != errPubkeyDoesNotExist {
		traceutil.AnnotateError(span, err)
		return resp
	}
	resp.Status = vStatus
	if err != errPubkeyDoesNotExist {
		resp.ActivationEpoch = headState.Validators[idx].ActivationEpoch
	}

	// If no connection to ETH1, the deposit block number or position in queue cannot be determined.
	if !vs.Eth1InfoFetcher.IsConnectedToETH1() {
		log.Warn("Not connected to ETH1. Cannot determine validator ETH1 deposit block number")
		return resp
	}

	_, eth1BlockNumBigInt := vs.DepositFetcher.DepositByPubkey(ctx, pubKey)
	if eth1BlockNumBigInt == nil { // No deposit found in ETH1.
		return resp
	}

	if resp.Status == pb.ValidatorStatus_UNKNOWN_STATUS {
		resp.Status = pb.ValidatorStatus_DEPOSIT_RECEIVED
	}

	resp.Eth1DepositBlockNumber = eth1BlockNumBigInt.Uint64()

	depositBlockSlot, err := vs.depositBlockSlot(ctx, headState.Slot, eth1BlockNumBigInt, headState)
	if err != nil {
		return resp
	}
	resp.DepositInclusionSlot = depositBlockSlot

	// If validator has been activated at any point, they are not in the queue so we can return
	// the request early. Additionally, if idx is zero (default return value) then we know this
	// validator cannot be in the queue either.
	if resp.ActivationEpoch != params.BeaconConfig().FarFutureEpoch || idx == 0 {
		return resp
	}

	var lastActivatedValidatorIdx uint64
	for j := len(headState.Validators) - 1; j >= 0; j-- {
		if helpers.IsActiveValidator(headState.Validators[j], helpers.CurrentEpoch(headState)) {
			lastActivatedValidatorIdx = uint64(j)
			break
		}
	}
	// Our position in the activation queue is the above index - our validator index.
	if lastActivatedValidatorIdx > idx {
		resp.PositionInActivationQueue = idx - lastActivatedValidatorIdx
	}

	return resp
}

func (vs *Server) retrieveStatusFromState(ctx context.Context, pubKey []byte,
	headState *pbp2p.BeaconState) (pb.ValidatorStatus, uint64, error) {
	if headState == nil {
		return pb.ValidatorStatus(0), 0, errors.New("head state does not exist")
	}
	idx, ok, err := vs.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
	if err != nil {
		return pb.ValidatorStatus(0), 0, err
	}
	if !ok || int(idx) >= len(headState.Validators) {
		return pb.ValidatorStatus(0), 0, errPubkeyDoesNotExist
	}
	return vs.assignmentStatus(idx, headState), idx, nil
}

func (vs *Server) assignmentStatus(validatorIdx uint64, beaconState *pbp2p.BeaconState) pb.ValidatorStatus {
	var status pb.ValidatorStatus
	v := beaconState.Validators[validatorIdx]
	epoch := helpers.CurrentEpoch(beaconState)
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	if epoch < v.ActivationEpoch {
		status = pb.ValidatorStatus_PENDING_ACTIVE
	} else if v.ExitEpoch == farFutureEpoch {
		status = pb.ValidatorStatus_ACTIVE
	} else if epoch >= v.WithdrawableEpoch {
		status = pb.ValidatorStatus_WITHDRAWABLE
	} else if v.Slashed && epoch >= v.ExitEpoch {
		status = pb.ValidatorStatus_EXITED_SLASHED
	} else if epoch >= v.ExitEpoch {
		status = pb.ValidatorStatus_EXITED
	} else if v.ExitEpoch != farFutureEpoch {
		status = pb.ValidatorStatus_INITIATED_EXIT
	} else {
		status = pb.ValidatorStatus_UNKNOWN_STATUS
	}

	return status
}

func (vs *Server) depositBlockSlot(ctx context.Context, currentSlot uint64,
	eth1BlockNumBigInt *big.Int, beaconState *pbp2p.BeaconState) (uint64, error) {
	blockTimeStamp, err := vs.BlockFetcher.BlockTimeByHeight(ctx, eth1BlockNumBigInt)
	if err != nil {
		return 0, err
	}
	followTime := time.Duration(params.BeaconConfig().Eth1FollowDistance*params.BeaconConfig().GoerliBlockTime) * time.Second
	eth1UnixTime := time.Unix(int64(blockTimeStamp), 0).Add(followTime)

	votingPeriodSlots := helpers.StartSlot(params.BeaconConfig().SlotsPerEth1VotingPeriod / params.BeaconConfig().SlotsPerEpoch)
	votingPeriodSeconds := time.Duration(votingPeriodSlots*params.BeaconConfig().SecondsPerSlot) * time.Second
	timeToInclusion := eth1UnixTime.Add(votingPeriodSeconds)

	eth2Genesis := time.Unix(int64(beaconState.GenesisTime), 0)
	eth2TimeDifference := timeToInclusion.Sub(eth2Genesis).Seconds()
	depositBlockSlot := uint64(eth2TimeDifference) / params.BeaconConfig().SecondsPerSlot

	return depositBlockSlot, nil
}
