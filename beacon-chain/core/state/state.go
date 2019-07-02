// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"bytes"
	"fmt"

	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BeaconState gets called to return a beacon state where all the fields are set to genesis values.
func BeaconState(blkHeader *pb.BeaconBlockHeader, genesisTime uint64, eth1Data *pb.Eth1Data) *pb.BeaconState {
	latestRandaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = make([]byte, 32)
	}

	zeroHash := params.BeaconConfig().ZeroHash[:]

	latestActiveIndexRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = zeroHash
	}

	latestSlashedExitBalances := make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)
	if eth1Data == nil {
		eth1Data = &pb.Eth1Data{}
	}

	crosslinks := make([]*pb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &pb.Crosslink{
			Shard: uint64(i),
		}
	}

	latestBlockRoots := make([][]byte, params.BeaconConfig().HistoricalRootsLimit)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = zeroHash
	}

	if eth1Data == nil {
		eth1Data = &pb.Eth1Data{}
	}

	state := &pb.BeaconState{
		// Misc fields.
		Slot:        0,
		GenesisTime: genesisTime,

		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			Epoch:           0,
		},

		// Validator registry fields.
		Validators: []*pb.Validator{},
		Balances:   []uint64{},

		// Randomness and committees.
		RandaoMixes: latestRandaoMixes,

		// Finality.
		PreviousJustifiedCheckpoint: &pb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &pb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		JustificationBits: []byte{0},
		FinalizedCheckpoint: &pb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},

		// Recent state.
		CurrentCrosslinks:         crosslinks,
		PreviousCrosslinks:        crosslinks,
		ActiveIndexRoots:          latestActiveIndexRoots,
		BlockRoots:                latestBlockRoots,
		Slashings:                 latestSlashedExitBalances,
		CurrentEpochAttestations:  []*pb.PendingAttestation{},
		PreviousEpochAttestations: []*pb.PendingAttestation{},
		LatestBlockHeader:         blkHeader,

		// Eth1 data.
		Eth1Data:         eth1Data,
		Eth1DataVotes:    []*pb.Eth1Data{},
		Eth1DepositIndex: 0,
	}
	return state
}

// GenesisBeaconState gets called when DepositsForChainStart count of
// full deposits were made to the deposit contract and the ChainStart log gets emitted.
//
// Spec pseudocode definition:
//  def get_genesis_beacon_state(deposits: List[Deposit], genesis_time: int, genesis_eth1_data: Eth1Data) -> BeaconState:
//    state = BeaconState(
//        genesis_time=genesis_time,
//        latest_eth1_data=genesis_eth1_data,
//        latest_block_header=BeaconBlockHeader(body_root=hash_tree_root(BeaconBlockBody())),
//    )
//
//    # Process genesis deposits
//    for deposit in deposits:
//        process_deposit(state, deposit)
//
//    # Process genesis activations
//    for validator in state.validator_registry:
//        if validator.effective_balance >= MAX_EFFECTIVE_BALANCE:
//            validator.activation_eligibility_epoch = GENESIS_EPOCH
//            validator.activation_epoch = GENESIS_EPOCH
//
//    # Populate latest_active_index_roots
//    genesis_active_index_root = hash_tree_root(get_active_validator_indices(state, GENESIS_EPOCH))
//    for index in range(LATEST_ACTIVE_INDEX_ROOTS_LENGTH):
//        state.latest_active_index_roots[index] = genesis_active_index_root
//
//    return state

// def initialize_beacon_state_from_eth1(eth1_block_hash: Hash,
//                                       eth1_timestamp: uint64,
//                                       deposits: Sequence[Deposit]) -> BeaconState:
//     state = BeaconState(
//         genesis_time=eth1_timestamp - eth1_timestamp % SECONDS_PER_DAY + 2 * SECONDS_PER_DAY,
//         eth1_data=Eth1Data(block_hash=eth1_block_hash, deposit_count=len(deposits)),
//         latest_block_header=BeaconBlockHeader(body_root=hash_tree_root(BeaconBlockBody())),
//     )
//
//     # Process deposits
//     leaves = list(map(lambda deposit: deposit.data, deposits))
//     for index, deposit in enumerate(deposits):
//         deposit_data_list = List[DepositData, 2**DEPOSIT_CONTRACT_TREE_DEPTH](*leaves[:index + 1])
//         state.eth1_data.deposit_root = hash_tree_root(deposit_data_list)
//         process_deposit(state, deposit)
//
//     # Process activations
//     for index, validator in enumerate(state.validators):
//         balance = state.balances[index]
//         validator.effective_balance = min(balance - balance % EFFECTIVE_BALANCE_INCREMENT, MAX_EFFECTIVE_BALANCE)
//         if validator.effective_balance == MAX_EFFECTIVE_BALANCE:
//             validator.activation_eligibility_epoch = GENESIS_EPOCH
//             validator.activation_epoch = GENESIS_EPOCH
//
//     # Populate active_index_roots and compact_committees_roots
//     indices_list = List[ValidatorIndex, VALIDATOR_REGISTRY_LIMIT](get_active_validator_indices(state, GENESIS_EPOCH))
//     active_index_root = hash_tree_root(indices_list)
//     committee_root = get_compact_committees_root(state, GENESIS_EPOCH)
//     for index in range(EPOCHS_PER_HISTORICAL_VECTOR):
//         state.active_index_roots[index] = active_index_root
//         state.compact_committees_roots[index] = committee_root
//     return state
func GenesisBeaconState(deposits []*pb.Deposit, genesisTime uint64, eth1Data *pb.Eth1Data) (*pb.BeaconState, error) {
	bodyRoot, err := ssz.HashTreeRoot(&pb.BeaconBlockBody{})
	if err != nil {
		return nil, fmt.Errorf("could not hash tree root: %v", bodyRoot)
	}
	blkHeader := &pb.BeaconBlockHeader{BodyRoot: bodyRoot[:]}

	state := BeaconState(blkHeader, genesisTime, eth1Data)

	// Process genesis deposits
	validatorMap := make(map[[32]byte]int)
	for _, deposit := range deposits {
		eth1DataExists := eth1Data != nil && !bytes.Equal(eth1Data.DepositRoot, []byte{})
		state, err = b.ProcessDeposit(
			state,
			deposit,
			validatorMap,
			false,
			eth1DataExists,
		)
		if err != nil {
			return nil, fmt.Errorf("could not process validator deposit: %v", err)
		}
	}

	// Process genesis activations
	for i := 0; i < len(state.Validators); i++ {
		if state.Validators[i].EffectiveBalance >=
			params.BeaconConfig().MaxEffectiveBalance {
			state.Validators[i].ActivationEligibilityEpoch = 0
			state.Validators[i].ActivationEpoch = 0
		}
	}

	// Populate latest_active_index_roots
	activeIndices, err := helpers.ActiveValidatorIndices(state, 0)
	if err != nil {
		return nil, fmt.Errorf("could not get active validator indices: %v", err)
	}
	indexRoot, err := ssz.HashTreeRoot(activeIndices)
	if err != nil {
		return nil, fmt.Errorf("could not hash tree root: %v", err)
	}
	for i := uint64(0); i < params.BeaconConfig().EpochsPerHistoricalVector; i++ {
		state.ActiveIndexRoots[i] = indexRoot[:]
	}
	return state, nil
}
