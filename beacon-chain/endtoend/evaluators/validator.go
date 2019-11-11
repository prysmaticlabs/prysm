package evaluators

import (
	"context"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Evaluator defines the structure of the evaluators used to
// conduct the current beacon state during the E2E.
type Evaluator struct {
	Name       string
	Evaluation func(client eth.BeaconChainClient) error
}

// ValidatorsAreActive ensures the expected amount of validators are active.
var ValidatorsAreActive = Evaluator{
	Name:       "checkpoint_finalizes",
	Evaluation: finalizationOccurs,
}

func validatorsAreActive(client eth.BeaconChainClient) error {
	in := new(ptypes.Empty)
	chainHead, err := client.GetChainHead(context.Background(), in)
	if err != nil {
		return fmt.Errorf("failed to get chain head: %v", err)
	}
	// Balances actually fluctuate but we just want to check initial balance.
	currentEpoch := chainHead.BlockSlot / params.BeaconConfig().SlotsPerEpoch
	if currentEpoch > 1 {
		return nil
	}
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
			return fmt.Errorf("expected genesis validator epoch to be 0, received %d", val.ActivationEpoch)
		}
		if val.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			return fmt.Errorf("expected genesis validator exit epoch to be far future, received %d", val.ExitEpoch)
		}
		if val.WithdrawableEpoch != params.BeaconConfig().FarFutureEpoch {
			return fmt.Errorf("expected genesis validator withdrawable epoch to be far future, received %d", val.WithdrawableEpoch)
		}
		if val.EffectiveBalance != params.BeaconConfig().MaxEffectiveBalance {
			return fmt.Errorf(
				"expected genesis validator effective balance to be %d, received %d",
				params.BeaconConfig().MaxEffectiveBalance,
				val.EffectiveBalance,
			)
		}
	}
	return nil
}

// ValidatorsParticipating ensures the validators have an acceptable participation rate.
// TODO(#3971) - Fix validator participation API to calculate based on previous epoch.
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

	slotsPerEpoch := float32(params.BeaconConfig().SlotsPerEpoch)
	partRate := participation.Participation.GlobalParticipationRate
	if partRate < (slotsPerEpoch-1)/slotsPerEpoch {
		return fmt.Errorf("validator participation not as high as expected, received: %f", partRate)
	}
	return nil
}
