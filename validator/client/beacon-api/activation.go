package beacon_api

import (
	"strconv"
	"time"

	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/grpc"
)

func (c beaconApiValidatorClient) waitForActivation(ctx context.Context, in *ethpb.ValidatorActivationRequest) (ethpb.BeaconNodeValidator_WaitForActivationClient, error) {
	return &waitForActivationClient{
		ctx:                        ctx,
		beaconApiValidatorClient:   c,
		ValidatorActivationRequest: in,
	}, nil
}

type waitForActivationClient struct {
	grpc.ClientStream
	ctx context.Context
	beaconApiValidatorClient
	*ethpb.ValidatorActivationRequest
	lastRecvTime time.Time
}

func computeWaitElements(now time.Time, lastRecvTime time.Time) (time.Duration, time.Time) {
	nextRecvTime := lastRecvTime.Add(time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second)

	if lastRecvTime.IsZero() {
		nextRecvTime = now
	}

	if nextRecvTime.Before(now) {
		return time.Duration(0), now
	}

	return nextRecvTime.Sub(now), nextRecvTime
}

func (c *waitForActivationClient) Recv() (*ethpb.ValidatorActivationResponse, error) {
	waitDuration, nextRecvTime := computeWaitElements(time.Now(), c.lastRecvTime)

	select {
	case <-time.After(waitDuration):
		c.lastRecvTime = nextRecvTime

		// Represents the target set of keys
		stringTargetPubKeysToPubKeys := make(map[string][]byte, len(c.ValidatorActivationRequest.PublicKeys))
		stringTargetPubKeys := make([]string, len(c.ValidatorActivationRequest.PublicKeys))

		// Represents the set of keys actually returned by the beacon node
		stringRetrievedPubKeys := make(map[string]struct{})

		// Contains all keys in targetPubKeys but not in retrievedPubKeys
		missingPubKeys := [][]byte{}

		statuses := []*ethpb.ValidatorActivationResponse_Status{}

		for index, publicKey := range c.ValidatorActivationRequest.PublicKeys {
			stringPubKey := hexutil.Encode(publicKey)
			stringTargetPubKeysToPubKeys[stringPubKey] = publicKey
			stringTargetPubKeys[index] = stringPubKey
		}

		stateValidators, err := c.beaconApiValidatorClient.getStateValidators(stringTargetPubKeys, nil)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get state validators")
		}

		for _, data := range stateValidators.Data {
			pubkey, err := hexutil.Decode(data.Validator.PublicKey)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse validator public key")
			}

			stringRetrievedPubKeys[data.Validator.PublicKey] = struct{}{}

			index, err := strconv.ParseUint(data.Index, 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse validator index")
			}

			validatorStatus, ok := beaconAPITogRPCValidatorStatus[data.Status]
			if !ok {
				return nil, errors.New("invalid validator status: " + data.Status)
			}

			statuses = append(statuses, &ethpb.ValidatorActivationResponse_Status{
				PublicKey: pubkey,
				Index:     types.ValidatorIndex(index),
				Status:    &ethpb.ValidatorStatusResponse{Status: validatorStatus},
			})
		}

		for stringTargetPubKey, targetPubKey := range stringTargetPubKeysToPubKeys {
			if _, ok := stringRetrievedPubKeys[stringTargetPubKey]; !ok {
				missingPubKeys = append(missingPubKeys, targetPubKey)
			}
		}

		for _, missingPubKey := range missingPubKeys {
			statuses = append(statuses, &ethpb.ValidatorActivationResponse_Status{
				PublicKey: missingPubKey,
				Index:     types.ValidatorIndex(^uint64(0)),
				Status:    &ethpb.ValidatorStatusResponse{Status: ethpb.ValidatorStatus_UNKNOWN_STATUS},
			})
		}

		return &ethpb.ValidatorActivationResponse{
			Statuses: statuses,
		}, nil
	case <-c.ctx.Done():
		return nil, errors.New("context canceled")
	}
}
