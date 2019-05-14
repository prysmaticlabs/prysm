// Package state implements the whole state transition
// function which consists of per slot, per-epoch transitions.
// It also bootstraps the genesis beacon state for slot 0.
package state

import (
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// GenesisBeaconState gets called when DepositsForChainStart count of
// full deposits were made to the deposit contract and the ChainStart log gets emitted.
// Spec pseudocode definition:
// def get_genesis_beacon_state(genesis_validator_deposits: List[Deposit],
//                             genesis_time: int,
//                             genesis_eth1_data: Eth1Data) -> BeaconState:
//    """
//    Get the genesis ``BeaconState``.
//    """
//    state = BeaconState(genesis_time=genesis_time, latest_eth1_data=genesis_eth1_data)
//
//    # Process genesis deposits
//    for deposit in genesis_validator_deposits:
//        process_deposit(state, deposit)
//
//    # Process genesis activations
//    for validator in state.validator_registry:
//        if validator.effective_balance >= MAX_EFFECTIVE_BALANCE:
//            validator.activation_eligibility_epoch = GENESIS_EPOCH
//            validator.activation_epoch = GENESIS_EPOCH
//
//    genesis_active_index_root = hash_tree_root(get_active_validator_indices(state, GENESIS_EPOCH))
//    for index in range(LATEST_ACTIVE_INDEX_ROOTS_LENGTH):
//        state.latest_active_index_roots[index] = genesis_active_index_root
//
//    return state
func GenesisBeaconState(
	genesisValidatorDeposits []*pb.Deposit,
	genesisTime uint64,
	eth1Data *pb.Eth1Data,
) (*pb.BeaconState, error) {
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().LatestRandaoMixesLength,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = make([]byte, 32)
	}

	zeroHash := params.BeaconConfig().ZeroHash[:]

	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().LatestActiveIndexRootsLength,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = zeroHash
	}

	latestCrosslinks := make([]*pb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(latestCrosslinks); i++ {
		latestCrosslinks[i] = &pb.Crosslink{
			Epoch:                   params.BeaconConfig().GenesisEpoch,
			CrosslinkDataRootHash32: zeroHash,
		}
	}

	latestBlockRoots := make([][]byte, params.BeaconConfig().LatestBlockRootsLength)
	for i := 0; i < len(latestBlockRoots); i++ {
		latestBlockRoots[i] = zeroHash
	}

	validatorRegistry := make([]*pb.Validator, len(genesisValidatorDeposits))
	for i, d := range genesisValidatorDeposits {
		depositInput, err := helpers.DecodeDepositInput(d.DepositData)
		if err != nil {
			return nil, fmt.Errorf("could decode deposit input %v", err)
		}

		validator := &pb.Validator{
			Pubkey:                depositInput.Pubkey,
			WithdrawalCredentials: depositInput.WithdrawalCredentialsHash32,
			ActivationEpoch:       params.BeaconConfig().FarFutureEpoch,
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
            Slashed: false,
			WithdrawableEpoch:     params.BeaconConfig().FarFutureEpoch,
		}

		validatorRegistry[i] = validator
	}

	latestBalances := make([]uint64, len(genesisValidatorDeposits))
	latestSlashedExitBalances := make([]uint64, params.BeaconConfig().LatestSlashedExitLength)

	state := &pb.BeaconState{
		// Misc fields.
		Slot:        params.BeaconConfig().GenesisSlot,
		GenesisTime: genesisTime,

		Fork: &pb.Fork{
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			Epoch:           params.BeaconConfig().GenesisEpoch,
		},

		// Validator registry fields.
		ValidatorRegistry:            validatorRegistry,
		Balances:                     latestBalances,
		ValidatorRegistryUpdateEpoch: params.BeaconConfig().GenesisEpoch,

		// Randomness and committees.
		LatestRandaoMixes:           latestRandaoMixes,
		PreviousShufflingStartShard: params.BeaconConfig().GenesisStartShard,
		CurrentShufflingStartShard:  params.BeaconConfig().GenesisStartShard,
		PreviousShufflingEpoch:      params.BeaconConfig().GenesisEpoch,
		CurrentShufflingEpoch:       params.BeaconConfig().GenesisEpoch,
		PreviousShufflingSeedHash32: zeroHash,
		CurrentShufflingSeedHash32:  zeroHash,

		// Finality.
		PreviousJustifiedEpoch: params.BeaconConfig().GenesisEpoch,
		PreviousJustifiedRoot:  params.BeaconConfig().ZeroHash[:],
		CurrentJustifiedEpoch:  params.BeaconConfig().GenesisEpoch,
		CurrentJustifiedRoot:   params.BeaconConfig().ZeroHash[:],
		JustificationBitfield:  0,
		FinalizedEpoch:         params.BeaconConfig().GenesisEpoch,
		FinalizedRoot:          params.BeaconConfig().ZeroHash[:],

		// Recent state.
		LatestCrosslinks:        latestCrosslinks,
		LatestActiveIndexRoots:  latestActiveIndexRoots,
		LatestBlockRoots:        latestBlockRoots,
		LatestSlashedBalances:   latestSlashedExitBalances,
		LatestAttestations:      []*pb.PendingAttestation{},
		BatchedBlockRootHash32S: [][]byte{},

		// Eth1 data.
		LatestEth1Data: eth1Data,
		Eth1DataVotes:  []*pb.Eth1DataVote{},
		DepositIndex:   0,
	}

	// Process initial deposits.
	var err error
	validatorMap := stateutils.ValidatorIndexMap(state)
	for _, deposit := range genesisValidatorDeposits {
		depositData := deposit.DepositData
		depositInput, err := helpers.DecodeDepositInput(depositData)
		if err != nil {
			return nil, fmt.Errorf("could not decode deposit input: %v", err)
		}
		value, _, err := helpers.DecodeDepositAmountAndTimeStamp(depositData)
		if err != nil {
			return nil, fmt.Errorf("could not decode deposit value and timestamp: %v", err)
		}
		state, err = v.ProcessDeposit(
			state,
			validatorMap,
			depositInput.Pubkey,
			value,
			depositInput.ProofOfPossession,
			depositInput.WithdrawalCredentialsHash32,
		)
		if err != nil {
			return nil, fmt.Errorf("could not process validator deposit: %v", err)
		}
	}
	for i := 0; i < len(state.ValidatorRegistry); i++ {
		if helpers.EffectiveBalance(state, uint64(i)) >=
			params.BeaconConfig().MaxDepositAmount {
			state, err = v.ActivateValidator(state, uint64(i), true)
			if err != nil {
				return nil, fmt.Errorf("could not activate validator: %v", err)
			}
		}
	}
	activeValidators := helpers.ActiveValidatorIndices(state.ValidatorRegistry, params.BeaconConfig().GenesisEpoch)
	indicesBytes := []byte{}
	for _, val := range activeValidators {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, val)
		indicesBytes = append(indicesBytes, buf...)
	}
	genesisActiveIndexRoot := hashutil.Hash(indicesBytes)
	for i := uint64(0); i < params.BeaconConfig().LatestActiveIndexRootsLength; i++ {
		state.LatestActiveIndexRoots[i] = genesisActiveIndexRoot[:]
	}
	return state, nil
}
