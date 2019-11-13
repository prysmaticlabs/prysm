package evaluators

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Evaluator defines the structure of the evaluators used to
// conduct the current beacon state during the E2E.
type Evaluator struct {
	Name       string
	Policy     func(currentEpoch uint64) bool
	Evaluation func(client eth.BeaconChainClient) error
}

// ValidatorsAreActive ensures the expected amount of validators are active.
var ValidatorsAreActive = Evaluator{
	Name:       "validators_active_epoch_%d",
	Policy:     onGenesisEpoch,
	Evaluation: validatorsAreActive,
}

// ValidatorsParticipating ensures the expected amount of validators are active.
var ValidatorsParticipating = Evaluator{
	Name:       "validators_participating_epoch_%d",
	Policy:     everyEpochAfterGenesis,
	Evaluation: validatorsParticipating,
}

func onGenesisEpoch(currentEpoch uint64) bool {
	return currentEpoch < 2
}

func everyEpochAfterGenesis(currentEpoch uint64) bool {
	return currentEpoch > 0
}

func validatorsAreActive(client eth.BeaconChainClient) error {
	// Balances actually fluctuate but we just want to check initial balance.
	validatorRequest := &eth.GetValidatorsRequest{}
	validators, err := client.GetValidators(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validators")
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

// validatorsParticipating ensures the validators have an acceptable participation rate.
func validatorsParticipating(client eth.BeaconChainClient) error {
	validatorRequest := &eth.GetValidatorParticipationRequest{}
	participation, err := client.GetValidatorParticipation(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validator participation")
	}

	slotsPerEpoch := float64(params.BeaconConfig().SlotsPerEpoch)
	totalBalance := float64(participation.Participation.EligibleEther)
	partRate := participation.Participation.GlobalParticipationRate
	expected := float32((totalBalance - totalBalance/slotsPerEpoch) / totalBalance)
	if partRate < expected {
		return fmt.Errorf("validator participation was below expected %f, received: %f", expected, partRate)
	}
	return nil
}
