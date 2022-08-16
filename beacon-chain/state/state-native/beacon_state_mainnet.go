//go:build !minimal

package state_native

import (
	"sync"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/fieldtrie"
	customtypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/custom-types"
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	eth2types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// BeaconState defines a struct containing utilities for the Ethereum Beacon Chain state, defining
// getters and setters for its respective values and helpful functions such as HashTreeRoot().
type BeaconState struct {
	version                      int
	genesisTime                  uint64                           `ssz-gen:"true"`
	genesisValidatorsRoot        customtypes.Byte32               `ssz-gen:"true" ssz-size:"32"`
	slot                         eth2types.Slot                   `ssz-gen:"true"`
	fork                         *ethpb.Fork                      `ssz-gen:"true"`
	latestBlockHeader            *ethpb.BeaconBlockHeader         `ssz-gen:"true"`
	blockRoots                   *customtypes.BlockRoots          `ssz-gen:"true" ssz-size:"8192,32"`
	stateRoots                   *customtypes.StateRoots          `ssz-gen:"true" ssz-size:"8192,32"`
	historicalRoots              customtypes.HistoricalRoots      `ssz-gen:"true" ssz-size:"?,32" ssz-max:"16777216"`
	eth1Data                     *ethpb.Eth1Data                  `ssz-gen:"true"`
	eth1DataVotes                []*ethpb.Eth1Data                `ssz-gen:"true" ssz-max:"2048"`
	eth1DepositIndex             uint64                           `ssz-gen:"true"`
	validators                   []*ethpb.Validator               `ssz-gen:"true" ssz-max:"1099511627776"`
	balances                     []uint64                         `ssz-gen:"true" ssz-max:"1099511627776"`
	randaoMixes                  *customtypes.RandaoMixes         `ssz-gen:"true" ssz-size:"65536,32"`
	slashings                    []uint64                         `ssz-gen:"true" ssz-size:"8192"`
	previousEpochAttestations    []*ethpb.PendingAttestation      `ssz-gen:"true" ssz-max:"4096"`
	currentEpochAttestations     []*ethpb.PendingAttestation      `ssz-gen:"true" ssz-max:"4096"`
	previousEpochParticipation   []byte                           `ssz-gen:"true" ssz-max:"1099511627776"`
	currentEpochParticipation    []byte                           `ssz-gen:"true" ssz-max:"1099511627776"`
	justificationBits            bitfield.Bitvector4              `ssz-gen:"true" ssz-size:"1"`
	previousJustifiedCheckpoint  *ethpb.Checkpoint                `ssz-gen:"true"`
	currentJustifiedCheckpoint   *ethpb.Checkpoint                `ssz-gen:"true"`
	finalizedCheckpoint          *ethpb.Checkpoint                `ssz-gen:"true"`
	inactivityScores             []uint64                         `ssz-gen:"true" ssz-max:"1099511627776"`
	currentSyncCommittee         *ethpb.SyncCommittee             `ssz-gen:"true"`
	nextSyncCommittee            *ethpb.SyncCommittee             `ssz-gen:"true"`
	latestExecutionPayloadHeader *enginev1.ExecutionPayloadHeader `ssz-gen:"true"`

	lock                  sync.RWMutex
	dirtyFields           map[nativetypes.FieldIndex]bool
	dirtyIndices          map[nativetypes.FieldIndex][]uint64
	stateFieldLeaves      map[nativetypes.FieldIndex]*fieldtrie.FieldTrie
	rebuildTrie           map[nativetypes.FieldIndex]bool
	valMapHandler         *stateutil.ValidatorMapHandler
	merkleLayers          [][][]byte
	sharedFieldReferences map[nativetypes.FieldIndex]*stateutil.Reference
}
