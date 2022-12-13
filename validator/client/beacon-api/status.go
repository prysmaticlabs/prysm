package beacon_api

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (c *beaconApiValidatorClient) validatorStatus(in *ethpb.ValidatorStatusRequest) (*ethpb.ValidatorStatusResponse, error) {
	stringPubKey := hexutil.Encode(in.PublicKey)

	resp := &ethpb.ValidatorStatusResponse{
		Status:          ethpb.ValidatorStatus_UNKNOWN_STATUS,
		ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
	}

	stateValidator, err := c.getStateValidators([]string{stringPubKey}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state validator")
	}

	// If no data, the validator is in unknown status
	if len(stateValidator.Data) == 0 {
		return resp, nil
	}

	validatorContainer := stateValidator.Data[0]

	// Set Status
	status, ok := beaconAPITogRPCValidatorStatus[validatorContainer.Status]
	if !ok {
		return nil, errors.New("invalid validator status: " + validatorContainer.Status)
	}

	resp.Status = status

	// Set activation epoch
	activationEpoch, err := strconv.ParseInt(validatorContainer.Validator.ActivationEpoch, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse activation epoch")
	}

	resp.ActivationEpoch = types.Epoch(activationEpoch)

	// Set PositionInActivationQueue
	switch status {
	case ethpb.ValidatorStatus_DEPOSITED, ethpb.ValidatorStatus_PENDING, ethpb.ValidatorStatus_PARTIALLY_DEPOSITED:
		validatorIndex, err := strconv.ParseUint(validatorContainer.Index, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse validator index")
		}

		activeStateValidators, err := c.getStateValidators(nil, []string{"active"})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get state validators")
		}

		data := activeStateValidators.Data

		var lastActivatedValidatorIndex uint64 = 0

		if nbActiveValidators := len(data); nbActiveValidators != 0 {
			lastValidator := data[nbActiveValidators-1]

			lastActivatedValidatorIndex, err = strconv.ParseUint(lastValidator.Index, 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse last validator index")
			}
		}

		resp.PositionInActivationQueue = validatorIndex - lastActivatedValidatorIndex
	}

	return resp, nil
}
