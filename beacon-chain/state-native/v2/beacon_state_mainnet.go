// +build !minimal

package v2

import (
	"sync"

	eth2types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/go-bitfield"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state-native/fieldtrie"
	"github.com/prysmaticlabs/prysm/beacon-chain/state-native/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state-native/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// BeaconState defines a struct containing utilities for the eth2 chain state, defining
// getters and setters for its respective values and helpful functions such as HashTreeRoot().
type BeaconState struct {
	genesisTime                 uint64                      `ssz-gen:"true"`
	genesisValidatorsRoot       customtypes.Byte32          `ssz-gen:"true" ssz-size:"32"`
	slot                        eth2types.Slot              `ssz-gen:"true"`
	fork                        *ethpb.Fork                 `ssz-gen:"true"`
	latestBlockHeader           *ethpb.BeaconBlockHeader    `ssz-gen:"true"`
	blockRoots                  *customtypes.BlockRoots     `ssz-gen:"true" ssz-size:"8192,32"`
	stateRoots                  *customtypes.StateRoots     `ssz-gen:"true" ssz-size:"8192,32"`
	historicalRoots             customtypes.HistoricalRoots `ssz-gen:"true" ssz-size:"?,32" ssz-max:"16777216"`
	eth1Data                    *ethpb.Eth1Data             `ssz-gen:"true"`
	eth1DataVotes               []*ethpb.Eth1Data           `ssz-gen:"true" ssz-max:"2048"`
	eth1DepositIndex            uint64                      `ssz-gen:"true"`
	validators                  []*ethpb.Validator          `ssz-gen:"true" ssz-max:"1099511627776"`
	balances                    []uint64                    `ssz-gen:"true" ssz-max:"1099511627776"`
	randaoMixes                 *customtypes.RandaoMixes    `ssz-gen:"true" ssz-size:"65536,32"`
	slashings                   []uint64                    `ssz-gen:"true" ssz-size:"8192"`
	previousEpochParticipation  []byte                      `ssz-gen:"true" ssz-max:"1099511627776"`
	currentEpochParticipation   []byte                      `ssz-gen:"true" ssz-max:"1099511627776"`
	justificationBits           bitfield.Bitvector4         `ssz-gen:"true" ssz-size:"1"`
	previousJustifiedCheckpoint *ethpb.Checkpoint           `ssz-gen:"true"`
	currentJustifiedCheckpoint  *ethpb.Checkpoint           `ssz-gen:"true"`
	finalizedCheckpoint         *ethpb.Checkpoint           `ssz-gen:"true"`
	inactivityScores            []uint64                    `ssz-gen:"true" ssz-max:"1099511627776"`
	currentSyncCommittee        *ethpb.SyncCommittee        `ssz-gen:"true"`
	nextSyncCommittee           *ethpb.SyncCommittee        `ssz-gen:"true"`

	lock                  sync.RWMutex
	dirtyFields           map[types.FieldIndex]bool
	dirtyIndices          map[types.FieldIndex][]uint64
	stateFieldLeaves      map[types.FieldIndex]*fieldtrie.FieldTrie
	rebuildTrie           map[types.FieldIndex]bool
	valMapHandler         *stateutil.ValidatorMapHandler
	merkleLayers          [][][]byte
	sharedFieldReferences map[types.FieldIndex]*stateutil.Reference
}
