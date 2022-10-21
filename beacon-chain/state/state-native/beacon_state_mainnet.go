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
	previousJustifiedCheckpoint         *ethpb.Checkpoint `ssz-gen:"true"`
	sharedFieldReferences               map[nativetypes.FieldIndex]*stateutil.Reference
	valMapHandler                       *stateutil.ValidatorMapHandler
	rebuildTrie                         map[nativetypes.FieldIndex]bool
	fork                                *ethpb.Fork              `ssz-gen:"true"`
	latestBlockHeader                   *ethpb.BeaconBlockHeader `ssz-gen:"true"`
	blockRoots                          *customtypes.BlockRoots  `ssz-gen:"true" ssz-size:"8192,32"`
	stateRoots                          *customtypes.StateRoots  `ssz-gen:"true" ssz-size:"8192,32"`
	stateFieldLeaves                    map[nativetypes.FieldIndex]*fieldtrie.FieldTrie
	eth1Data                            *ethpb.Eth1Data `ssz-gen:"true"`
	dirtyIndices                        map[nativetypes.FieldIndex][]uint64
	dirtyFields                         map[nativetypes.FieldIndex]bool
	latestExecutionPayloadHeaderCapella *enginev1.ExecutionPayloadHeaderCapella `ssz-gen:"true"`
	latestExecutionPayloadHeader        *enginev1.ExecutionPayloadHeader        `ssz-gen:"true"`
	randaoMixes                         *customtypes.RandaoMixes                `ssz-gen:"true" ssz-size:"65536,32"`
	nextSyncCommittee                   *ethpb.SyncCommittee                    `ssz-gen:"true"`
	currentSyncCommittee                *ethpb.SyncCommittee                    `ssz-gen:"true"`
	finalizedCheckpoint                 *ethpb.Checkpoint                       `ssz-gen:"true"`
	currentJustifiedCheckpoint          *ethpb.Checkpoint                       `ssz-gen:"true"`
	justificationBits                   bitfield.Bitvector4                     `ssz-gen:"true" ssz-size:"1"`
	historicalRoots                     customtypes.HistoricalRoots             `ssz-gen:"true" ssz-size:"?,32" ssz-max:"16777216"`
	currentEpochParticipation           []byte                                  `ssz-gen:"true" ssz-max:"1099511627776"`
	previousEpochParticipation          []byte                                  `ssz-gen:"true" ssz-max:"1099511627776"`
	currentEpochAttestations            []*ethpb.PendingAttestation             `ssz-gen:"true" ssz-max:"4096"`
	inactivityScores                    []uint64                                `ssz-gen:"true" ssz-max:"1099511627776"`
	previousEpochAttestations           []*ethpb.PendingAttestation             `ssz-gen:"true" ssz-max:"4096"`
	slashings                           []uint64                                `ssz-gen:"true" ssz-size:"8192"`
	balances                            []uint64                                `ssz-gen:"true" ssz-max:"1099511627776"`
	validators                          []*ethpb.Validator                      `ssz-gen:"true" ssz-max:"1099511627776"`
	withdrawalQueue                     []*enginev1.Withdrawal                  `ssz-gen:"true" ssz-max:"1099511627776"`
	merkleLayers                        [][][]byte
	eth1DataVotes                       []*ethpb.Eth1Data        `ssz-gen:"true" ssz-max:"2048"`
	eth1DepositIndex                    uint64                   `ssz-gen:"true"`
	nextPartialWithdrawalValidatorIndex eth2types.ValidatorIndex `ssz-gen:"true"`
	version                             int
	slot                                eth2types.Slot `ssz-gen:"true"`
	nextWithdrawalIndex                 uint64         `ssz-gen:"true"`
	genesisTime                         uint64         `ssz-gen:"true"`
	lock                                sync.RWMutex
	genesisValidatorsRoot               [32]byte `ssz-gen:"true" ssz-size:"32"`
}
