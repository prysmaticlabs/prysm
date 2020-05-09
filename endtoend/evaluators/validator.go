package evaluators

import (
	"context"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	e2e "github.com/prysmaticlabs/prysm/endtoend/params"
	"github.com/prysmaticlabs/prysm/endtoend/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc"
)

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

// ProcessesDepositedValidators ensures the expected amount of validator deposits are processed into the state.
var ProcessesDepositedValidators = types.Evaluator{
	Name:       "processes_deposit_validators_epoch_%d",
	Policy:     isBetweenEpochs(8, 12),
	Evaluation: processesDepositedValidators,
}

// DepositedValidatorsAreActive ensures the expected amount of validators are active after their deposits are processed.
var DepositedValidatorsAreActive = types.Evaluator{
	Name:       "deposited_validators_are_active_epoch_%d",
	Policy:     afterNthEpoch(12),
	Evaluation: depositedValidatorsAreActive,
}

// Not including first epoch because of issues with genesis.
func afterNthEpoch(afterEpoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return currentEpoch > afterEpoch
	}
}

// Not including first epoch because of issues with genesis.
func isBetweenEpochs(fromEpoch uint64, toEpoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return fromEpoch < currentEpoch && currentEpoch > toEpoch
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
	activeEpochWrongCount := 0
	exitEpochWrongCount := 0
	withdrawEpochWrongCount := 0
	for _, item := range validators.ValidatorList {
		if item.Validator.EffectiveBalance < params.BeaconConfig().MaxEffectiveBalance {
			effBalanceLowCount++
		}
		if item.Validator.ActivationEpoch != 0 {
			activeEpochWrongCount++
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
	} else if activeEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have genesis validator epoch of 0", activeEpochWrongCount)
	} else if exitEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have genesis validator exit epoch of far future epoch", exitEpochWrongCount)
	} else if activeEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have genesis validator withdrawable epoch of far future epoch", activeEpochWrongCount)
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
	expected := float32(1)
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

func processesDepositedValidators(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := eth.NewBeaconChainClient(conn)
	validatorRequest := &eth.ListValidatorsRequest{
		PageSize:  int32(params.BeaconConfig().MinGenesisActiveValidatorCount),
		PageToken: "1",
	}
	validators, err := client.ListValidators(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validators")
	}

	expectedCount := params.BeaconConfig().MinGenesisActiveValidatorCount / uint64(e2e.TestParams.BeaconNodeCount)
	receivedCount := uint64(len(validators.ValidatorList))
	if expectedCount != receivedCount {
		return fmt.Errorf("expected validator count to be %d, recevied %d", expectedCount, receivedCount)
	}

	churnLimit, err := helpers.ValidatorChurnLimit(params.BeaconConfig().MinGenesisActiveValidatorCount)
	if err != nil {
		return errors.Wrap(err, "failed to calculate churn limit")
	}
	effBalanceLowCount := 0
	activeEpoch10Count := 0
	activeEpoch11Count := 0
	activeEpoch12Count := 0
	activeEpoch13Count := 0
	exitEpochWrongCount := 0
	withdrawEpochWrongCount := 0
	for _, item := range validators.ValidatorList {
		if item.Validator.EffectiveBalance < params.BeaconConfig().MaxEffectiveBalance {
			effBalanceLowCount++
		}
		if item.Validator.ActivationEpoch == 10 {
			activeEpoch10Count++
		} else if item.Validator.ActivationEpoch == 11 {
			activeEpoch11Count++
		} else if item.Validator.ActivationEpoch == 12 {
			activeEpoch12Count++
		} else if item.Validator.ActivationEpoch == 13 {
			activeEpoch13Count++
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
	} else if activeEpoch10Count != int(churnLimit) {
		return fmt.Errorf("%d validators did not have activation epoch of 10", activeEpoch10Count)
	} else if activeEpoch11Count != int(churnLimit) {
		return fmt.Errorf("%d validators did not have activation epoch of 11", activeEpoch11Count)
	} else if activeEpoch12Count != int(churnLimit) {
		return fmt.Errorf("%d validators did not have activation epoch of 12", activeEpoch12Count)
	} else if activeEpoch13Count != int(churnLimit) {
		return fmt.Errorf("%d validators did not have activation epoch of 13", activeEpoch13Count)
	} else if exitEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have an exit epoch of far future epoch", exitEpochWrongCount)
	} else if withdrawEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have a withdrawable epoch of far future epoch", withdrawEpochWrongCount)
	}
	return nil
}

func depositedValidatorsAreActive(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := eth.NewBeaconChainClient(conn)
	validatorRequest := &eth.ListValidatorsRequest{
		PageSize:  int32(params.BeaconConfig().MinGenesisActiveValidatorCount),
		PageToken: "1",
	}
	validators, err := client.ListValidators(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validators")
	}

	expectedCount := params.BeaconConfig().MinGenesisActiveValidatorCount / uint64(e2e.TestParams.BeaconNodeCount)
	receivedCount := uint64(len(validators.ValidatorList))
	if expectedCount != receivedCount {
		return fmt.Errorf("expected validator count to be %d, recevied %d", expectedCount, receivedCount)
	}

	chainHead, err := client.GetChainHead(context.Background(), &ptypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}

	inactiveCount := 0
	for _, item := range validators.ValidatorList {
		if !helpers.IsActiveValidator(item.Validator, chainHead.HeadEpoch) {
			inactiveCount++
		}
	}

	if inactiveCount > 0 {
		return fmt.Errorf(
			"%d validators were not active, expected %d active validators from deposits",
			inactiveCount,
			params.BeaconConfig().MinGenesisActiveValidatorCount,
		)
	}
	return nil
}
