package evaluators

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	ethtypes "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpbservice "github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	"github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	e2eparams "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/policies"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"

	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var expectedParticipation = 0.99

var expectedMulticlientParticipation = 0.98

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
	debugClient := ethpbservice.NewBeaconDebugClient(conn)
	validatorRequest := &ethpb.GetValidatorParticipationRequest{}
	participation, err := client.GetValidatorParticipation(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validator participation")
	}

	partRate := participation.Participation.GlobalParticipationRate
	expected := float32(expectedParticipation)
	if e2eparams.TestParams.LighthouseBeaconNodeCount != 0 {
		expected = float32(expectedMulticlientParticipation)
	}
	if participation.Epoch > 0 && participation.Epoch.Sub(1) == helpers.BellatrixE2EForkEpoch {
		// Reduce Participation requirement to 95% to account for longer EE calls for
		// the merge block. Target and head will likely be missed for a few validators at
		// slot 0.
		expected = 0.95
	}
	if partRate < expected {
		st, err := debugClient.GetBeaconStateV2(context.Background(), &eth.BeaconStateRequestV2{StateId: []byte("head")})
		if err != nil {
			return errors.Wrap(err, "failed to get beacon state")
		}
		var missSrcVals []uint64
		var missTgtVals []uint64
		var missHeadVals []uint64
		switch obj := st.Data.State.(type) {
		case *eth.BeaconStateContainer_Phase0State:
		// Do Nothing
		case *eth.BeaconStateContainer_AltairState:
			missSrcVals, missTgtVals, missHeadVals, err = findMissingValidators(obj.AltairState.PreviousEpochParticipation)
			if err != nil {
				return errors.Wrap(err, "failed to get missing validators")
			}
		case *eth.BeaconStateContainer_BellatrixState:
			missSrcVals, missTgtVals, missHeadVals, err = findMissingValidators(obj.BellatrixState.PreviousEpochParticipation)
			if err != nil {
				return errors.Wrap(err, "failed to get missing validators")
			}
		default:
			return fmt.Errorf("unrecognized version: %v", st.Version)
		}
		return fmt.Errorf(
			"validator participation was below for epoch %d, expected %f, received: %f."+
				" Missing Source,Target and Head validators are %v, %v, %v",
			participation.Epoch,
			expected,
			partRate,
			missSrcVals,
			missTgtVals,
			missHeadVals,
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
		b, err := syncCompatibleBlockFromCtr(ctr)
		if err != nil {
			return errors.Wrapf(err, "block type doesn't exist for block at epoch %d", lowestBound)
		}

		if b.IsNil() {
			return errors.New("nil block provided")
		}
		forkStartSlot, err := slots.EpochStart(helpers.AltairE2EForkEpoch)
		if err != nil {
			return err
		}
		if forkStartSlot == b.Block().Slot() {
			// Skip fork slot.
			continue
		}
		expectedParticipation := expectedSyncParticipation
		switch slots.ToEpoch(b.Block().Slot()) {
		case helpers.AltairE2EForkEpoch:
			// Drop expected sync participation figure.
			expectedParticipation = 0.90
		default:
			// no-op
		}
		syncAgg, err := b.Block().Body().SyncAggregate()
		if err != nil {
			return err
		}
		threshold := uint64(float64(syncAgg.SyncCommitteeBits.Len()) * expectedParticipation)
		if syncAgg.SyncCommitteeBits.Count() < threshold {
			return errors.Errorf("In block of slot %d ,the aggregate bitvector with length of %d only got a count of %d", b.Block().Slot(), threshold, syncAgg.SyncCommitteeBits.Count())
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
		b, err := syncCompatibleBlockFromCtr(ctr)
		if err != nil {
			return errors.Wrapf(err, "block type doesn't exist for block at epoch %d", lowestBound)
		}

		if b.IsNil() {
			return errors.New("nil block provided")
		}
		forkSlot, err := slots.EpochStart(helpers.AltairE2EForkEpoch)
		if err != nil {
			return err
		}
		nexForkSlot, err := slots.EpochStart(helpers.BellatrixE2EForkEpoch)
		if err != nil {
			return err
		}
		switch b.Block().Slot() {
		case forkSlot, forkSlot + 1, nexForkSlot:
			// Skip evaluation of the slot.
			continue
		default:
			// no-op
		}
		syncAgg, err := b.Block().Body().SyncAggregate()
		if err != nil {
			return err
		}
		threshold := uint64(float64(syncAgg.SyncCommitteeBits.Len()) * expectedSyncParticipation)
		if syncAgg.SyncCommitteeBits.Count() < threshold {
			return errors.Errorf("In block of slot %d ,the aggregate bitvector with length of %d only got a count of %d", b.Block().Slot(), threshold, syncAgg.SyncCommitteeBits.Count())
		}
	}
	return nil
}

func syncCompatibleBlockFromCtr(container *ethpb.BeaconBlockContainer) (interfaces.SignedBeaconBlock, error) {
	if container.GetPhase0Block() != nil {
		return nil, errors.New("block doesn't support sync committees")
	}
	if container.GetAltairBlock() != nil {
		return blocks.NewSignedBeaconBlock(container.GetAltairBlock())
	}
	if container.GetBellatrixBlock() != nil {
		return blocks.NewSignedBeaconBlock(container.GetBellatrixBlock())
	}
	if container.GetBlindedBellatrixBlock() != nil {
		return blocks.NewSignedBeaconBlock(container.GetBlindedBellatrixBlock())
	}
	return nil, errors.New("no supported block type in container")
}

func findMissingValidators(participation []byte) ([]uint64, []uint64, []uint64, error) {
	cfg := params.BeaconConfig()
	sourceFlagIndex := cfg.TimelySourceFlagIndex
	targetFlagIndex := cfg.TimelyTargetFlagIndex
	headFlagIndex := cfg.TimelyHeadFlagIndex
	var missingSourceValidators []uint64
	var missingHeadValidators []uint64
	var missingTargetValidators []uint64
	for i, b := range participation {
		hasSource, err := altair.HasValidatorFlag(b, sourceFlagIndex)
		if err != nil {
			return nil, nil, nil, err
		}
		if !hasSource {
			missingSourceValidators = append(missingSourceValidators, uint64(i))
		}
		hasTarget, err := altair.HasValidatorFlag(b, targetFlagIndex)
		if err != nil {
			return nil, nil, nil, err
		}
		if !hasTarget {
			missingTargetValidators = append(missingTargetValidators, uint64(i))
		}
		hasHead, err := altair.HasValidatorFlag(b, headFlagIndex)
		if err != nil {
			return nil, nil, nil, err
		}
		if !hasHead {
			missingHeadValidators = append(missingHeadValidators, uint64(i))
		}
	}
	return missingSourceValidators, missingTargetValidators, missingHeadValidators, nil
}
