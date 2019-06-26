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
	latestRandaoMixes := make([][]byte, params.BeaconConfig().LatestRandaoMixesLength)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = make([]byte, 32)
	}

	zeroHash := params.BeaconConfig().ZeroHash[:]

	latestActiveIndexRoots := make([][]byte, params.BeaconConfig().LatestActiveIndexRootsLength)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = zeroHash
	}

	crosslinks := make([]*pb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &pb.Crosslink{
			Shard: uint64(i),
		}
	}

	latestBlockRoots := make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = zeroHash
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
		ValidatorRegistry: nil,
		Balances:          nil,

		// Randomness and committees.
		LatestRandaoMixes: latestRandaoMixes,

		// Finality.
		PreviousJustifiedEpoch: 0,
		PreviousJustifiedRoot:  params.BeaconConfig().ZeroHash[:],
		CurrentJustifiedEpoch:  0,
		CurrentJustifiedRoot:   params.BeaconConfig().ZeroHash[:],
		JustificationBitfield:  0,
		FinalizedEpoch:         0,
		FinalizedRoot:          params.BeaconConfig().ZeroHash[:],

		// Recent state.
		CurrentCrosslinks:         crosslinks,
		PreviousCrosslinks:        crosslinks,
		LatestActiveIndexRoots:    latestActiveIndexRoots,
		LatestBlockRoots:          latestBlockRoots,
		LatestSlashedBalances:     nil,
		CurrentEpochAttestations:  []*pb.PendingAttestation{},
		PreviousEpochAttestations: []*pb.PendingAttestation{},
		LatestBlockHeader:         blkHeader,

		// Eth1 data.
		LatestEth1Data: eth1Data,
		Eth1DataVotes:  []*pb.Eth1Data{},
		DepositIndex:   0,
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
		eth1DataExists := !bytes.Equal(eth1Data.DepositRoot, []byte{})
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
	for i := 0; i < len(state.ValidatorRegistry); i++ {
		if state.ValidatorRegistry[i].EffectiveBalance >=
			params.BeaconConfig().MaxDepositAmount {
			state.ValidatorRegistry[i].ActivationEligibilityEpoch = 0
			state.ValidatorRegistry[i].ActivationEpoch = 0
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
	for i := uint64(0); i < params.BeaconConfig().LatestActiveIndexRootsLength; i++ {
		state.LatestActiveIndexRoots[i] = indexRoot[:]
	}
	return state, nil
}
