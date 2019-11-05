package main

import (
	"context"
	"errors"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type policy func(chainHead *eth.ChainHead, options ...uint64) error
type evaluation func(client *eth.BeaconChainClient, options ...uint64) error

// Evaluator defines the function signature for function to run during the E2E.
type Evaluator struct {
	Policy     policy
	Evaluation evaluation
}

// AfterNEpochs run the evaluator after N epochs.
func AfterNEpochs(chainHead *eth.ChainHead, epochs uint64) bool {
	return chainHead.BlockSlot/params.BeaconConfig().SlotsPerEpoch >= epochs
}

// AfterChainStart ensures the chain has started before performing the evaluator.
func AfterChainStart(chainHead *eth.ChainHead) bool {
	return chainHead.BlockSlot > 0
}

// ValidatorsActivate ensures the expected amount of validators
// are active.
func ValidatorsActivate(client eth.BeaconChainClient, expectedCount uint64) error {
	validatorRequest := &eth.GetValidatorsRequest{}
	validators, err := client.GetValidators(context.Background(), validatorRequest)
	if err != nil {
		return errors.New("failed to get validators")
	}

	receivedCount := uint64(len(validators.Validators))
	if expectedCount != receivedCount {
		return fmt.Errorf("expected validator count to be %d, recevied %d", expectedCount, receivedCount)
	}

	for _, val := range validators.Validators {
		if val.ActivationEpoch != 0 {
			return fmt.Errorf("genesis validator epoch should be 0, received %d", val.ActivationEpoch)
		}
		if val.WithdrawableEpoch != params.BeaconConfig().FarFutureEpoch {
			return fmt.Errorf("genesis validator withdrawable epoch should be far future, received %d", val.WithdrawableEpoch)
		}
	}
	return nil
}

// ValidatorsParticipating ensures the validators have an acceptable participation rate.
func ValidatorsParticipating(client eth.BeaconChainClient, expectedCount uint64) error {
	validatorRequest := &eth.GetValidatorParticipationRequest{
		QueryFilter: &eth.GetValidatorParticipationRequest_Epoch{
			Epoch: 2,
		},
	}
	participation, err := client.GetValidatorParticipation(context.Background(), validatorRequest)
	if err != nil {
		return errors.New("failed to get validator participation")
	}

	partRate := participation.Participation.GlobalParticipationRate
	if partRate < 0.95 {
		return fmt.Errorf("validator participation not as high as expected, received: %f", partRate)
	}
	return nil
}

// FinalizationOccurs is an evaluator to make sure finalization is performing as it should.
// Requires to be run after at least 4 epochs have passed.
func FinalizationOccurs(client eth.BeaconChainClient) error {
	in := new(ptypes.Empty)
	chainHead, err := client.GetChainHead(context.Background(), in)
	if err != nil {
		return errors.New("failed to get chain head")
	}

	currentEpoch := chainHead.BlockSlot / params.BeaconConfig().SlotsPerEpoch
	if currentEpoch < 4 {
		return fmt.Errorf("current epoch is less than 2, received: %d", currentEpoch)
	}
	finalizedEpoch := chainHead.FinalizedSlot / params.BeaconConfig().SlotsPerEpoch
	if finalizedEpoch < 2 {
		return fmt.Errorf("expected finalized epoch to be greater than 2, received: %d", currentEpoch)
	}
	return nil
}
