// Package genesis defines the initial state and block for Ethereum 2.0's beacon chain.
package genesis

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

// NewGenesisBlock initializes an initial block for the Ethereum 2.0 beacon chain that is
// fixed in all clients and embedded in the protocol.
func NewGenesisBlock(stateRoot []byte) *pb.BeaconBlock {
	block := &pb.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot,
		ParentRootHash32: params.BeaconConfig().ZeroHash[:],
		StateRootHash32:  stateRoot,
		RandaoReveal:     params.BeaconConfig().ZeroHash[:],
		Signature:        params.BeaconConfig().EmptySignature[:],
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: params.BeaconConfig().ZeroHash[:],
			BlockHash32:       params.BeaconConfig().ZeroHash[:],
		},
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: []*pb.ProposerSlashing{},
			AttesterSlashings: []*pb.AttesterSlashing{},
			Attestations:      []*pb.Attestation{},
			Deposits:          []*pb.Deposit{},
			VoluntaryExits:    []*pb.VoluntaryExit{},
		},
	}
	return block
}

// BeaconState initializes a genesis beacon state - it gets called when DepositsForChainStart count of
// full deposits were made to the deposit contract and the ChainStart log gets emitted.
func BeaconState(
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
			Pubkey:                      depositInput.Pubkey,
			WithdrawalCredentialsHash32: depositInput.WithdrawalCredentialsHash32,
			ActivationEpoch:             params.BeaconConfig().FarFutureEpoch,
			ExitEpoch:                   params.BeaconConfig().FarFutureEpoch,
			SlashedEpoch:                params.BeaconConfig().FarFutureEpoch,
			WithdrawalEpoch:             params.BeaconConfig().FarFutureEpoch,
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
		ValidatorBalances:            latestBalances,
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
		JustifiedEpoch:         params.BeaconConfig().GenesisEpoch,
		JustificationBitfield:  0,
		FinalizedEpoch:         params.BeaconConfig().GenesisEpoch,

		// Recent state.
		LatestCrosslinks:        latestCrosslinks,
		LatestBlockRootHash32S:  latestBlockRoots,
		LatestIndexRootHash32S:  latestActiveIndexRoots,
		LatestSlashedBalances:   latestSlashedExitBalances,
		LatestAttestations:      []*pb.PendingAttestation{},
		BatchedBlockRootHash32S: [][]byte{},

		// Eth1 data.
		LatestEth1Data: eth1Data,
		Eth1DataVotes:  []*pb.Eth1DataVote{},
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
		state.LatestIndexRootHash32S[i] = genesisActiveIndexRoot[:]
	}
	seed, err := helpers.GenerateSeed(state, params.BeaconConfig().GenesisEpoch)
	if err != nil {
		return nil, fmt.Errorf("could not generate initial seed: %v", err)
	}
	state.CurrentShufflingSeedHash32 = seed[:]
	return state, nil
}
