package endtoend

import (
	"context"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Formally defining them might not be needed.
// type policy func(chainHead *eth.ChainHead, options ...uint64) error
// type evaluation func(client *eth.BeaconChainClient, options ...uint64) error

// // Evaluator defines the function signature for function to run during the E2E.
// type Evaluator struct {
// 	Policy     policy
// 	Evaluation evaluation
// }

// AfterNEpochs run the evaluator after N epochs.
func AfterNEpochs(chainHead *eth.ChainHead, epochs uint64) bool {
	currentEpoch := chainHead.BlockSlot / params.BeaconConfig().SlotsPerEpoch
	return currentEpoch == epochs
}

// AfterChainStart ensures the chain has started before performing the evaluator.
func AfterChainStart(chainHead *eth.ChainHead) bool {
	return chainHead.BlockSlot >= 0 && chainHead.BlockSlot < params.BeaconConfig().SlotsPerEpoch
}

// ValidatorsActivate ensures the expected amount of validators
// are active.
func ValidatorsActivate(client eth.BeaconChainClient, expectedCount uint64) error {
	validatorRequest := &eth.GetValidatorsRequest{}
	validators, err := client.GetValidators(context.Background(), validatorRequest)
	if err != nil {
		return fmt.Errorf("failed to get validators: %v", err)
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
func ValidatorsParticipating(client eth.BeaconChainClient, epoch uint64) error {
	in := new(ptypes.Empty)
	chainHead, err := client.GetChainHead(context.Background(), in)
	if err != nil {
		return fmt.Errorf("failed to get chain head: %v", err)
	}
	currentEpoch := chainHead.BlockSlot / params.BeaconConfig().SlotsPerEpoch
	if epoch > currentEpoch {
		return fmt.Errorf("requested epoch hasn't passed yet, received: %d, current: %d", epoch, currentEpoch)
	}

	validatorRequest := &eth.GetValidatorParticipationRequest{
		QueryFilter: &eth.GetValidatorParticipationRequest_Epoch{
			Epoch: epoch,
		},
	}
	participation, err := client.GetValidatorParticipation(context.Background(), validatorRequest)
	if err != nil {
		return fmt.Errorf("failed to get validator participation: %v", err)
	}

	partRate := participation.Participation.GlobalParticipationRate
	if partRate < 0.85 {
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
		return fmt.Errorf("failed to get chain head: %v", err)
	}

	currentEpoch := chainHead.BlockSlot / params.BeaconConfig().SlotsPerEpoch
	if currentEpoch < 4 {
		return fmt.Errorf("current epoch is less than 4, received: %d", currentEpoch)
	}
	finalizedEpoch := chainHead.FinalizedSlot / params.BeaconConfig().SlotsPerEpoch
	if finalizedEpoch < 2 {
		return fmt.Errorf("expected finalized epoch to be greater than 2, received: %d", currentEpoch)
	}
	previousJustifiedEpoch := chainHead.PreviousJustifiedSlot / params.BeaconConfig().SlotsPerEpoch
	currentJustifiedEpoch := chainHead.JustifiedSlot / params.BeaconConfig().SlotsPerEpoch
	if previousJustifiedEpoch+1 != currentJustifiedEpoch {
		return fmt.Errorf(
			"there should be no gaps between current and previous justified epochs, received current %d and previous %d",
			currentJustifiedEpoch,
			previousJustifiedEpoch,
		)
	}

	return nil
}
