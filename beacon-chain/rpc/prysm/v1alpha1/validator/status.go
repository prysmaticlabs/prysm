package validator

import (
	"context"
	"errors"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/contracts/deposit"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errPubkeyDoesNotExist = errors.New("pubkey does not exist")
var nonExistentIndex = types.ValidatorIndex(^uint64(0))

const numStatesToCheck = 2

// ValidatorStatus returns the validator status of the current epoch.
// The status response can be one of the following:
//  DEPOSITED - validator's deposit has been recognized by Ethereum 1, not yet recognized by Ethereum.
//  PENDING - validator is in Ethereum's activation queue.
//  ACTIVE - validator is active.
//  EXITING - validator has initiated an an exit request, or has dropped below the ejection balance and is being kicked out.
//  EXITED - validator is no longer validating.
//  SLASHING - validator has been kicked out due to meeting a slashing condition.
//  UNKNOWN_STATUS - validator does not have a known status in the network.
func (vs *Server) ValidatorStatus(
	ctx context.Context,
	req *ethpb.ValidatorStatusRequest,
) (*ethpb.ValidatorStatusResponse, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	vStatus, _ := vs.validatorStatus(ctx, headState, req.PublicKey)
	return vStatus, nil
}

// MultipleValidatorStatus is the same as ValidatorStatus. Supports retrieval of multiple
// validator statuses. Takes a list of public keys or a list of validator indices.
func (vs *Server) MultipleValidatorStatus(
	ctx context.Context,
	req *ethpb.MultipleValidatorStatusRequest,
) (*ethpb.MultipleValidatorStatusResponse, error) {
	if vs.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	responseCap := len(req.PublicKeys) + len(req.Indices)
	pubKeys := make([][]byte, 0, responseCap)
	filtered := make(map[[48]byte]bool)
	filtered[[48]byte{}] = true // Filter out keys with all zeros.
	// Filter out duplicate public keys.
	for _, pubKey := range req.PublicKeys {
		pubkeyBytes := bytesutil.ToBytes48(pubKey)
		if !filtered[pubkeyBytes] {
			pubKeys = append(pubKeys, pubKey)
			filtered[pubkeyBytes] = true
		}
	}
	// Convert indices to public keys.
	for _, idx := range req.Indices {
		pubkeyBytes := headState.PubkeyAtIndex(types.ValidatorIndex(idx))
		if !filtered[pubkeyBytes] {
			pubKeys = append(pubKeys, pubkeyBytes[:])
			filtered[pubkeyBytes] = true
		}
	}
	// Fetch statuses from beacon state.
	statuses := make([]*ethpb.ValidatorStatusResponse, len(pubKeys))
	indices := make([]types.ValidatorIndex, len(pubKeys))
	for i, pubKey := range pubKeys {
		statuses[i], indices[i] = vs.validatorStatus(ctx, headState, pubKey)
	}

	return &ethpb.MultipleValidatorStatusResponse{
		PublicKeys: pubKeys,
		Statuses:   statuses,
		Indices:    indices,
	}, nil
}

// CheckDoppelGanger checks if the provided keys are currently active in the network.
func (vs *Server) CheckDoppelGanger(ctx context.Context, req *ethpb.DoppelGangerRequest) (*ethpb.DoppelGangerResponse, error) {
	if vs.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}
	if req == nil || req.ValidatorRequests == nil || len(req.ValidatorRequests) == 0 {
		return &ethpb.DoppelGangerResponse{
			Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{},
		}, nil
	}
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}

	currEpoch := slots.ToEpoch(headState.Slot())
	isRecent, resp := checkValidatorsAreRecent(currEpoch, req)
	// If all provided keys are recent we skip this check
	// as we are unable to effectively determine if a doppelganger
	// is active.
	if isRecent {
		return resp, nil
	}
	// We walk back from the current head state to the state at the beginning of the previous 2 epochs.
	// Where S_i , i := 0,1,2. i = 0 would signify the current head state in this epoch.
	previousEpoch, err := currEpoch.SafeSub(1)
	if err != nil {
		previousEpoch = currEpoch
	}
	olderEpoch, err := previousEpoch.SafeSub(1)
	if err != nil {
		olderEpoch = previousEpoch
	}
	prevState, err := vs.retrieveAfterEpochTransition(ctx, previousEpoch)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get previous state")
	}
	olderState, err := vs.retrieveAfterEpochTransition(ctx, olderEpoch)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get older state")
	}
	resp = &ethpb.DoppelGangerResponse{
		Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{},
	}
	for _, v := range req.ValidatorRequests {
		// If the validator's last recorded epoch was
		// less than or equal to `numStatesToCheck` epochs ago, this method will not
		// be able to catch duplicates. This is due to how attestation
		// inclusion works, where an attestation for the current epoch
		// is able to included in the current or next epoch. Depending
		// on which epoch it is included the balance change will be
		// reflected in the following epoch.
		if v.Epoch+numStatesToCheck >= currEpoch {
			resp.Responses = append(resp.Responses,
				&ethpb.DoppelGangerResponse_ValidatorResponse{
					PublicKey:       v.PublicKey,
					DuplicateExists: false,
				})
			continue
		}
		valIndex, ok := olderState.ValidatorIndexByPubkey(bytesutil.ToBytes48(v.PublicKey))
		if !ok {
			// Ignore if validator pubkey doesn't exist.
			continue
		}
		baseBal, err := olderState.BalanceAtIndex(valIndex)
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get validator's balance")
		}
		nextBal, err := prevState.BalanceAtIndex(valIndex)
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get validator's balance")
		}
		// If the next epoch's balance is higher, we mark it as an existing
		// duplicate.
		if nextBal > baseBal {
			log.Infof("current %d with last epoch %d and difference in bal %d gwei", currEpoch, v.Epoch, nextBal-baseBal)
			resp.Responses = append(resp.Responses,
				&ethpb.DoppelGangerResponse_ValidatorResponse{
					PublicKey:       v.PublicKey,
					DuplicateExists: true,
				})
			continue
		}
		currBal, err := headState.BalanceAtIndex(valIndex)
		if err != nil {
			return nil, status.Error(codes.Internal, "Could not get validator's balance")
		}
		// If the current epoch's balance is higher, we mark it as an existing
		// duplicate.
		if currBal > nextBal {
			resp.Responses = append(resp.Responses,
				&ethpb.DoppelGangerResponse_ValidatorResponse{
					PublicKey:       v.PublicKey,
					DuplicateExists: true,
				})
			continue
		}
		// Mark the public key as valid.
		resp.Responses = append(resp.Responses,
			&ethpb.DoppelGangerResponse_ValidatorResponse{
				PublicKey:       v.PublicKey,
				DuplicateExists: false,
			})
	}
	return resp, nil
}

// activationStatus returns the validator status response for the set of validators
// requested by their pub keys.
func (vs *Server) activationStatus(
	ctx context.Context,
	pubKeys [][]byte,
) (bool, []*ethpb.ValidatorActivationResponse_Status, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return false, nil, err
	}
	activeValidatorExists := false
	statusResponses := make([]*ethpb.ValidatorActivationResponse_Status, len(pubKeys))
	for i, pubKey := range pubKeys {
		if ctx.Err() != nil {
			return false, nil, ctx.Err()
		}
		vStatus, idx := vs.validatorStatus(ctx, headState, pubKey)
		if vStatus == nil {
			continue
		}
		resp := &ethpb.ValidatorActivationResponse_Status{
			Status:    vStatus,
			PublicKey: pubKey,
			Index:     idx,
		}
		statusResponses[i] = resp
		if vStatus.Status == ethpb.ValidatorStatus_ACTIVE {
			activeValidatorExists = true
		}
	}

	return activeValidatorExists, statusResponses, nil
}

// validatorStatus searches for the requested validator's state and deposit to retrieve its inclusion estimate. Also returns the validators index.
func (vs *Server) validatorStatus(
	ctx context.Context,
	headState state.ReadOnlyBeaconState,
	pubKey []byte,
) (*ethpb.ValidatorStatusResponse, types.ValidatorIndex) {
	ctx, span := trace.StartSpan(ctx, "ValidatorServer.validatorStatus")
	defer span.End()

	// Using ^0 as the default value for index, in case the validators index cannot be determined.
	resp := &ethpb.ValidatorStatusResponse{
		Status:          ethpb.ValidatorStatus_UNKNOWN_STATUS,
		ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
	}
	vStatus, idx, err := statusForPubKey(headState, pubKey)
	if err != nil && err != errPubkeyDoesNotExist {
		tracing.AnnotateError(span, err)
		return resp, nonExistentIndex
	}
	resp.Status = vStatus
	if err != errPubkeyDoesNotExist {
		val, err := headState.ValidatorAtIndexReadOnly(idx)
		if err != nil {
			tracing.AnnotateError(span, err)
			return resp, idx
		}
		resp.ActivationEpoch = val.ActivationEpoch()
	}

	switch resp.Status {
	// Unknown status means the validator has not been put into the state yet.
	case ethpb.ValidatorStatus_UNKNOWN_STATUS:
		// If no connection to ETH1, the deposit block number or position in queue cannot be determined.
		if !vs.Eth1InfoFetcher.IsConnectedToETH1() {
			log.Warn("Not connected to ETH1. Cannot determine validator ETH1 deposit block number")
			return resp, nonExistentIndex
		}
		dep, eth1BlockNumBigInt := vs.DepositFetcher.DepositByPubkey(ctx, pubKey)
		if eth1BlockNumBigInt == nil { // No deposit found in ETH1.
			return resp, nonExistentIndex
		}
		domain, err := signing.ComputeDomain(
			params.BeaconConfig().DomainDeposit,
			nil, /*forkVersion*/
			nil, /*genesisValidatorsRoot*/
		)
		if err != nil {
			log.Warn("Could not compute domain")
			return resp, nonExistentIndex
		}
		if err := deposit.VerifyDepositSignature(dep.Data, domain); err != nil {
			resp.Status = ethpb.ValidatorStatus_INVALID
			log.Warn("Invalid Eth1 deposit")
			return resp, nonExistentIndex
		}
		// Set validator deposit status if their deposit is visible.
		resp.Status = depositStatus(dep.Data.Amount)
		resp.Eth1DepositBlockNumber = eth1BlockNumBigInt.Uint64()

		return resp, nonExistentIndex
	// Deposited, Pending or Partially Deposited mean the validator has been put into the state.
	case ethpb.ValidatorStatus_DEPOSITED, ethpb.ValidatorStatus_PENDING, ethpb.ValidatorStatus_PARTIALLY_DEPOSITED:
		if resp.Status == ethpb.ValidatorStatus_PENDING {
			if vs.DepositFetcher == nil {
				log.Warn("Not connected to ETH1. Cannot determine validator ETH1 deposit.")
			} else {
				// Check if there was a deposit deposit.
				deposit, eth1BlockNumBigInt := vs.DepositFetcher.DepositByPubkey(ctx, pubKey)
				if eth1BlockNumBigInt != nil {
					resp.Status = depositStatus(deposit.Data.Amount)
					resp.Eth1DepositBlockNumber = eth1BlockNumBigInt.Uint64()
				}
			}
		}

		var lastActivatedvalidatorIndex types.ValidatorIndex
		for j := headState.NumValidators() - 1; j >= 0; j-- {
			val, err := headState.ValidatorAtIndexReadOnly(types.ValidatorIndex(j))
			if err != nil {
				return resp, idx
			}
			if helpers.IsActiveValidatorUsingTrie(val, time.CurrentEpoch(headState)) {
				lastActivatedvalidatorIndex = types.ValidatorIndex(j)
				break
			}
		}
		// Our position in the activation queue is the above index - our validator index.
		if lastActivatedvalidatorIndex < idx {
			resp.PositionInActivationQueue = uint64(idx - lastActivatedvalidatorIndex)
		}
		return resp, idx
	default:
		return resp, idx
	}
}

func (vs *Server) retrieveAfterEpochTransition(ctx context.Context, epoch types.Epoch) (state.BeaconState, error) {
	endSlot, err := slots.EpochEnd(epoch)
	if err != nil {
		return nil, err
	}
	retState, err := vs.StateGen.StateBySlot(ctx, endSlot)
	if err != nil {
		return nil, err
	}
	return transition.ProcessSlots(ctx, retState, retState.Slot()+1)
}

func checkValidatorsAreRecent(headEpoch types.Epoch, req *ethpb.DoppelGangerRequest) (bool, *ethpb.DoppelGangerResponse) {
	validatorsAreRecent := true
	resp := &ethpb.DoppelGangerResponse{
		Responses: []*ethpb.DoppelGangerResponse_ValidatorResponse{},
	}
	for _, v := range req.ValidatorRequests {
		// Due to how balances are reflected for individual
		// validators, we can only effectively determine if a
		// validator voted or not if we are able to look
		// back more than `numStatesToCheck` epochs into the past.
		if v.Epoch+numStatesToCheck < headEpoch {
			validatorsAreRecent = false
			// Zero out response if we encounter non-recent validators to
			// guard against potential misuse.
			resp.Responses = []*ethpb.DoppelGangerResponse_ValidatorResponse{}
			break
		}
		resp.Responses = append(resp.Responses,
			&ethpb.DoppelGangerResponse_ValidatorResponse{
				PublicKey:       v.PublicKey,
				DuplicateExists: false,
			})
	}
	return validatorsAreRecent, resp
}

func statusForPubKey(headState state.ReadOnlyBeaconState, pubKey []byte) (ethpb.ValidatorStatus, types.ValidatorIndex, error) {
	if headState == nil || headState.IsNil() {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS, 0, errors.New("head state does not exist")
	}
	idx, ok := headState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
	if !ok || uint64(idx) >= uint64(headState.NumValidators()) {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS, 0, errPubkeyDoesNotExist
	}
	return assignmentStatus(headState, idx), idx, nil
}

func assignmentStatus(beaconState state.ReadOnlyBeaconState, validatorIndex types.ValidatorIndex) ethpb.ValidatorStatus {
	validator, err := beaconState.ValidatorAtIndexReadOnly(validatorIndex)
	if err != nil {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS
	}
	currentEpoch := time.CurrentEpoch(beaconState)
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch
	validatorBalance := validator.EffectiveBalance()

	if validator.IsNil() {
		return ethpb.ValidatorStatus_UNKNOWN_STATUS
	}
	if currentEpoch < validator.ActivationEligibilityEpoch() {
		return depositStatus(validatorBalance)
	}
	if currentEpoch < validator.ActivationEpoch() {
		return ethpb.ValidatorStatus_PENDING
	}
	if validator.ExitEpoch() == farFutureEpoch {
		return ethpb.ValidatorStatus_ACTIVE
	}
	if currentEpoch < validator.ExitEpoch() {
		if validator.Slashed() {
			return ethpb.ValidatorStatus_SLASHING
		}
		return ethpb.ValidatorStatus_EXITING
	}
	return ethpb.ValidatorStatus_EXITED
}

func depositStatus(depositOrBalance uint64) ethpb.ValidatorStatus {
	if depositOrBalance == 0 {
		return ethpb.ValidatorStatus_PENDING
	} else if depositOrBalance < params.BeaconConfig().MaxEffectiveBalance {
		return ethpb.ValidatorStatus_PARTIALLY_DEPOSITED
	}
	return ethpb.ValidatorStatus_DEPOSITED
}
