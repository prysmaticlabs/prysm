package transition

import (
	"context"

	"github.com/pkg/errors"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// GenesisBeaconState gets called when MinGenesisActiveValidatorCount count of
// full deposits were made to the deposit contract and the ChainStart log gets emitted.
//
// Spec pseudocode definition:
//  def initialize_beacon_state_from_eth1(eth1_block_hash: Bytes32,
//                                      eth1_timestamp: uint64,
//                                      deposits: Sequence[Deposit]) -> BeaconState:
//    fork = Fork(
//        previous_version=GENESIS_FORK_VERSION,
//        current_version=GENESIS_FORK_VERSION,
//        epoch=GENESIS_EPOCH,
//    )
//    state = BeaconState(
//        genesis_time=eth1_timestamp + GENESIS_DELAY,
//        fork=fork,
//        eth1_data=Eth1Data(block_hash=eth1_block_hash, deposit_count=uint64(len(deposits))),
//        latest_block_header=BeaconBlockHeader(body_root=hash_tree_root(BeaconBlockBody())),
//        randao_mixes=[eth1_block_hash] * EPOCHS_PER_HISTORICAL_VECTOR,  # Seed RANDAO with Eth1 entropy
//    )
//
//    # Process deposits
//    leaves = list(map(lambda deposit: deposit.data, deposits))
//    for index, deposit in enumerate(deposits):
//        deposit_data_list = List[DepositData, 2**DEPOSIT_CONTRACT_TREE_DEPTH](*leaves[:index + 1])
//        state.eth1_data.deposit_root = hash_tree_root(deposit_data_list)
//        process_deposit(state, deposit)
//
//    # Process activations
//    for index, validator in enumerate(state.validators):
//        balance = state.balances[index]
//        validator.effective_balance = min(balance - balance % EFFECTIVE_BALANCE_INCREMENT, MAX_EFFECTIVE_BALANCE)
//        if validator.effective_balance == MAX_EFFECTIVE_BALANCE:
//            validator.activation_eligibility_epoch = GENESIS_EPOCH
//            validator.activation_epoch = GENESIS_EPOCH
//
//    # Set genesis validators root for domain separation and chain versioning
//    state.genesis_validators_root = hash_tree_root(state.validators)
//
//    return state
// This method differs from the spec so as to process deposits beforehand instead of the end of the function.
func GenesisBeaconState(ctx context.Context, deposits []*ethpb.Deposit, genesisTime uint64, eth1Data *ethpb.Eth1Data) (state.BeaconState, error) {
	state, err := EmptyGenesisState()
	if err != nil {
		return nil, err
	}

	// Process initial deposits.
	state, err = helpers.UpdateGenesisEth1Data(state, deposits, eth1Data)
	if err != nil {
		return nil, err
	}

	state, err = b.ProcessPreGenesisDeposits(ctx, state, deposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not process validator deposits")
	}

	return OptimizedGenesisBeaconState(genesisTime, state, state.Eth1Data())
}

// OptimizedGenesisBeaconState is used to create a state that has already processed deposits. This is to efficiently
// create a mainnet state at chainstart.
func OptimizedGenesisBeaconState(genesisTime uint64, preState state.BeaconState, eth1Data *ethpb.Eth1Data) (state.BeaconState, error) {
	if eth1Data == nil {
		return nil, errors.New("no eth1data provided for genesis state")
	}

	var randaoMixes [customtypes.RandaoMixesSize][32]byte
	for i := 0; i < len(randaoMixes); i++ {
		var h [32]byte
		copy(h[:], eth1Data.BlockHash)
		randaoMixes[i] = h
	}

	zeroHash32 := params.BeaconConfig().ZeroHash
	zeroHash := zeroHash32[:]

	activeIndexRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(activeIndexRoots); i++ {
		activeIndexRoots[i] = zeroHash
	}

	var blockRoots [customtypes.BlockRootsSize][32]byte
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = zeroHash32
	}

	var stateRoots [customtypes.StateRootsSize][32]byte
	for i := 0; i < len(stateRoots); i++ {
		stateRoots[i] = zeroHash32
	}

	slashings := make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)

	genesisValidatorsRoot, err := v1.ValidatorRegistryRoot(preState.Validators())
	if err != nil {
		return nil, errors.Wrapf(err, "could not hash tree root genesis validators %v", err)
	}

	bodyRoot, err := (&ethpb.BeaconBlockBody{
		RandaoReveal: make([]byte, 96),
		Eth1Data: &ethpb.Eth1Data{
			DepositRoot: make([]byte, 32),
			BlockHash:   make([]byte, 32),
		},
		Graffiti: make([]byte, 32),
	}).HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not hash tree root empty block body")
	}

	s, err := v1.Initialize()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize state from proto state")
	}

	if err = s.SetGenesisTime(genesisTime); err != nil {
		return nil, errors.Wrap(err, "could not set genesis time")
	}
	if err = s.SetGenesisValidatorRoot(genesisValidatorsRoot); err != nil {
		return nil, errors.Wrap(err, "could not set genesis validators root")
	}
	if err = s.SetSlot(0); err != nil {
		return nil, errors.Wrap(err, "could not set slot")
	}
	if err = s.SetFork(&ethpb.Fork{
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		Epoch:           0,
	}); err != nil {
		return nil, errors.Wrap(err, "could not set fork")
	}
	if err = s.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		ParentRoot: zeroHash,
		StateRoot:  zeroHash,
		BodyRoot:   bodyRoot[:],
	}); err != nil {
		return nil, errors.Wrap(err, "could not set latest block header")
	}
	if err = s.SetBlockRoots(&blockRoots); err != nil {
		return nil, errors.Wrap(err, "could not set block roots")
	}
	if err = s.SetStateRoots(&stateRoots); err != nil {
		return nil, errors.Wrap(err, "could not set state roots")
	}
	if err = s.SetHistoricalRoots([][32]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set historical roots")
	}
	if err = s.SetEth1Data(eth1Data); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data")
	}
	if err = s.SetEth1DataVotes([]*ethpb.Eth1Data{}); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data votes")
	}
	if err = s.SetEth1DepositIndex(preState.Eth1DepositIndex()); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 deposit index")
	}
	if err = s.SetValidators(preState.Validators()); err != nil {
		return nil, errors.Wrap(err, "could not set validators")
	}
	if err = s.SetBalances(preState.Balances()); err != nil {
		return nil, errors.Wrap(err, "could not set balances")
	}
	if err = s.SetRandaoMixes(&randaoMixes); err != nil {
		return nil, errors.Wrap(err, "could not set randao mixes")
	}
	if err = s.SetSlashings(slashings); err != nil {
		return nil, errors.Wrap(err, "could not set slashings")
	}
	if err = s.SetJustificationBits([]byte{0}); err != nil {
		return nil, errors.Wrap(err, "could not set justification bits")
	}
	if err = s.SetPreviousJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 0,
		Root:  params.BeaconConfig().ZeroHash[:],
	}); err != nil {
		return nil, errors.Wrap(err, "could not set previous justified checkpoint")
	}
	if err = s.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{
		Epoch: 0,
		Root:  params.BeaconConfig().ZeroHash[:],
	}); err != nil {
		return nil, errors.Wrap(err, "could not set current justified checkpoint")
	}
	if err = s.SetFinalizedCheckpoint(&ethpb.Checkpoint{
		Epoch: 0,
		Root:  params.BeaconConfig().ZeroHash[:],
	}); err != nil {
		return nil, errors.Wrap(err, "could not set finalized checkpoint")
	}

	return s, nil
}

// EmptyGenesisState returns an empty beacon state object.
func EmptyGenesisState() (state.BeaconState, error) {
	s, err := v1.Initialize()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize state from proto state")
	}

	if err = s.SetSlot(0); err != nil {
		return nil, errors.Wrap(err, "could not set slot")
	}
	if err = s.SetFork(&ethpb.Fork{
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		Epoch:           0,
	}); err != nil {
		return nil, errors.Wrap(err, "could not set fork")
	}
	if err = s.SetHistoricalRoots([][32]byte{}); err != nil {
		return nil, errors.Wrap(err, "could not set historical roots")
	}
	if err = s.SetEth1Data(&ethpb.Eth1Data{}); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data")
	}
	if err = s.SetEth1DataVotes([]*ethpb.Eth1Data{}); err != nil {
		return nil, errors.Wrap(err, "could not set eth1 data votes")
	}
	if err = s.SetValidators([]*ethpb.Validator{}); err != nil {
		return nil, errors.Wrap(err, "could not set validators")
	}
	if err = s.SetBalances([]uint64{}); err != nil {
		return nil, errors.Wrap(err, "could not set balances")
	}
	if err = s.SetJustificationBits([]byte{0}); err != nil {
		return nil, errors.Wrap(err, "could not set justification bits")
	}

	return s, nil
}

// IsValidGenesisState gets called whenever there's a deposit event,
// it checks whether there's enough effective balance to trigger and
// if the minimum genesis time arrived already.
//
// Spec pseudocode definition:
//  def is_valid_genesis_state(state: BeaconState) -> bool:
//     if state.genesis_time < MIN_GENESIS_TIME:
//         return False
//     if len(get_active_validator_indices(state, GENESIS_EPOCH)) < MIN_GENESIS_ACTIVE_VALIDATOR_COUNT:
//         return False
//     return True
// This method has been modified from the spec to allow whole states not to be saved
// but instead only cache the relevant information.
func IsValidGenesisState(chainStartDepositCount, currentTime uint64) bool {
	if currentTime < params.BeaconConfig().MinGenesisTime {
		return false
	}
	if chainStartDepositCount < params.BeaconConfig().MinGenesisActiveValidatorCount {
		return false
	}
	return true
}
