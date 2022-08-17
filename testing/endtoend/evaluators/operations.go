package evaluators

import (
	"bytes"
	"context"
	"fmt"
	"math"

	"github.com/pkg/errors"
	corehelpers "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	e2e "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/policies"
	e2etypes "github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// exitedIndex holds the exited index from ProposeVoluntaryExit in memory so other functions don't confuse it
// for a normal validator.
var exitedIndex types.ValidatorIndex

// valExited is used to know if exitedIndex is set, since default value is 0.
var valExited bool

// churnLimit is normally 4 unless the validator set is extremely large.
var churnLimit = uint64(4)
var depositValCount = e2e.DepositCount

// Deposits should be processed in twice the length of the epochs per eth1 voting period.
var depositsInBlockStart = types.Epoch(math.Floor(float64(params.E2ETestConfig().EpochsPerEth1VotingPeriod) * 2))

// deposits included + finalization + MaxSeedLookahead for activation.
var depositActivationStartEpoch = depositsInBlockStart + 2 + params.E2ETestConfig().MaxSeedLookahead
var depositEndEpoch = depositActivationStartEpoch + types.Epoch(math.Ceil(float64(depositValCount)/float64(churnLimit)))

// ProcessesDepositsInBlocks ensures the expected amount of deposits are accepted into blocks.
var ProcessesDepositsInBlocks = e2etypes.Evaluator{
	Name:       "processes_deposits_in_blocks_epoch_%d",
	Policy:     policies.OnEpoch(depositsInBlockStart), // We expect all deposits to enter in one epoch.
	Evaluation: processesDepositsInBlocks,
}

// VerifyBlockGraffiti ensures the block graffiti is one of the random list.
var VerifyBlockGraffiti = e2etypes.Evaluator{
	Name:       "verify_graffiti_in_blocks_epoch_%d",
	Policy:     policies.AfterNthEpoch(0),
	Evaluation: verifyGraffitiInBlocks,
}

// ActivatesDepositedValidators ensures the expected amount of validator deposits are activated into the state.
var ActivatesDepositedValidators = e2etypes.Evaluator{
	Name:       "processes_deposit_validators_epoch_%d",
	Policy:     policies.BetweenEpochs(depositActivationStartEpoch, depositEndEpoch),
	Evaluation: activatesDepositedValidators,
}

// DepositedValidatorsAreActive ensures the expected amount of validators are active after their deposits are processed.
var DepositedValidatorsAreActive = e2etypes.Evaluator{
	Name:       "deposited_validators_are_active_epoch_%d",
	Policy:     policies.AfterNthEpoch(depositEndEpoch),
	Evaluation: depositedValidatorsAreActive,
}

// ProposeVoluntaryExit sends a voluntary exit from randomly selected validator in the genesis set.
var ProposeVoluntaryExit = e2etypes.Evaluator{
	Name:       "propose_voluntary_exit_epoch_%d",
	Policy:     policies.OnEpoch(7),
	Evaluation: proposeVoluntaryExit,
}

// ValidatorHasExited checks the beacon state for the exited validator and ensures its marked as exited.
var ValidatorHasExited = e2etypes.Evaluator{
	Name:       "voluntary_has_exited_%d",
	Policy:     policies.OnEpoch(8),
	Evaluation: validatorIsExited,
}

// ValidatorsVoteWithTheMajority verifies whether validator vote for eth1data using the majority algorithm.
var ValidatorsVoteWithTheMajority = e2etypes.Evaluator{
	Name:       "validators_vote_with_the_majority_%d",
	Policy:     policies.AfterNthEpoch(0),
	Evaluation: validatorsVoteWithTheMajority,
}

func processesDepositsInBlocks(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconChainClient(conn)
	chainHead, err := client.GetChainHead(context.Background(), &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}

	req := &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: chainHead.HeadEpoch - 1}}
	blks, err := client.ListBeaconBlocks(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "failed to get blocks from beacon-chain")
	}
	var numDeposits uint64
	for _, blk := range blks.BlockContainers {
		var slot types.Slot
		var eth1Data *ethpb.Eth1Data
		var deposits []*ethpb.Deposit
		switch blk.Block.(type) {
		case *ethpb.BeaconBlockContainer_Phase0Block:
			b := blk.GetPhase0Block().Block
			slot = b.Slot
			eth1Data = b.Body.Eth1Data
			deposits = b.Body.Deposits
		case *ethpb.BeaconBlockContainer_AltairBlock:
			b := blk.GetAltairBlock().Block
			slot = b.Slot
			eth1Data = b.Body.Eth1Data
			deposits = b.Body.Deposits
		default:
			return errors.New("block neither phase0 nor altair")
		}
		fmt.Printf(
			"Slot: %d with %d deposits, Eth1 block %#x with %d deposits\n",
			slot,
			len(deposits),
			eth1Data.BlockHash, eth1Data.DepositCount,
		)
		numDeposits += uint64(len(deposits))
	}
	if numDeposits != depositValCount {
		return fmt.Errorf("expected %d deposits to be processed, received %d", depositValCount, numDeposits)
	}
	return nil
}

func verifyGraffitiInBlocks(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconChainClient(conn)
	chainHead, err := client.GetChainHead(context.Background(), &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}
	req := &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: chainHead.HeadEpoch.Sub(1)}}
	blks, err := client.ListBeaconBlocks(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "failed to get blocks from beacon-chain")
	}
	for _, ctr := range blks.BlockContainers {
		blk, err := convertToBlockInterface(ctr)
		if err != nil {
			return err
		}
		var e bool
		slot := blk.Block().Slot()
		graffitiInBlock := blk.Block().Body().Graffiti()
		for _, graffiti := range helpers.Graffiti {
			if bytes.Equal(bytesutil.PadTo([]byte(graffiti), 32), graffitiInBlock) {
				e = true
				break
			}
		}
		if !e && slot != 0 {
			return errors.New("could not get graffiti from the list")
		}
	}

	return nil
}

func activatesDepositedValidators(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconChainClient(conn)

	chainHead, err := client.GetChainHead(context.Background(), &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}

	validatorRequest := &ethpb.ListValidatorsRequest{
		PageSize:  int32(params.BeaconConfig().MinGenesisActiveValidatorCount),
		PageToken: "1",
	}
	validators, err := client.ListValidators(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validators")
	}

	expectedCount := depositValCount
	receivedCount := uint64(len(validators.ValidatorList))
	if expectedCount != receivedCount {
		return fmt.Errorf("expected validator count to be %d, recevied %d", expectedCount, receivedCount)
	}

	epoch := chainHead.HeadEpoch
	depositsInEpoch := uint64(0)
	var effBalanceLowCount, exitEpochWrongCount, withdrawEpochWrongCount uint64
	for _, item := range validators.ValidatorList {
		if item.Validator.ActivationEpoch == epoch {
			depositsInEpoch++
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
	}
	if depositsInEpoch != churnLimit {
		return fmt.Errorf("expected %d deposits to be processed in epoch %d, received %d", churnLimit, epoch, depositsInEpoch)
	}

	if effBalanceLowCount > 0 {
		return fmt.Errorf(
			"%d validators did not have genesis validator effective balance of %d",
			effBalanceLowCount,
			params.BeaconConfig().MaxEffectiveBalance,
		)
	} else if exitEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have an exit epoch of far future epoch", exitEpochWrongCount)
	} else if withdrawEpochWrongCount > 0 {
		return fmt.Errorf("%d validators did not have a withdrawable epoch of far future epoch", withdrawEpochWrongCount)
	}
	return nil
}

func depositedValidatorsAreActive(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconChainClient(conn)
	validatorRequest := &ethpb.ListValidatorsRequest{
		PageSize:  int32(params.BeaconConfig().MinGenesisActiveValidatorCount),
		PageToken: "1",
	}
	validators, err := client.ListValidators(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validators")
	}

	expectedCount := depositValCount
	receivedCount := uint64(len(validators.ValidatorList))
	if expectedCount != receivedCount {
		return fmt.Errorf("expected validator count to be %d, recevied %d", expectedCount, receivedCount)
	}

	chainHead, err := client.GetChainHead(context.Background(), &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}

	inactiveCount, belowBalanceCount := 0, 0
	for _, item := range validators.ValidatorList {
		if !corehelpers.IsActiveValidator(item.Validator, chainHead.HeadEpoch) {
			inactiveCount++
		}
		if item.Validator.EffectiveBalance < params.BeaconConfig().MaxEffectiveBalance {
			belowBalanceCount++
		}
	}

	if inactiveCount > 0 {
		return fmt.Errorf(
			"%d validators were not active, expected %d active validators from deposits",
			inactiveCount,
			params.BeaconConfig().MinGenesisActiveValidatorCount,
		)
	}
	if belowBalanceCount > 0 {
		return fmt.Errorf(
			"%d validators did not have a proper balance, expected %d validators to have 32 ETH",
			belowBalanceCount,
			params.BeaconConfig().MinGenesisActiveValidatorCount,
		)
	}
	return nil
}

func proposeVoluntaryExit(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	valClient := ethpb.NewBeaconNodeValidatorClient(conn)
	beaconClient := ethpb.NewBeaconChainClient(conn)

	ctx := context.Background()
	chainHead, err := beaconClient.GetChainHead(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "could not get chain head")
	}

	_, privKeys, err := util.DeterministicDepositsAndKeys(params.BeaconConfig().MinGenesisActiveValidatorCount)
	if err != nil {
		return err
	}

	exitedIndex = types.ValidatorIndex(rand.Uint64() % params.BeaconConfig().MinGenesisActiveValidatorCount)
	valExited = true

	voluntaryExit := &ethpb.VoluntaryExit{
		Epoch:          chainHead.HeadEpoch,
		ValidatorIndex: exitedIndex,
	}
	req := &ethpb.DomainRequest{
		Epoch:  chainHead.HeadEpoch,
		Domain: params.BeaconConfig().DomainVoluntaryExit[:],
	}
	domain, err := valClient.DomainData(ctx, req)
	if err != nil {
		return err
	}
	signingData, err := signing.ComputeSigningRoot(voluntaryExit, domain.SignatureDomain)
	if err != nil {
		return err
	}
	signature := privKeys[exitedIndex].Sign(signingData[:])
	signedExit := &ethpb.SignedVoluntaryExit{
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
	client := ethpb.NewBeaconChainClient(conn)
	validatorRequest := &ethpb.GetValidatorRequest{
		QueryFilter: &ethpb.GetValidatorRequest_Index{
			Index: exitedIndex,
		},
	}
	validator, err := client.GetValidator(context.Background(), validatorRequest)
	if err != nil {
		return errors.Wrap(err, "failed to get validators")
	}
	if validator.ExitEpoch == params.BeaconConfig().FarFutureEpoch {
		return fmt.Errorf("expected validator %d to be submitted for exit", exitedIndex)
	}
	return nil
}

func validatorsVoteWithTheMajority(conns ...*grpc.ClientConn) error {
	conn := conns[0]
	client := ethpb.NewBeaconChainClient(conn)
	chainHead, err := client.GetChainHead(context.Background(), &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "failed to get chain head")
	}

	req := &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: chainHead.HeadEpoch.Sub(1)}}
	blks, err := client.ListBeaconBlocks(context.Background(), req)
	if err != nil {
		return errors.Wrap(err, "failed to get blocks from beacon-chain")
	}

	for _, blk := range blks.BlockContainers {
		var slot types.Slot
		var vote []byte
		switch blk.Block.(type) {
		case *ethpb.BeaconBlockContainer_Phase0Block:
			b := blk.GetPhase0Block().Block
			slot = b.Slot
			vote = b.Body.Eth1Data.BlockHash
		case *ethpb.BeaconBlockContainer_AltairBlock:
			b := blk.GetAltairBlock().Block
			slot = b.Slot
			vote = b.Body.Eth1Data.BlockHash
		case *ethpb.BeaconBlockContainer_BellatrixBlock:
			b := blk.GetBellatrixBlock().Block
			slot = b.Slot
			vote = b.Body.Eth1Data.BlockHash
		case *ethpb.BeaconBlockContainer_BlindedBellatrixBlock:
			b := blk.GetBlindedBellatrixBlock().Block
			slot = b.Slot
			vote = b.Body.Eth1Data.BlockHash
		default:
			return errors.New("block neither phase0,altair or bellatrix")
		}
		slotsPerVotingPeriod := params.E2ETestConfig().SlotsPerEpoch.Mul(uint64(params.E2ETestConfig().EpochsPerEth1VotingPeriod))

		// We treat epoch 1 differently from other epoch for two reasons:
		// - this evaluator is not executed for epoch 0 so we have to calculate the first slot differently
		// - for some reason the vote for the first slot in epoch 1 is 0x000... so we skip this slot
		var isFirstSlotInVotingPeriod bool
		if chainHead.HeadEpoch == 1 && slot%params.BeaconConfig().SlotsPerEpoch == 0 {
			continue
		}
		// We skipped the first slot so we treat the second slot as the starting slot of epoch 1.
		if chainHead.HeadEpoch == 1 {
			isFirstSlotInVotingPeriod = slot%params.BeaconConfig().SlotsPerEpoch == 1
		} else {
			isFirstSlotInVotingPeriod = slot%slotsPerVotingPeriod == 0
		}
		if isFirstSlotInVotingPeriod {
			expectedEth1DataVote = vote
			return nil
		}

		if !bytes.Equal(vote, expectedEth1DataVote) {
			return fmt.Errorf("incorrect eth1data vote for slot %d; expected: %#x vs voted: %#x",
				slot, expectedEth1DataVote, vote)
		}
	}
	return nil
}

var expectedEth1DataVote []byte

func convertToBlockInterface(obj *ethpb.BeaconBlockContainer) (interfaces.SignedBeaconBlock, error) {
	if obj.GetPhase0Block() != nil {
		return blocks.NewSignedBeaconBlock(obj.GetPhase0Block())
	}
	if obj.GetAltairBlock() != nil {
		return blocks.NewSignedBeaconBlock(obj.GetAltairBlock())
	}
	if obj.GetBellatrixBlock() != nil {
		return blocks.NewSignedBeaconBlock(obj.GetBellatrixBlock())
	}
	if obj.GetBlindedBellatrixBlock() != nil {
		return blocks.NewSignedBeaconBlock(obj.GetBlindedBellatrixBlock())
	}
	return nil, errors.New("container has no block")
}
