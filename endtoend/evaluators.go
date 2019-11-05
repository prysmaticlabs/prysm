package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/params"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

type policy func(chainHead *eth.ChainHead, options ...uint64) error
type evaluation func(client *eth.BeaconChainClient, options ...uint64) error

// Evaluator defines the function signature for function to run during the E2E.
type Evaluator struct {
	Policy     policy
	Evaluation evaluation
}

// EveryNEpochs run the evaluator every N epochs.
func EveryNEpochs(chainHead *eth.ChainHead, epochs uint64) bool {
	return helpers.SlotToEpoch(chainHead.BlockSlot)%epochs != 0
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

	for i, val := range validators.Validators {
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
	validatorRequest := &eth.GetValidatorsRequest{}
	validators, err := client.GetValidators(context.Background(), validatorRequest)
	if err != nil {
		return errors.New("failed to get validators")
	}

	receivedCount := uint64(len(validators.Validators))
	if expectedCount != receivedCount {
		return fmt.Errorf("expected validator count to be %d, recevied %d", expectedCount, receivedCount)
	}

	for i, val := range validators.Validators {
		if val.ActivationEpoch != 0 {
			return fmt.Errorf("genesis validator epoch should be 0, received %d", val.ActivationEpoch)
		}
		if val.EffectiveBalance <= 3.2*1e9 {
			return fmt.Errorf("expected genesis validator balance to be greater than 3.2 ETH, received %d", val.EffectiveBalance)
		}
		if val.WithdrawableEpoch != params.BeaconConfig().FarFutureEpoch {
			return fmt.Errorf()
		}
	}
}

// FinalizationOccurs is an evaluator to make sure finalization is performing as it should.
// Requires to be run after at least 4 epochs have passed.
func FinalizationOccurs(client eth.BeaconChainClient) error {
	in := new(ptypes.Empty)
	chainHead, err := client.GetChainHead(context.Background(), in)
	if err != nil {
		return errors.New("failed to get chain head")
	}

	currentEpoch := helpers.SlotToEpoch(chainHead.BlockSlot)
	if currentEpoch < 4 {
		return fmt.Errorf("current epoch is less than 2, received: %d", currentEpoch)
	}
	finalizedEpoch := helpers.SlotToEpoch(chainHead.FinalizedSlot)
	if finalizedEpoch < 2 {
		return fmt.Errorf("expected finalized epoch to be greater than 2, received: %d", currentEpoch)
	}
	return nil
}
