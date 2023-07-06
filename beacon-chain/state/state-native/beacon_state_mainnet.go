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
	GenesisTime                         uint64                                  `json:"genesisTime" yaml:"genesisTime"`
	GenesisValidatorsRoot               [32]byte                                `json:"genesisValidatorsRoot" yaml:"genesisValidatorsRoot"`
	Slot                                primitives.Slot                         `json:"slot" yaml:"slot"`
	Fork                                *ethpb.Fork                             `json:"fork" yaml:"fork"`
	LatestBlockHeader                   *ethpb.BeaconBlockHeader                `json:"latestBlockHeader" yaml:"latestBlockHeader"`
	BlockRoots                          *customtypes.BlockRoots                 `json:"blockRoots" yaml:"blockRoots"`
	StateRoots                          *customtypes.StateRoots                 `json:"stateRoots" yaml:"stateRoots"`
	HistoricalRoots                     customtypes.HistoricalRoots             `json:"historicalRoots" yaml:"historicalRoots"`
	HistoricalSummaries                 []*ethpb.HistoricalSummary              `json:"historicalSummaries" yaml:"historicalSummaries"`
	Eth1Data                            *ethpb.Eth1Data                         `json:"eth1Data" yaml:"eth1Data"`
	Eth1DataVotes                       []*ethpb.Eth1Data                       `json:"eth1DataVotes" yaml:"eth1DataVotes"`
	Eth1DepositIndex                    uint64                                  `json:"eth1DepositIndex" yaml:"eth1DepositIndex"`
	Validators                          []*ethpb.Validator                      `json:"validators" yaml:"validators"`
	Balances                            []uint64                                `json:"balances" yaml:"balances"`
	RandaoMixes                         *customtypes.RandaoMixes                `json:"randaoMixes" yaml:"randaoMixes"`
	Slashings                           []uint64                                `json:"slashings" yaml:"slashings"`
	PreviousEpochAttestations           []*ethpb.PendingAttestation             `json:"previousEpochAttestations" yaml:"previousEpochAttestations"`
	CurrentEpochAttestations            []*ethpb.PendingAttestation             `json:"currentEpochAttestations" yaml:"currentEpochAttestations"`
	PreviousEpochParticipation          []byte                                  `json:"previousEpochParticipation" yaml:"previousEpochParticipation"`
	CurrentEpochParticipation           []byte                                  `json:"currentEpochParticipation" yaml:"currentEpochParticipation"`
	JustificationBits                   bitfield.Bitvector4                     `json:"justificationBits" yaml:"justificationBits"`
	PreviousJustifiedCheckpoint         *ethpb.Checkpoint                       `json:"previousJustifiedCheckpoint" yaml:"previousJustifiedCheckpoint"`
	CurrentJustifiedCheckpoint          *ethpb.Checkpoint                       `json:"currentJustifiedCheckpoint" yaml:"currentJustifiedCheckpoint"`
	FinalizedCheckpoint                 *ethpb.Checkpoint                       `json:"finalizedCheckpoint" yaml:"finalizedCheckpoint"`
	InactivityScores                    []uint64                                `json:"inactivityScores" yaml:"inactivityScores"`
	CurrentSyncCommittee                *ethpb.SyncCommittee                    `json:"currentSyncCommittee" yaml:"currentSyncCommittee"`
	NextSyncCommittee                   *ethpb.SyncCommittee                    `json:"nextSyncCommittee" yaml:"nextSyncCommittee"`
	LatestExecutionPayloadHeader        *enginev1.ExecutionPayloadHeader        `json:"latestExecutionPayloadHeader" yaml:"latestExecutionPayloadHeader"`
	LatestExecutionPayloadHeaderCapella *enginev1.ExecutionPayloadHeaderCapella `json:"latestExecutionPayloadHeaderCapella" yaml:"latestExecutionPayloadHeaderCapella"`
	NextWithdrawalIndex                 uint64                                  `json:"nextWithdrawalIndex" yaml:"nextWithdrawalIndex"`
	NextWithdrawalValidatorIndex        primitives.ValidatorIndex               `json:"nextWithdrawalValidatorIndex" yaml:"nextWithdrawalValidatorIndex"`
}

func (bs *BeaconState) MarshalJSON() ([]byte, error) {
	marshalable := &beaconStateMarshalable{
		Version:               bs.version,
		GenesisTime:           bs.genesisTime,
		GenesisValidatorsRoot: bs.genesisValidatorsRoot,
		Fork:                  bs.fork,
		LatestBlockHeader:     bs.latestBlockHeader,
		BlockRoots:            bs.blockRoots,
	}
	return json.Marshal(marshalable)
}

func (bs *BeaconState) MarshalYAML() ([]byte, error) {
	marshalable := &beaconStateMarshalable{
		Version:               bs.version,
		GenesisTime:           bs.genesisTime,
		GenesisValidatorsRoot: bs.genesisValidatorsRoot,
		Fork:                  bs.fork,
		LatestBlockHeader:     bs.latestBlockHeader,
		BlockRoots:            bs.blockRoots,
	}
	return yaml.Marshal(marshalable)
}
