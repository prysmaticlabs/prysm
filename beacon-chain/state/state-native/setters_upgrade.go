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

	payloadHeader := b.latestExecutionPayloadHeaderVal()
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
		historicalSummaries:        make([]*ethpb.HistoricalSummary, 0),
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
			ParentHash:       payloadHeader.ParentHash,
			FeeRecipient:     payloadHeader.FeeRecipient,
			StateRoot:        payloadHeader.StateRoot,
			ReceiptsRoot:     payloadHeader.ReceiptsRoot,
			LogsBloom:        payloadHeader.LogsBloom,
			PrevRandao:       payloadHeader.PrevRandao,
			BlockNumber:      payloadHeader.BlockNumber,
			GasLimit:         payloadHeader.GasLimit,
			GasUsed:          payloadHeader.GasUsed,
			Timestamp:        payloadHeader.Timestamp,
			ExtraData:        payloadHeader.ExtraData,
			BaseFeePerGas:    payloadHeader.BaseFeePerGas,
			BlockHash:        payloadHeader.BlockHash,
			TransactionsRoot: payloadHeader.TransactionsRoot,
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
