package evaluators

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethtypes "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/endtoend/helpers"
	"github.com/prysmaticlabs/prysm/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/time/slots"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var expectedParticipation = 0.95

var expectedSyncParticipation = 0.99

// ValidatorsAreActive ensures the expected amount of validators are active.
var ValidatorsAreActive = types.Evaluator{
	Name:       "validators_active_epoch_%d",
	Policy:     policies.AllEpochs,
	Evaluation: validatorsAreActive,
}

// ValidatorsParticipatingAtEpoch ensures the expected amount of validators are participating.
var ValidatorsParticipatingAtEpoch = func(epoch ethtypes.Epoch) types.Evaluator {
	return types.Evaluator{
		Name:       "validators_participating_epoch_%d",
		Policy:     policies.AfterNthEpoch(epoch),
		Evaluation: validatorsParticipating,
	}
}

// ValidatorSyncParticipation ensures the expected amount of sync committee participants
// are active.
var ValidatorSyncParticipation = types.Evaluator{
	Name:       "validator_sync_participation_%d",
	Policy:     policies.AfterNthEpoch(helpers.AltairE2EForkEpoch - 1),
	Evaluation: validatorsSyncParticipation,
}

func validatorsAreActive(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconChainClient(conn)
	// Balances actually fluctuate but we just want to check initial balance.
	validatorRequest := &ethpb.ListValidatorsRequest{
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
		if valExited && item.Index == exitedIndex {
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
	client := ethpb.NewBeaconChainClient(conn)
	validatorRequest := &ethpb.GetValidatorParticipationRequest{}
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

// validatorsSyncParticipation ensures the validators have an acceptable participation rate for
// sync committee assignments.
func validatorsSyncParticipation(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewNodeClient(conn)
	altairClient := ethpb.NewBeaconChainClient(conn)
	genesis, err := client.GetGenesis(context.Background(), &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get genesis data")
	}
	currSlot := slots.CurrentSlot(uint64(genesis.GenesisTime.AsTime().Unix()))
	currEpoch := slots.ToEpoch(currSlot)
	lowestBound := currEpoch - 1

	if lowestBound < helpers.AltairE2EForkEpoch {
		lowestBound = helpers.AltairE2EForkEpoch
	}
	blockCtrs, err := altairClient.ListBeaconBlocks(context.Background(), &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: lowestBound}})
	if err != nil {
		return errors.Wrap(err, "failed to get validator participation")
	}
	for _, ctr := range blockCtrs.BlockContainers {
		if ctr.GetAltairBlock() == nil {
			return errors.Errorf("Altair block type doesn't exist for block at epoch %d", lowestBound)
		}
		blk := ctr.GetAltairBlock()
		if blk.Block == nil || blk.Block.Body == nil || blk.Block.Body.SyncAggregate == nil {
			return errors.New("nil block provided")
		}
		forkSlot, err := slots.EpochStart(helpers.AltairE2EForkEpoch)
		if err != nil {
			return err
		}
		// Skip evaluation of the fork slot.
		if blk.Block.Slot == forkSlot {
			continue
		}
		syncAgg := blk.Block.Body.SyncAggregate
		threshold := uint64(float64(syncAgg.SyncCommitteeBits.Len()) * expectedSyncParticipation)
		if syncAgg.SyncCommitteeBits.Count() < threshold {
			return errors.Errorf("In block of slot %d ,the aggregate bitvector with length of %d only got a count of %d", blk.Block.Slot, threshold, syncAgg.SyncCommitteeBits.Count())
		}
	}
	if lowestBound == currEpoch {
		return nil
	}
	blockCtrs, err = altairClient.ListBeaconBlocks(context.Background(), &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: currEpoch}})
	if err != nil {
		return errors.Wrap(err, "failed to get validator participation")
	}
	for _, ctr := range blockCtrs.BlockContainers {
		if ctr.GetAltairBlock() == nil {
			return errors.Errorf("Altair block type doesn't exist for block at epoch %d", lowestBound)
		}
		blk := ctr.GetAltairBlock()
		if blk.Block == nil || blk.Block.Body == nil || blk.Block.Body.SyncAggregate == nil {
			return errors.New("nil block provided")
		}
		forkSlot, err := slots.EpochStart(helpers.AltairE2EForkEpoch)
		if err != nil {
			return err
		}
		// Skip evaluation of the fork slot.
		if blk.Block.Slot == forkSlot {
			continue
		}
		syncAgg := blk.Block.Body.SyncAggregate
		threshold := uint64(float64(syncAgg.SyncCommitteeBits.Len()) * expectedSyncParticipation)
		if syncAgg.SyncCommitteeBits.Count() < threshold {
			return errors.Errorf("In block of slot %d ,the aggregate bitvector with length of %d only got a count of %d", blk.Block.Slot, threshold, syncAgg.SyncCommitteeBits.Count())
		}
	}
	return nil
}
