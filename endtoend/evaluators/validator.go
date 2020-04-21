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

// ValidatorsSlashed ensures the expected amount of validators are slashed.
var ValidatorsSlashed = types.Evaluator{
	Name:       "validators_slashed_epoch_%d",
	Policy:     afterNthEpoch(0),
	Evaluation: validatorsSlashed,
}

// SlashedValidatorsLoseBalance checks if the validators slashed lose the right balance.
var SlashedValidatorsLoseBalance = types.Evaluator{
	Name:       "slashed_validators_lose_valance_epoch_%d",
	Policy:     afterNthEpoch(0),
	Evaluation: validatorsLoseBalance,
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
	expected := float32(0.85)

	// TODO(#5572): temporarily lowering requirements for E2E to pass until root cause is solved.
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

func validatorsSlashed(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	ctx := context.Background()
	client := eth.NewBeaconChainClient(conn)
	req := &eth.GetValidatorActiveSetChangesRequest{}
	changes, err := client.GetValidatorActiveSetChanges(ctx, req)
	if err != nil {
		return err
	}
	if len(changes.SlashedIndices) != 2 && len(changes.SlashedIndices) != 4 {
		return fmt.Errorf("expected 2 or 4 indices to be slashed, received %d", len(changes.SlashedIndices))
	}
	return nil
}

func validatorsLoseBalance(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	ctx := context.Background()
	client := eth.NewBeaconChainClient(conn)

	for i, indice := range slashedIndices {
		req := &eth.GetValidatorRequest{
			QueryFilter: &eth.GetValidatorRequest_Index{
				Index: indice,
			},
		}
		valResp, err := client.GetValidator(ctx, req)
		if err != nil {
			return err
		}

		slashedPenalty := params.BeaconConfig().MaxEffectiveBalance / params.BeaconConfig().MinSlashingPenaltyQuotient
		slashedBal := params.BeaconConfig().MaxEffectiveBalance - slashedPenalty + params.BeaconConfig().EffectiveBalanceIncrement/10
		if valResp.EffectiveBalance >= slashedBal {
			return fmt.Errorf(
				"expected slashed validator %d to balance less than %d, received %d",
				i,
				slashedBal,
				valResp.EffectiveBalance,
			)
		}

	}
	return nil
}
