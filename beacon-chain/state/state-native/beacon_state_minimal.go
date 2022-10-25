//go:build minimal

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
	lock                                sync.RWMutex
	version                             int
	genesisTime                         uint64                      `ssz-gen:"true"`
	eth1DepositIndex                    uint64                      `ssz-gen:"true"`
	nextWithdrawalIndex                 uint64                      `ssz-gen:"true"`
	previousEpochParticipation          []byte                      `ssz-gen:"true" ssz-max:"1099511627776"`
	currentEpochParticipation           []byte                      `ssz-gen:"true" ssz-max:"1099511627776"`
	randaoMixes                         *customtypes.RandaoMixes    `ssz-gen:"true" ssz-size:"64,32"`
	slot                                eth2types.Slot              `ssz-gen:"true"`
	fork                                *ethpb.Fork                 `ssz-gen:"true"`
	latestBlockHeader                   *ethpb.BeaconBlockHeader    `ssz-gen:"true"`
	stateRoots                          *customtypes.StateRoots     `ssz-gen:"true" ssz-size:"64,32"`
	currentSyncCommittee                *ethpb.SyncCommittee        `ssz-gen:"true"`
	genesisValidatorsRoot               [32]byte                    `ssz-gen:"true" ssz-size:"32"`
	eth1DataVotes                       []*ethpb.Eth1Data           `ssz-gen:"true" ssz-max:"32"`
	balances                            []uint64                    `ssz-gen:"true" ssz-max:"1099511627776"`
	inactivityScores                    []uint64                    `ssz-gen:"true" ssz-max:"1099511627776"`
	slashings                           []uint64                    `ssz-gen:"true" ssz-size:"64"`
	previousEpochAttestations           []*ethpb.PendingAttestation `ssz-gen:"true" ssz-max:"1024"`
	currentEpochAttestations            []*ethpb.PendingAttestation `ssz-gen:"true" ssz-max:"1024"`
	withdrawalQueue                     []*enginev1.Withdrawal      `ssz-gen:"true" ssz-max:"1099511627776"`
	validators                          []*ethpb.Validator          `ssz-gen:"true" ssz-max:"1099511627776"`
	valMapHandler                       *stateutil.ValidatorMapHandler
	blockRoots                          *customtypes.BlockRoots                 `ssz-gen:"true" ssz-size:"64,32"`
	eth1Data                            *ethpb.Eth1Data                         `ssz-gen:"true"`
	previousJustifiedCheckpoint         *ethpb.Checkpoint                       `ssz-gen:"true"`
	currentJustifiedCheckpoint          *ethpb.Checkpoint                       `ssz-gen:"true"`
	finalizedCheckpoint                 *ethpb.Checkpoint                       `ssz-gen:"true"`
	nextSyncCommittee                   *ethpb.SyncCommittee                    `ssz-gen:"true"`
	latestExecutionPayloadHeader        *enginev1.ExecutionPayloadHeader        `ssz-gen:"true"`
	latestExecutionPayloadHeaderCapella *enginev1.ExecutionPayloadHeaderCapella `ssz-gen:"true"`
	nextPartialWithdrawalValidatorIndex eth2types.ValidatorIndex                `ssz-gen:"true"`
	historicalRoots                     customtypes.HistoricalRoots             `ssz-gen:"true" ssz-size:"?,32" ssz-max:"16777216"`
	justificationBits                   bitfield.Bitvector4                     `ssz-gen:"true" ssz-size:"1"`
	dirtyFields                         map[nativetypes.FieldIndex]bool
	sharedFieldReferences               map[nativetypes.FieldIndex]*stateutil.Reference
	dirtyIndices                        map[nativetypes.FieldIndex][]uint64
	stateFieldLeaves                    map[nativetypes.FieldIndex]*fieldtrie.FieldTrie
	rebuildTrie                         map[nativetypes.FieldIndex]bool
	merkleLayers                        [][][]byte
}
