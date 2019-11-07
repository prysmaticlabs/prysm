package endtoend

import (
	"context"
	"fmt"
	"testing"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Policy defines an function that returns a bool
// for whether the evaluator it is assigned to runs.
type Policy func(currentEpoch uint64) bool

// Evaluation defines a function that takes in BeaconChainClient
// and uses it to monitor the chain, to determine where it fulfills the conditions required.
type Evaluation func(client eth.BeaconChainClient) error

// Evaluator defines a struct for executing a given evaluation
// on a running E2E test given that the policy returns true.
type Evaluator struct {
	Name       string
	Policy     Policy
	Evaluation Evaluation
}

// if AfterNEpochs(chainHead, 6) {
// 	fmt.Println("Running participation test")
// 	// Requesting last epoch here since I can't guarantee which slot this request is being made.
// 	t.Run("validators are participating", func(t *testing.T) {
// 		if err := ValidatorsParticipating(beaconClient, 5); err != nil {
// 			t.Fatal(err)
// 		}
// 	})

// func currentEpoch(client eth.BeaconChainClient) (uint64, error) {
// 	in := new(ptypes.Empty)
// 	chainHead, err := client.GetChainHead(context.Background(), in)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to get chain head: %v", err)
// 	}
// 	currentEpoch := chainHead.BlockSlot / params.BeaconConfig().SlotsPerEpoch
// 	return currentEpoch, nil
// }

// RunEvaluators takes in testing.T, BeaconChainClient and an array of evaluators.
// Using the client to check the state, it runs each evaluator in a subtest.
func RunEvaluators(t *testing.T, client eth.BeaconChainClient, evaluators []Evaluator) {
	in := new(ptypes.Empty)
	chainHead, err := client.GetChainHead(context.Background(), in)
	if err != nil {
		t.Errorf("failed to get chain head: %v", err)
	}
	currentEpoch := chainHead.BlockSlot / params.BeaconConfig().SlotsPerEpoch

	for _, evaluator := range evaluators {
		if evaluator.Policy(currentEpoch) {
			fmt.Printf("Running %s\n", evaluator.Name)
			t.Run(evaluator.Name, func(t *testing.T) {
				if err := evaluator.Evaluation(client); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

// AfterNEpochs run the evaluator after N epochs.
func AfterNEpochs(epochs uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return currentEpoch == epochs
	}
}

// TODO change this to make more sense
// OnChainStart ensures the chain has started before performing the evaluator.
func OnChainStart(currentEpoch uint64) bool {
	return currentEpoch == 0
}

// ValidatorsActivate ensures the expected amount of validators
// are active.
func ValidatorsActivate(client eth.BeaconChainClient) error {
	validatorRequest := &eth.GetValidatorsRequest{}
	validators, err := client.GetValidators(context.Background(), validatorRequest)
	if err != nil {
		return fmt.Errorf("failed to get validators: %v", err)
	}

	expectedCount := params.BeaconConfig().MinGenesisActiveValidatorCount
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
func ValidatorsParticipating(client eth.BeaconChainClient) error {
	in := new(ptypes.Empty)
	chainHead, err := client.GetChainHead(context.Background(), in)
	if err != nil {
		return fmt.Errorf("failed to get chain head: %v", err)
	}
	currentEpoch := chainHead.BlockSlot / params.BeaconConfig().SlotsPerEpoch
	if currentEpoch > 0 {
		return fmt.Errorf("current epoch must be greater than 0, received %d", currentEpoch)
	}

	validatorRequest := &eth.GetValidatorParticipationRequest{
		QueryFilter: &eth.GetValidatorParticipationRequest_Epoch{
			Epoch: currentEpoch - 1,
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
