package evaluators

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc"
)

var expectedParticipation = 0.98 // 98% participation to make room for minor issues.

// ValidatorsAreActive ensures the expected amount of validators are active.
var ValidatorsAreActive = types.Evaluator{
	Name:       "validators_active_epoch_%d",
	Policy:     allEpochs,
	Evaluation: validatorsAreActive,
}

// ValidatorsParticipating ensures the expected amount of validators are active.
var ValidatorsParticipating = types.Evaluator{
	Name:       "validators_participating_epoch_%d",
	Policy:     afterNthEpoch(2),
	Evaluation: validatorsParticipating,
}

// Not including first epoch because of issues with genesis.
func afterNthEpoch(afterEpoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return currentEpoch > afterEpoch
	}
}

// All epochs.
func allEpochs(currentEpoch uint64) bool {
	return true
}

func validatorsAreActive(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := eth.NewBeaconChainClient(conn)
	// Balances actually fluctuate but we just want to check initial balance.
	validatorRequest := &eth.ListValidatorsRequest{
		PageSize: int32(params.BeaconConfig().MinGenesisActiveValidatorCount),
		Active:   true,
	}
	validators, err := client.ListValidators(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validators")
	}

	expectedCount := params.BeaconConfig().MinGenesisActiveValidatorCount
	receivedCount := uint64(len(validators.ValidatorList))
	if expectedCount != receivedCount {
		return fmt.Errorf("expected validator count to be %d, recevied %d", expectedCount, receivedCount)
	}

	effBalanceLowCount := 0
	exitEpochWrongCount := 0
	withdrawEpochWrongCount := 0
	for _, item := range validators.ValidatorList {
		if valExited && item.Index == exitedIndice {
			continue
		}
		if item.Validator.EffectiveBalance < params.BeaconConfig().MaxEffectiveBalance {
			effBalanceLowCount++
		}
		if item.Validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			exitEpochWrongCount++
		}
		if item.Validator.WithdrawableEpoch != params.BeaconConfig().FarFutureEpoch {
			withdrawEpochWrongCount++
		}
	}

	if effBalanceLowCount > 0 {
		return fmt.Errorf(
			"%d validators did not have genesis validator effective balance of %d",
			effBalanceLowCount,
			params.BeaconConfig().MaxEffectiveBalance,
		)
	} else if exitEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have genesis validator exit epoch of far future epoch", exitEpochWrongCount)
	} else if withdrawEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have genesis validator withdrawable epoch of far future epoch", withdrawEpochWrongCount)
	}

	return nil
}

// validatorsParticipating ensures the validators have an acceptable participation rate.
func validatorsParticipating(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := eth.NewBeaconChainClient(conn)
	validatorRequest := &eth.GetValidatorParticipationRequest{}
	participation, err := client.GetValidatorParticipation(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validator participation")
	}

	partRate := participation.Participation.GlobalParticipationRate
	expected := float32(expectedParticipation)
	if partRate < expected {
		return fmt.Errorf(
			"validator participation was below for epoch %d, expected %f, received: %f",
			participation.Epoch,
			expected,
			partRate,
		)
	}
	return nil
}
