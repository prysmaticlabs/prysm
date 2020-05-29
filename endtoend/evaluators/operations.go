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
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
)

// exitedIndice holds the exited indice from ProposeVoluntaryExit in memory so other functions don't confuse it
// for a normal validator.
var exitedIndice uint64

// valExited is used to know if exitedIndice is set, since default value is 0.
var valExited bool

// ProcessesDepositedValidators ensures the expected amount of validator deposits are processed into the state.
var ProcessesDepositedValidators = types.Evaluator{
	Name:       "processes_deposit_validators_epoch_%d",
	Policy:     isBetweenEpochs(8, 21), //Choosing 8-21 because of the churn limit of 4 per epoch for 256 vals / 4 beacon nodes = 64 deposits. )
	Evaluation: processesDepositedValidators,
}

// DepositedValidatorsAreActive ensures the expected amount of validators are active after their deposits are processed.
var DepositedValidatorsAreActive = types.Evaluator{
	Name:       "deposited_validators_are_active_epoch_%d",
	Policy:     afterNthEpoch(22),
	Evaluation: depositedValidatorsAreActive,
}

// ProposeVoluntaryExit sends a voluntary exit from randomly selected validator in the genesis set.
var ProposeVoluntaryExit = types.Evaluator{
	Name:       "propose_voluntary_exit_epoch_%d",
	Policy:     onEpoch(5),
	Evaluation: proposeVoluntaryExit,
}

// ValidatorHasExited checks the beacon state for the exited validator and ensures its marked as exited.
var ValidatorHasExited = types.Evaluator{
	Name:       "voluntary_has_exited_%d",
	Policy:     onEpoch(6),
	Evaluation: validatorIsExited,
}

// Not including first epoch because of issues with genesis.
func isBetweenEpochs(fromEpoch uint64, toEpoch uint64) func(uint64) bool {
	return func(currentEpoch uint64) bool {
		return fromEpoch < currentEpoch && currentEpoch > toEpoch
	}
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

	churnLimit, err := helpers.ValidatorChurnLimit(params.BeaconConfig().MinGenesisActiveValidatorCount + uint64(len(validators.ValidatorList)))
	if err != nil {
		return errors.Wrap(err, "failed to calculate churn limit")
	}
	var effBalanceLowCount, exitEpochWrongCount, withdrawEpochWrongCount uint64
	var activeEpoch10Count, activeEpoch11Count, activeEpoch12Count, activeEpoch13Count uint64
	for _, item := range validators.ValidatorList {
		switch item.Validator.ActivationEpoch {
		case 10:
			activeEpoch10Count++
		case 11:
			activeEpoch11Count++
		case 12:
			activeEpoch12Count++
		case 13:
			activeEpoch13Count++
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
	} else if activeEpoch10Count != churnLimit {
		return fmt.Errorf("%d validators did not have activation epoch of 10", activeEpoch10Count)
	} else if activeEpoch11Count != churnLimit {
		return fmt.Errorf("%d validators did not have activation epoch of 11", activeEpoch11Count)
	} else if activeEpoch12Count != churnLimit {
		return fmt.Errorf("%d validators did not have activation epoch of 12", activeEpoch12Count)
	} else if activeEpoch13Count != churnLimit {
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

func proposeVoluntaryExit(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	valClient := eth.NewBeaconNodeValidatorClient(conn)
	beaconClient := eth.NewBeaconChainClient(conn)

	ctx := context.Background()
	chainHead, err := beaconClient.GetChainHead(ctx, &ptypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "could not get chain head")
	}

	_, privKeys, err := testutil.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	if err != nil {
		return err
	}

	exitedIndice = rand.Uint64() % params.BeaconConfig().MinGenesisActiveValidatorCount
	valExited = true

	voluntaryExit := &eth.VoluntaryExit{
		Epoch:          chainHead.HeadEpoch,
		ValidatorIndex: exitedIndice,
	}
	req := &eth.DomainRequest{
		Epoch:  chainHead.HeadEpoch,
		Domain: params.BeaconConfig().DomainVoluntaryExit[:],
	}
	domain, err := valClient.DomainData(ctx, req)
	if err != nil {
		return err
	}
	signingData, err := helpers.ComputeSigningRoot(voluntaryExit, domain.SignatureDomain)
	if err != nil {
		return err
	}
	signature := privKeys[exitedIndice].Sign(signingData[:])
	signedExit := &eth.SignedVoluntaryExit{
		Exit:      voluntaryExit,
		Signature: signature.Marshal(),
	}

	if _, err = valClient.ProposeExit(ctx, signedExit); err != nil {
		return errors.Wrap(err, "could not propose exit")
	}
	return nil
}

func validatorIsExited(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := eth.NewBeaconChainClient(conn)
	validatorRequest := &eth.GetValidatorRequest{
		QueryFilter: &eth.GetValidatorRequest_Index{
			Index: exitedIndice,
		},
	}
	validator, err := client.GetValidator(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validators")
	}
	if validator.ExitEpoch == params.BeaconConfig().FarFutureEpoch {
		return fmt.Errorf("expected validator %d to be submitted for exit", exitedIndice)
	}
	return nil
}
