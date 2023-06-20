package beacon_api

import (
	"context"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

func (c *beaconApiValidatorClient) validatorStatus(ctx context.Context, in *ethpb.ValidatorStatusRequest) (*ethpb.ValidatorStatusResponse, error) {
	_, _, validatorsStatusResponse, err := c.getValidatorsStatusResponse(ctx, [][]byte{in.PublicKey}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validator status response")
	}

	if len(validatorsStatusResponse) != 1 {
		return nil, errors.New("number of validator status response not expected")
	}

	validatorStatusResponse := validatorsStatusResponse[0]

	return validatorStatusResponse, nil
}

func (c *beaconApiValidatorClient) multipleValidatorStatus(ctx context.Context, in *ethpb.MultipleValidatorStatusRequest) (*ethpb.MultipleValidatorStatusResponse, error) {
	publicKeys, indices, statuses, err := c.getValidatorsStatusResponse(ctx, in.PublicKeys, in.Indices)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validators status response")
	}

	return &ethpb.MultipleValidatorStatusResponse{
		PublicKeys: publicKeys,
		Indices:    indices,
		Statuses:   statuses,
	}, nil
}

func (c *beaconApiValidatorClient) getValidatorsStatusResponse(ctx context.Context, inPubKeys [][]byte, inIndexes []int64) (
	[][]byte,
	[]primitives.ValidatorIndex,
	[]*ethpb.ValidatorStatusResponse,
	error,
) {
	// Represents the target set of keys
	stringTargetPubKeysToPubKeys := make(map[string][]byte, len(inPubKeys))
	stringTargetPubKeys := make([]string, len(inPubKeys))

	// Represents the set of keys actually returned by the beacon node
	stringRetrievedPubKeys := make(map[string]struct{})

	// Contains all keys in targetPubKeys but not in retrievedPubKeys
	missingPubKeys := [][]byte{}

	totalLen := len(inPubKeys) + len(inIndexes)

	outPubKeys := make([][]byte, totalLen)
	outIndexes := make([]primitives.ValidatorIndex, totalLen)
	outValidatorsStatuses := make([]*ethpb.ValidatorStatusResponse, totalLen)

	for index, publicKey := range inPubKeys {
		stringPubKey := hexutil.Encode(publicKey)
		stringTargetPubKeysToPubKeys[stringPubKey] = publicKey
		stringTargetPubKeys[index] = stringPubKey
	}

	// Get state for the current validator
	stateValidatorsResponse, err := c.stateValidatorsProvider.GetStateValidators(ctx, stringTargetPubKeys, inIndexes, nil)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to get state validators")
	}

	isLastActivatedValidatorIndexRetrieved := false
	var lastActivatedValidatorIndex uint64 = 0

	for i, validatorContainer := range stateValidatorsResponse.Data {
		stringPubKey := validatorContainer.Validator.PublicKey

		stringRetrievedPubKeys[stringPubKey] = struct{}{}

		pubKey, ok := stringTargetPubKeysToPubKeys[stringPubKey]
		if !ok {
			// string pub key is not already known because the index was used for this validator
			pubKey, err = hexutil.Decode(stringPubKey)
			if err != nil {
				return nil, nil, nil, errors.Wrapf(err, "failed to parse validator public key %s", stringPubKey)
			}
		}

		validatorIndex, err := strconv.ParseUint(validatorContainer.Index, 10, 64)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to parse validator index %s", validatorContainer.Index)
		}

		outPubKeys[i] = pubKey
		outIndexes[i] = primitives.ValidatorIndex(validatorIndex)

		validatorStatus := &ethpb.ValidatorStatusResponse{}

		// Set Status
		status, ok := beaconAPITogRPCValidatorStatus[validatorContainer.Status]
		if !ok {
			return nil, nil, nil, errors.New("invalid validator status " + validatorContainer.Status)
		}

		validatorStatus.Status = status

		// Set activation epoch
		activationEpoch, err := strconv.ParseUint(validatorContainer.Validator.ActivationEpoch, 10, 64)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed to parse activation epoch %s", validatorContainer.Validator.ActivationEpoch)
		}

		validatorStatus.ActivationEpoch = primitives.Epoch(activationEpoch)

		// Set PositionInActivationQueue
		switch status {
		case ethpb.ValidatorStatus_PENDING, ethpb.ValidatorStatus_PARTIALLY_DEPOSITED, ethpb.ValidatorStatus_DEPOSITED:
			if !isLastActivatedValidatorIndexRetrieved {
				isLastActivatedValidatorIndexRetrieved = true
				// TODO: double check this due to potential of PENDING STATE being active..
				// edge case https://github.com/prysmaticlabs/prysm/blob/0669050ffabe925c3d6e5e5d535a86361ae8522b/validator/client/validator.go#L1068
				activeStateValidators, err := c.stateValidatorsProvider.GetStateValidators(ctx, nil, nil, []string{"active"})
				if err != nil {
					return nil, nil, nil, errors.Wrap(err, "failed to get state validators")
				}

				data := activeStateValidators.Data

				if nbActiveValidators := len(data); nbActiveValidators != 0 {
					lastValidator := data[nbActiveValidators-1]

					lastActivatedValidatorIndex, err = strconv.ParseUint(lastValidator.Index, 10, 64)
					if err != nil {
						return nil, nil, nil, errors.Wrapf(err, "failed to parse last validator index %s", lastValidator.Index)
					}
				}
			}

			validatorStatus.PositionInActivationQueue = validatorIndex - lastActivatedValidatorIndex
		}

		outValidatorsStatuses[i] = validatorStatus
	}

	for _, stringTargetPubKey := range stringTargetPubKeys {
		if _, ok := stringRetrievedPubKeys[stringTargetPubKey]; !ok {
			targetPubKey := stringTargetPubKeysToPubKeys[stringTargetPubKey]
			missingPubKeys = append(missingPubKeys, targetPubKey)
		}
	}

	nbStringRetrievedPubKeys := len(stringRetrievedPubKeys)

	for i, missingPubKey := range missingPubKeys {
		outPubKeys[nbStringRetrievedPubKeys+i] = missingPubKey
		outIndexes[nbStringRetrievedPubKeys+i] = primitives.ValidatorIndex(^uint64(0))

		outValidatorsStatuses[nbStringRetrievedPubKeys+i] = &ethpb.ValidatorStatusResponse{
			Status:          ethpb.ValidatorStatus_UNKNOWN_STATUS,
			ActivationEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	outLen := len(stateValidatorsResponse.Data) + len(missingPubKeys)
	return outPubKeys[:outLen], outIndexes[:outLen], outValidatorsStatuses[:outLen], nil
}
