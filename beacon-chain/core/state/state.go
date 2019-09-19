// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// GenesisBeaconState gets called when MinGenesisActiveValidatorCount count of
// full deposits were made to the deposit contract and the ChainStart log gets emitted.
//
// Spec pseudocode definition:
//	def initialize_beacon_state_from_eth1(eth1_block_hash: Hash,
//	  eth1_timestamp: uint64,
//	  deposits: Sequence[Deposit]) -> BeaconState:
//	  state = BeaconState(
//	    genesis_time=eth1_timestamp - eth1_timestamp % SECONDS_PER_DAY + 2 * SECONDS_PER_DAY,
//	    eth1_data=Eth1Data(block_hash=eth1_block_hash, deposit_count=len(deposits)),
//	    latest_block_header=BeaconBlockHeader(body_root=hash_tree_root(BeaconBlockBody())),
//	  )
//
//	  # Process deposits
//	  leaves = list(map(lambda deposit: deposit.data, deposits))
//	  for index, deposit in enumerate(deposits):
//	    deposit_data_list = List[DepositData, 2**DEPOSIT_CONTRACT_TREE_DEPTH](*leaves[:index + 1])
//	    state.eth1_data.deposit_root = hash_tree_root(deposit_data_list)
//	    process_deposit(state, deposit)
//
//	  # Process activations
//	  for index, validator in enumerate(state.validators):
//	    balance = state.balances[index]
//	    validator.effective_balance = min(balance - balance % EFFECTIVE_BALANCE_INCREMENT, MAX_EFFECTIVE_BALANCE)
//	    if validator.effective_balance == MAX_EFFECTIVE_BALANCE:
//	    validator.activation_eligibility_epoch = GENESIS_EPOCH
//	    validator.activation_epoch = GENESIS_EPOCH
//
//	  # Populate active_index_roots and compact_committees_roots
//	  indices_list = List[ValidatorIndex, VALIDATOR_REGISTRY_LIMIT](get_active_validator_indices(state, GENESIS_EPOCH))
//	  active_index_root = hash_tree_root(indices_list)
//	  committee_root = get_compact_committees_root(state, GENESIS_EPOCH)
//	  for index in range(EPOCHS_PER_HISTORICAL_VECTOR):
//	    state.active_index_roots[index] = active_index_root
//	    state.compact_committees_roots[index] = committee_root
//	  return state
func GenesisBeaconState(deposits []*ethpb.Deposit, genesisTime uint64, eth1Data *ethpb.Eth1Data) (*pb.BeaconState, error) {
	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		randaoMixes[i] = make([]byte, 32)
	}

	zeroHash := params.BeaconConfig().ZeroHash[:]

	activeIndexRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(activeIndexRoots); i++ {
		activeIndexRoots[i] = zeroHash
	}

	compactRoots := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)

	crosslinks := make([]*ethpb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &ethpb.Crosslink{
			ParentRoot: make([]byte, 32),
			DataRoot:   make([]byte, 32),
		}
	}

	blockRoots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := 0; i < len(blockRoots); i++ {
		blockRoots[i] = zeroHash
	}

	stateRoots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := 0; i < len(stateRoots); i++ {
		stateRoots[i] = zeroHash
	}

	slashings := make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)

	if eth1Data == nil {
		return nil, errors.New("no eth1data provided for genesis state")
	}

	eth1Data.DepositCount = uint64(len(deposits))

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
		Validators: []*ethpb.Validator{},
		Balances:   []uint64{},

		// Randomness and committees.
		RandaoMixes: randaoMixes,

		// Finality.
		PreviousJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},
		JustificationBits: []byte{0},
		FinalizedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
			Root:  params.BeaconConfig().ZeroHash[:],
		},

		// Recent state.
		CurrentCrosslinks:         crosslinks,
		PreviousCrosslinks:        crosslinks,
		ActiveIndexRoots:          activeIndexRoots,
		CompactCommitteesRoots:    compactRoots,
		HistoricalRoots:           [][]byte{},
		BlockRoots:                blockRoots,
		StateRoots:                stateRoots,
		Slashings:                 slashings,
		CurrentEpochAttestations:  []*pb.PendingAttestation{},
		PreviousEpochAttestations: []*pb.PendingAttestation{},

		// Eth1 data.
		Eth1Data:         eth1Data,
		Eth1DataVotes:    []*ethpb.Eth1Data{},
		Eth1DepositIndex: 0,
	}

	bodyRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlockBody{})
	if err != nil {
		return nil, errors.Wrapf(err, "could not hash tree root %v", bodyRoot)
	}

	state.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		ParentRoot: zeroHash,
		StateRoot:  zeroHash,
		BodyRoot:   bodyRoot[:],
		Signature:  params.BeaconConfig().EmptySignature[:],
	}

	// Process initial deposits.
	validatorMap := make(map[[32]byte]int)
	leaves := [][]byte{}
	for _, deposit := range deposits {
		hash, err := ssz.HashTreeRoot(deposit.Data)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, hash[:])
	}
	var trie *trieutil.MerkleTrie
	if len(leaves) > 0 {
		trie, err = trieutil.GenerateTrieFromItems(leaves, int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			return nil, err
		}
	} else {
		trie, err = trieutil.NewTrie(int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			return nil, err
		}
	}

	depositRoot := trie.Root()
	state.Eth1Data.DepositRoot = depositRoot[:]
	for i, deposit := range deposits {
		state, err = b.ProcessDeposit(state, deposit, validatorMap)
		if err != nil {
			return nil, errors.Wrapf(err, "could not process validator deposit %d", i)
		}
	}
	// Process genesis activations
	for i, validator := range state.Validators {
		balance := state.Balances[i]
		validator.EffectiveBalance = mathutil.Min(balance-balance%params.BeaconConfig().EffectiveBalanceIncrement, params.BeaconConfig().MaxEffectiveBalance)
		if state.Validators[i].EffectiveBalance ==
			params.BeaconConfig().MaxEffectiveBalance {
			state.Validators[i].ActivationEligibilityEpoch = 0
			state.Validators[i].ActivationEpoch = 0
		}
	}

	// Populate latest_active_index_roots
	activeIndices, err := helpers.ActiveValidatorIndices(state, 0)
	if err != nil {
		return nil, errors.Wrap(err, "could not get active validator indices")
	}
	genesisActiveIndexRoot, err := ssz.HashTreeRootWithCapacity(activeIndices, params.BeaconConfig().ValidatorRegistryLimit)
	if err != nil {
		return nil, errors.Wrap(err, "could not hash tree root active indices")
	}
	genesisCompactCommRoot, err := helpers.CompactCommitteesRoot(state, 0)
	if err != nil {
		return nil, errors.Wrap(err, "could not get compact committee root")
	}
	for i := uint64(0); i < params.BeaconConfig().EpochsPerHistoricalVector; i++ {
		state.ActiveIndexRoots[i] = genesisActiveIndexRoot[:]
		state.CompactCommitteesRoots[i] = genesisCompactCommRoot[:]
	}
	return state, nil
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
func IsValidGenesisState(chainStartDepositCount uint64, currentTime uint64) bool {
	if currentTime < params.BeaconConfig().MinGenesisTime {
		return false
	}
	if chainStartDepositCount < params.BeaconConfig().MinGenesisActiveValidatorCount {
		return false
	}
	return true
}
