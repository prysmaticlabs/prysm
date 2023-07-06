//go:build !minimal

package state_native

import (
	"encoding/json"
	"sync"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/fieldtrie"
	customtypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"gopkg.in/yaml.v2"
)

// BeaconState defines a struct containing utilities for the Ethereum Beacon Chain state, defining
// getters and setters for its respective values and helpful functions such as HashTreeRoot().
type BeaconState struct {
	version                             int
	genesisTime                         uint64
	genesisValidatorsRoot               [32]byte
	slot                                primitives.Slot
	fork                                *ethpb.Fork
	latestBlockHeader                   *ethpb.BeaconBlockHeader
	blockRoots                          *customtypes.BlockRoots
	stateRoots                          *customtypes.StateRoots
	historicalRoots                     customtypes.HistoricalRoots
	historicalSummaries                 []*ethpb.HistoricalSummary
	eth1Data                            *ethpb.Eth1Data
	eth1DataVotes                       []*ethpb.Eth1Data
	eth1DepositIndex                    uint64
	validators                          []*ethpb.Validator
	balances                            []uint64
	randaoMixes                         *customtypes.RandaoMixes
	slashings                           []uint64
	previousEpochAttestations           []*ethpb.PendingAttestation
	currentEpochAttestations            []*ethpb.PendingAttestation
	previousEpochParticipation          []byte
	currentEpochParticipation           []byte
	justificationBits                   bitfield.Bitvector4
	previousJustifiedCheckpoint         *ethpb.Checkpoint
	currentJustifiedCheckpoint          *ethpb.Checkpoint
	finalizedCheckpoint                 *ethpb.Checkpoint
	inactivityScores                    []uint64
	currentSyncCommittee                *ethpb.SyncCommittee
	nextSyncCommittee                   *ethpb.SyncCommittee
	latestExecutionPayloadHeader        *enginev1.ExecutionPayloadHeader
	latestExecutionPayloadHeaderCapella *enginev1.ExecutionPayloadHeaderCapella
	nextWithdrawalIndex                 uint64
	nextWithdrawalValidatorIndex        primitives.ValidatorIndex

	lock                  sync.RWMutex
	dirtyFields           map[types.FieldIndex]bool
	dirtyIndices          map[types.FieldIndex][]uint64
	stateFieldLeaves      map[types.FieldIndex]*fieldtrie.FieldTrie
	rebuildTrie           map[types.FieldIndex]bool
	valMapHandler         *stateutil.ValidatorMapHandler
	merkleLayers          [][][]byte
	sharedFieldReferences map[types.FieldIndex]*stateutil.Reference
}

type beaconStateMarshalable struct {
	Version                             int                                     `json:"version" yaml:"version"`
	GenesisTime                         uint64                                  `json:"genesis_time" yaml:"genesisTime"`
	GenesisValidatorsRoot               [32]byte                                `json:"genesis_validators_root" yaml:"genesisValidatorsRoot"`
	Slot                                primitives.Slot                         `json:"slot" yaml:"slot"`
	Fork                                *ethpb.Fork                             `json:"fork" yaml:"fork"`
	LatestBlockHeader                   *ethpb.BeaconBlockHeader                `json:"latest_block_header" yaml:"latestBlockHeader"`
	BlockRoots                          *customtypes.BlockRoots                 `json:"block_roots" yaml:"blockRoots"`
	StateRoots                          *customtypes.StateRoots                 `json:"state_roots" yaml:"stateRoots"`
	HistoricalRoots                     customtypes.HistoricalRoots             `json:"historical_roots" yaml:"historicalRoots"`
	HistoricalSummaries                 []*ethpb.HistoricalSummary              `json:"historical_summaries" yaml:"historicalSummaries"`
	Eth1Data                            *ethpb.Eth1Data                         `json:"eth_1_data" yaml:"eth1Data"`
	Eth1DataVotes                       []*ethpb.Eth1Data                       `json:"eth_1_data_votes" yaml:"eth1DataVotes"`
	Eth1DepositIndex                    uint64                                  `json:"eth_1_deposit_index" yaml:"eth1DepositIndex"`
	Validators                          []*ethpb.Validator                      `json:"validators" yaml:"validators"`
	Balances                            []uint64                                `json:"balances" yaml:"balances"`
	RandaoMixes                         *customtypes.RandaoMixes                `json:"randao_mixes" yaml:"randaoMixes"`
	Slashings                           []uint64                                `json:"slashings" yaml:"slashings"`
	PreviousEpochAttestations           []*ethpb.PendingAttestation             `json:"previous_epoch_attestations" yaml:"previousEpochAttestations"`
	CurrentEpochAttestations            []*ethpb.PendingAttestation             `json:"current_epoch_attestations" yaml:"currentEpochAttestations"`
	PreviousEpochParticipation          []byte                                  `json:"previous_epoch_participation" yaml:"previousEpochParticipation"`
	CurrentEpochParticipation           []byte                                  `json:"current_epoch_participation" yaml:"currentEpochParticipation"`
	JustificationBits                   bitfield.Bitvector4                     `json:"justification_bits" yaml:"justificationBits"`
	PreviousJustifiedCheckpoint         *ethpb.Checkpoint                       `json:"previous_justified_checkpoint" yaml:"previousJustifiedCheckpoint"`
	CurrentJustifiedCheckpoint          *ethpb.Checkpoint                       `json:"current_justified_checkpoint" yaml:"currentJustifiedCheckpoint"`
	FinalizedCheckpoint                 *ethpb.Checkpoint                       `json:"finalized_checkpoint" yaml:"finalizedCheckpoint"`
	InactivityScores                    []uint64                                `json:"inactivity_scores" yaml:"inactivityScores"`
	CurrentSyncCommittee                *ethpb.SyncCommittee                    `json:"current_sync_committee" yaml:"currentSyncCommittee"`
	NextSyncCommittee                   *ethpb.SyncCommittee                    `json:"next_sync_committee" yaml:"nextSyncCommittee"`
	LatestExecutionPayloadHeader        *enginev1.ExecutionPayloadHeader        `json:"latest_execution_payload_header" yaml:"latestExecutionPayloadHeader"`
	LatestExecutionPayloadHeaderCapella *enginev1.ExecutionPayloadHeaderCapella `json:"latest_execution_payload_header_capella" yaml:"latestExecutionPayloadHeaderCapella"`
	NextWithdrawalIndex                 uint64                                  `json:"next_withdrawal_index" yaml:"nextWithdrawalIndex"`
	NextWithdrawalValidatorIndex        primitives.ValidatorIndex               `json:"next_withdrawal_validator_index" yaml:"nextWithdrawalValidatorIndex"`
}

func (b *BeaconState) MarshalJSON() ([]byte, error) {
	marshalable := &beaconStateMarshalable{
		Version:                             b.version,
		GenesisTime:                         b.genesisTime,
		GenesisValidatorsRoot:               b.genesisValidatorsRoot,
		Slot:                                b.slot,
		Fork:                                b.fork,
		LatestBlockHeader:                   b.latestBlockHeader,
		BlockRoots:                          b.blockRoots,
		StateRoots:                          b.stateRoots,
		HistoricalRoots:                     b.historicalRoots,
		HistoricalSummaries:                 b.historicalSummaries,
		Eth1Data:                            b.eth1Data,
		Eth1DataVotes:                       b.eth1DataVotes,
		Eth1DepositIndex:                    b.eth1DepositIndex,
		Validators:                          b.validators,
		Balances:                            b.balances,
		RandaoMixes:                         b.randaoMixes,
		Slashings:                           b.slashings,
		PreviousEpochAttestations:           b.previousEpochAttestations,
		CurrentEpochAttestations:            b.currentEpochAttestations,
		PreviousEpochParticipation:          b.previousEpochParticipation,
		CurrentEpochParticipation:           b.currentEpochParticipation,
		JustificationBits:                   b.justificationBits,
		PreviousJustifiedCheckpoint:         b.previousJustifiedCheckpoint,
		CurrentJustifiedCheckpoint:          b.currentJustifiedCheckpoint,
		FinalizedCheckpoint:                 b.finalizedCheckpoint,
		InactivityScores:                    b.inactivityScores,
		CurrentSyncCommittee:                b.currentSyncCommittee,
		NextSyncCommittee:                   b.nextSyncCommittee,
		LatestExecutionPayloadHeader:        b.latestExecutionPayloadHeader,
		LatestExecutionPayloadHeaderCapella: b.latestExecutionPayloadHeaderCapella,
		NextWithdrawalIndex:                 b.nextWithdrawalIndex,
		NextWithdrawalValidatorIndex:        b.nextWithdrawalValidatorIndex,
	}
	return json.Marshal(marshalable)
}

func (b *BeaconState) MarshalYAML() (interface{}, error) {
	marshalable := &beaconStateMarshalable{
		Version:                             b.version,
		GenesisTime:                         b.genesisTime,
		GenesisValidatorsRoot:               b.genesisValidatorsRoot,
		Slot:                                b.slot,
		Fork:                                b.fork,
		LatestBlockHeader:                   b.latestBlockHeader,
		BlockRoots:                          b.blockRoots,
		StateRoots:                          b.stateRoots,
		HistoricalRoots:                     b.historicalRoots,
		HistoricalSummaries:                 b.historicalSummaries,
		Eth1Data:                            b.eth1Data,
		Eth1DataVotes:                       b.eth1DataVotes,
		Eth1DepositIndex:                    b.eth1DepositIndex,
		Validators:                          b.validators,
		Balances:                            b.balances,
		RandaoMixes:                         b.randaoMixes,
		Slashings:                           b.slashings,
		PreviousEpochAttestations:           b.previousEpochAttestations,
		CurrentEpochAttestations:            b.currentEpochAttestations,
		PreviousEpochParticipation:          b.previousEpochParticipation,
		CurrentEpochParticipation:           b.currentEpochParticipation,
		JustificationBits:                   b.justificationBits,
		PreviousJustifiedCheckpoint:         b.previousJustifiedCheckpoint,
		CurrentJustifiedCheckpoint:          b.currentJustifiedCheckpoint,
		FinalizedCheckpoint:                 b.finalizedCheckpoint,
		InactivityScores:                    b.inactivityScores,
		CurrentSyncCommittee:                b.currentSyncCommittee,
		NextSyncCommittee:                   b.nextSyncCommittee,
		LatestExecutionPayloadHeader:        b.latestExecutionPayloadHeader,
		LatestExecutionPayloadHeaderCapella: b.latestExecutionPayloadHeaderCapella,
		NextWithdrawalIndex:                 b.nextWithdrawalIndex,
		NextWithdrawalValidatorIndex:        b.nextWithdrawalValidatorIndex,
	}
	return yaml.Marshal(marshalable)
}
