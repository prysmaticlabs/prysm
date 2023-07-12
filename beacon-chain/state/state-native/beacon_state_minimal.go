//go:build minimal

package state_native

import (
	"sync"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/fieldtrie"
	customtypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
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
	blockRoots                          customtypes.BlockRoots
	blockRootsMultiValue                *MultiValueBlockRoots
	stateRoots                          customtypes.StateRoots
	stateRootsMultiValue                *MultiValueStateRoots
	historicalRoots                     customtypes.HistoricalRoots
	historicalSummaries                 []*ethpb.HistoricalSummary
	eth1Data                            *ethpb.Eth1Data
	eth1DataVotes                       []*ethpb.Eth1Data
	eth1DepositIndex                    uint64
	validators                          []*ethpb.Validator
	validatorsMultiValue                *MultiValueValidators
	balances                            []uint64
	balancesMultiValue                  *MultiValueBalances
	randaoMixes                         customtypes.RandaoMixes
	randaoMixesMultiValue               *MultiValueRandaoMixes
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
	inactivityScoresMultiValue          *MultiValueInactivityScores
	currentSyncCommittee                *ethpb.SyncCommittee
	nextSyncCommittee                   *ethpb.SyncCommittee
	latestExecutionPayloadHeader        *enginev1.ExecutionPayloadHeader
	latestExecutionPayloadHeaderCapella *enginev1.ExecutionPayloadHeaderCapella
	nextWithdrawalIndex                 uint64
	nextWithdrawalValidatorIndex        primitives.ValidatorIndex

	id                    uint64
	lock                  sync.RWMutex
	dirtyFields           map[types.FieldIndex]bool
	dirtyIndices          map[types.FieldIndex][]uint64
	stateFieldLeaves      map[types.FieldIndex]*fieldtrie.FieldTrie
	rebuildTrie           map[types.FieldIndex]bool
	valMapHandler         *stateutil.ValidatorMapHandler
	merkleLayers          [][][]byte
	sharedFieldReferences map[types.FieldIndex]*stateutil.Reference
}
