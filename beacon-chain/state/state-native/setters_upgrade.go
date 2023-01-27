package state_native

import (
	"runtime"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/fieldtrie"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	nativetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// UpgradeToCapella upgrades the beacon state to capella.
func (b *BeaconState) UpgradeToCapella() (state.BeaconState, error) {
	if b.version != version.Bellatrix {
		return nil, errNotSupported("UpgradeToCapella", b.version)
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	fieldCount := params.BeaconConfig().BeaconStateCapellaFieldCount
	epoch := slots.ToEpoch(b.slot)

	dst := &BeaconState{
		version: version.Capella,

		// Primitive nativetypes, safe to copy.
		genesisTime:                  b.genesisTime,
		slot:                         b.slot,
		eth1DepositIndex:             b.eth1DepositIndex,
		nextWithdrawalIndex:          0,
		nextWithdrawalValidatorIndex: 0,

		// Large arrays, infrequently changed, constant size.
		blockRoots:                b.blockRoots,
		stateRoots:                b.stateRoots,
		randaoMixes:               b.randaoMixes,
		previousEpochAttestations: b.previousEpochAttestations,
		currentEpochAttestations:  b.currentEpochAttestations,
		eth1DataVotes:             b.eth1DataVotes,
		slashings:                 b.slashings,

		// Large arrays, increases over time.
		balances:                   b.balances,
		historicalRoots:            b.historicalRoots,
		historicalSummaries:        b.historicalSummaries,
		validators:                 b.validators,
		previousEpochParticipation: b.previousEpochParticipation,
		currentEpochParticipation:  b.currentEpochParticipation,
		inactivityScores:           b.inactivityScores,

		// Everything else, too small to be concerned about, constant size.
		genesisValidatorsRoot: b.genesisValidatorsRoot,
		justificationBits:     b.justificationBitsVal(),
		fork: &ethpb.Fork{
			PreviousVersion: b.fork.CurrentVersion,
			CurrentVersion:  params.BeaconConfig().CapellaForkVersion,
			Epoch:           epoch,
		},
		latestBlockHeader:           b.latestBlockHeaderVal(),
		eth1Data:                    b.eth1DataVal(),
		previousJustifiedCheckpoint: b.previousJustifiedCheckpointVal(),
		currentJustifiedCheckpoint:  b.currentJustifiedCheckpointVal(),
		finalizedCheckpoint:         b.finalizedCheckpointVal(),
		currentSyncCommittee:        b.currentSyncCommitteeVal(),
		nextSyncCommittee:           b.nextSyncCommitteeVal(),
		latestExecutionPayloadHeaderCapella: &enginev1.ExecutionPayloadHeaderCapella{
			ParentHash:       b.latestExecutionPayloadHeaderVal().ParentHash(),
			FeeRecipient:     b.latestExecutionPayloadHeaderVal().FeeRecipient(),
			StateRoot:        b.latestExecutionPayloadHeaderVal().StateRoot(),
			ReceiptsRoot:     b.latestExecutionPayloadHeaderVal().ReceiptsRoot(),
			LogsBloom:        b.latestExecutionPayloadHeaderVal().LogsBloom(),
			PrevRandao:       b.latestExecutionPayloadHeaderVal().PrevRandao(),
			BlockNumber:      b.latestExecutionPayloadHeaderVal().BlockNumber(),
			GasLimit:         b.latestExecutionPayloadHeaderVal().GasLimit(),
			GasUsed:          b.latestExecutionPayloadHeaderVal().GasUsed(),
			Timestamp:        b.latestExecutionPayloadHeaderVal().Timestamp(),
			ExtraData:        b.latestExecutionPayloadHeaderVal().ExtraData(),
			BaseFeePerGas:    b.latestExecutionPayloadHeaderVal().BaseFeePerGas(),
			BlockHash:        b.latestExecutionPayloadHeaderVal().BlockHash(),
			TransactionsRoot: b.latestExecutionPayloadHeaderVal().TxRoot(),
			WithdrawalsRoot:  make([]byte, 32),
		},

		dirtyFields:      make(map[nativetypes.FieldIndex]bool, fieldCount),
		dirtyIndices:     make(map[nativetypes.FieldIndex][]uint64, fieldCount),
		rebuildTrie:      make(map[nativetypes.FieldIndex]bool, fieldCount),
		stateFieldLeaves: make(map[nativetypes.FieldIndex]*fieldtrie.FieldTrie, fieldCount),

		// Share the reference to validator index map.
		valMapHandler: b.valMapHandler,
	}

	dst.sharedFieldReferences = make(map[nativetypes.FieldIndex]*stateutil.Reference, capellaSharedFieldRefCount)

	for field, ref := range b.sharedFieldReferences {
		ref.AddRef()
		dst.sharedFieldReferences[field] = ref
	}

	// Increment ref for validator map
	b.valMapHandler.AddRef()

	for i := range b.dirtyFields {
		dst.dirtyFields[i] = true
	}

	for i := range b.dirtyIndices {
		indices := make([]uint64, len(b.dirtyIndices[i]))
		copy(indices, b.dirtyIndices[i])
		dst.dirtyIndices[i] = indices
	}

	for i := range b.rebuildTrie {
		dst.rebuildTrie[i] = true
	}

	for fldIdx, fieldTrie := range b.stateFieldLeaves {
		dst.stateFieldLeaves[fldIdx] = fieldTrie
		if fieldTrie.FieldReference() != nil {
			fieldTrie.Lock()
			fieldTrie.FieldReference().AddRef()
			fieldTrie.Unlock()
		}
	}

	if b.merkleLayers != nil {
		dst.merkleLayers = make([][][]byte, len(b.merkleLayers))
		for i, layer := range b.merkleLayers {
			dst.merkleLayers[i] = make([][]byte, len(layer))
			for j, content := range layer {
				dst.merkleLayers[i][j] = make([]byte, len(content))
				copy(dst.merkleLayers[i][j], content)
			}
		}
	}
	for _, f := range []types.FieldIndex{types.LatestExecutionPayloadHeaderCapella, types.HistoricalSummaries} {
		b.dirtyFields[f] = true
		b.rebuildTrie[f] = true
		b.dirtyIndices[f] = []uint64{}
		trie, err := fieldtrie.NewFieldTrie(f, types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
		}
		b.stateFieldLeaves[f] = trie
	}

	state.StateCount.Inc()
	// Finalizer runs when dst is being destroyed in garbage collection.
	runtime.SetFinalizer(dst, finalizerCleanup)
	return dst, nil
}
