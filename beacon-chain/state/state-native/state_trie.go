package state_native

import (
	"context"
	"encoding/binary"
	"runtime"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/fieldtrie"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"
	nativetypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/slice"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

var phase0Fields = []nativetypes.FieldIndex{
	nativetypes.GenesisTime,
	nativetypes.GenesisValidatorsRoot,
	nativetypes.Slot,
	nativetypes.Fork,
	nativetypes.LatestBlockHeader,
	nativetypes.BlockRoots,
	nativetypes.StateRoots,
	nativetypes.HistoricalRoots,
	nativetypes.Eth1Data,
	nativetypes.Eth1DataVotes,
	nativetypes.Eth1DepositIndex,
	nativetypes.Validators,
	nativetypes.Balances,
	nativetypes.RandaoMixes,
	nativetypes.Slashings,
	nativetypes.PreviousEpochAttestations,
	nativetypes.CurrentEpochAttestations,
	nativetypes.JustificationBits,
	nativetypes.PreviousJustifiedCheckpoint,
	nativetypes.CurrentJustifiedCheckpoint,
	nativetypes.FinalizedCheckpoint,
}

var altairFields = []nativetypes.FieldIndex{
	nativetypes.GenesisTime,
	nativetypes.GenesisValidatorsRoot,
	nativetypes.Slot,
	nativetypes.Fork,
	nativetypes.LatestBlockHeader,
	nativetypes.BlockRoots,
	nativetypes.StateRoots,
	nativetypes.HistoricalRoots,
	nativetypes.Eth1Data,
	nativetypes.Eth1DataVotes,
	nativetypes.Eth1DepositIndex,
	nativetypes.Validators,
	nativetypes.Balances,
	nativetypes.RandaoMixes,
	nativetypes.Slashings,
	nativetypes.PreviousEpochParticipationBits,
	nativetypes.CurrentEpochParticipationBits,
	nativetypes.JustificationBits,
	nativetypes.PreviousJustifiedCheckpoint,
	nativetypes.CurrentJustifiedCheckpoint,
	nativetypes.FinalizedCheckpoint,
	nativetypes.InactivityScores,
	nativetypes.CurrentSyncCommittee,
	nativetypes.NextSyncCommittee,
}

var bellatrixFields = []nativetypes.FieldIndex{
	nativetypes.GenesisTime,
	nativetypes.GenesisValidatorsRoot,
	nativetypes.Slot,
	nativetypes.Fork,
	nativetypes.LatestBlockHeader,
	nativetypes.BlockRoots,
	nativetypes.StateRoots,
	nativetypes.HistoricalRoots,
	nativetypes.Eth1Data,
	nativetypes.Eth1DataVotes,
	nativetypes.Eth1DepositIndex,
	nativetypes.Validators,
	nativetypes.Balances,
	nativetypes.RandaoMixes,
	nativetypes.Slashings,
	nativetypes.PreviousEpochParticipationBits,
	nativetypes.CurrentEpochParticipationBits,
	nativetypes.JustificationBits,
	nativetypes.PreviousJustifiedCheckpoint,
	nativetypes.CurrentJustifiedCheckpoint,
	nativetypes.FinalizedCheckpoint,
	nativetypes.InactivityScores,
	nativetypes.CurrentSyncCommittee,
	nativetypes.NextSyncCommittee,
	nativetypes.LatestExecutionPayloadHeader,
}

// InitializeFromProtoPhase0 the beacon state from a protobuf representation.
func InitializeFromProtoPhase0(st *ethpb.BeaconState) (state.BeaconState, error) {
	return InitializeFromProtoUnsafePhase0(proto.Clone(st).(*ethpb.BeaconState))
}

// InitializeFromProtoAltair the beacon state from a protobuf representation.
func InitializeFromProtoAltair(st *ethpb.BeaconStateAltair) (state.BeaconState, error) {
	return InitializeFromProtoUnsafeAltair(proto.Clone(st).(*ethpb.BeaconStateAltair))
}

// InitializeFromProtoBellatrix the beacon state from a protobuf representation.
func InitializeFromProtoBellatrix(st *ethpb.BeaconStateBellatrix) (state.BeaconState, error) {
	return InitializeFromProtoUnsafeBellatrix(proto.Clone(st).(*ethpb.BeaconStateBellatrix))
}

// InitializeFromProtoUnsafePhase0 directly uses the beacon state protobuf fields
// and sets them as fields of the BeaconState type.
func InitializeFromProtoUnsafePhase0(st *ethpb.BeaconState) (state.BeaconState, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	var bRoots customtypes.BlockRoots
	for i, r := range st.BlockRoots {
		copy(bRoots[i][:], r)
	}
	var sRoots customtypes.StateRoots
	for i, r := range st.StateRoots {
		copy(sRoots[i][:], r)
	}
	hRoots := customtypes.HistoricalRoots(make([][32]byte, len(st.HistoricalRoots)))
	for i, r := range st.HistoricalRoots {
		copy(hRoots[i][:], r)
	}
	var mixes customtypes.RandaoMixes
	for i, m := range st.RandaoMixes {
		copy(mixes[i][:], m)
	}

	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	b := &BeaconState{
		version:                     version.Phase0,
		genesisTime:                 st.GenesisTime,
		genesisValidatorsRoot:       bytesutil.ToBytes32(st.GenesisValidatorsRoot),
		slot:                        st.Slot,
		fork:                        st.Fork,
		latestBlockHeader:           st.LatestBlockHeader,
		blockRoots:                  &bRoots,
		stateRoots:                  &sRoots,
		historicalRoots:             hRoots,
		eth1Data:                    st.Eth1Data,
		eth1DataVotes:               st.Eth1DataVotes,
		eth1DepositIndex:            st.Eth1DepositIndex,
		validators:                  st.Validators,
		balances:                    st.Balances,
		randaoMixes:                 &mixes,
		slashings:                   st.Slashings,
		previousEpochAttestations:   st.PreviousEpochAttestations,
		currentEpochAttestations:    st.CurrentEpochAttestations,
		justificationBits:           st.JustificationBits,
		previousJustifiedCheckpoint: st.PreviousJustifiedCheckpoint,
		currentJustifiedCheckpoint:  st.CurrentJustifiedCheckpoint,
		finalizedCheckpoint:         st.FinalizedCheckpoint,

		dirtyFields:           make(map[nativetypes.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[nativetypes.FieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[nativetypes.FieldIndex]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[nativetypes.FieldIndex]*stateutil.Reference, 10),
		rebuildTrie:           make(map[nativetypes.FieldIndex]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	b.populateFieldIndexes(phase0Fields)

	for _, f := range phase0Fields {
		b.dirtyFields[f] = true
		b.rebuildTrie[f] = true
		b.dirtyIndices[f] = []uint64{}
		trie, err := fieldtrie.NewFieldTrie(f, types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
		}
		b.stateFieldLeaves[f] = trie
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[nativetypes.BlockRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.StateRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.HistoricalRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Eth1DataVotes] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Validators] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Balances] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.RandaoMixes] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Slashings] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.PreviousEpochAttestations] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.CurrentEpochAttestations] = stateutil.NewRef(1)

	state.StateCount.Inc()
	return b, nil
}

// InitializeFromProtoUnsafeAltair directly uses the beacon state protobuf fields
// and sets them as fields of the BeaconState type.
func InitializeFromProtoUnsafeAltair(st *ethpb.BeaconStateAltair) (state.BeaconStateAltair, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	var bRoots customtypes.BlockRoots
	for i, r := range st.BlockRoots {
		bRoots[i] = bytesutil.ToBytes32(r)
	}
	var sRoots customtypes.StateRoots
	for i, r := range st.StateRoots {
		sRoots[i] = bytesutil.ToBytes32(r)
	}
	hRoots := customtypes.HistoricalRoots(make([][32]byte, len(st.HistoricalRoots)))
	for i, r := range st.HistoricalRoots {
		hRoots[i] = bytesutil.ToBytes32(r)
	}
	var mixes customtypes.RandaoMixes
	for i, m := range st.RandaoMixes {
		mixes[i] = bytesutil.ToBytes32(m)
	}

	fieldCount := params.BeaconConfig().BeaconStateAltairFieldCount
	b := &BeaconState{
		version:                     version.Altair,
		genesisTime:                 st.GenesisTime,
		genesisValidatorsRoot:       bytesutil.ToBytes32(st.GenesisValidatorsRoot),
		slot:                        st.Slot,
		fork:                        st.Fork,
		latestBlockHeader:           st.LatestBlockHeader,
		blockRoots:                  &bRoots,
		stateRoots:                  &sRoots,
		historicalRoots:             hRoots,
		eth1Data:                    st.Eth1Data,
		eth1DataVotes:               st.Eth1DataVotes,
		eth1DepositIndex:            st.Eth1DepositIndex,
		validators:                  st.Validators,
		balances:                    st.Balances,
		randaoMixes:                 &mixes,
		slashings:                   st.Slashings,
		previousEpochParticipation:  st.PreviousEpochParticipation,
		currentEpochParticipation:   st.CurrentEpochParticipation,
		justificationBits:           st.JustificationBits,
		previousJustifiedCheckpoint: st.PreviousJustifiedCheckpoint,
		currentJustifiedCheckpoint:  st.CurrentJustifiedCheckpoint,
		finalizedCheckpoint:         st.FinalizedCheckpoint,
		inactivityScores:            st.InactivityScores,
		currentSyncCommittee:        st.CurrentSyncCommittee,
		nextSyncCommittee:           st.NextSyncCommittee,

		dirtyFields:           make(map[nativetypes.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[nativetypes.FieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[nativetypes.FieldIndex]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[nativetypes.FieldIndex]*stateutil.Reference, 11),
		rebuildTrie:           make(map[nativetypes.FieldIndex]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	b.populateFieldIndexes(altairFields)

	for _, f := range altairFields {
		b.dirtyFields[f] = true
		b.rebuildTrie[f] = true
		b.dirtyIndices[f] = []uint64{}
		trie, err := fieldtrie.NewFieldTrie(f, types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
		}
		b.stateFieldLeaves[f] = trie
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[nativetypes.BlockRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.StateRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.HistoricalRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Eth1DataVotes] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Validators] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Balances] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.RandaoMixes] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Slashings] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits] = stateutil.NewRef(1) // New in Altair.
	b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits] = stateutil.NewRef(1)  // New in Altair.
	b.sharedFieldReferences[nativetypes.InactivityScores] = stateutil.NewRef(1)               // New in Altair.

	state.StateCount.Inc()
	return b, nil
}

// InitializeFromProtoUnsafeBellatrix directly uses the beacon state protobuf fields
// and sets them as fields of the BeaconState type.
func InitializeFromProtoUnsafeBellatrix(st *ethpb.BeaconStateBellatrix) (state.BeaconStateBellatrix, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	var bRoots customtypes.BlockRoots
	for i, r := range st.BlockRoots {
		bRoots[i] = bytesutil.ToBytes32(r)
	}
	var sRoots customtypes.StateRoots
	for i, r := range st.StateRoots {
		sRoots[i] = bytesutil.ToBytes32(r)
	}
	hRoots := customtypes.HistoricalRoots(make([][32]byte, len(st.HistoricalRoots)))
	for i, r := range st.HistoricalRoots {
		hRoots[i] = bytesutil.ToBytes32(r)
	}
	var mixes customtypes.RandaoMixes
	for i, m := range st.RandaoMixes {
		mixes[i] = bytesutil.ToBytes32(m)
	}

	fieldCount := params.BeaconConfig().BeaconStateBellatrixFieldCount
	b := &BeaconState{
		version:                      version.Bellatrix,
		genesisTime:                  st.GenesisTime,
		genesisValidatorsRoot:        bytesutil.ToBytes32(st.GenesisValidatorsRoot),
		slot:                         st.Slot,
		fork:                         st.Fork,
		latestBlockHeader:            st.LatestBlockHeader,
		blockRoots:                   &bRoots,
		stateRoots:                   &sRoots,
		historicalRoots:              hRoots,
		eth1Data:                     st.Eth1Data,
		eth1DataVotes:                st.Eth1DataVotes,
		eth1DepositIndex:             st.Eth1DepositIndex,
		validators:                   st.Validators,
		balances:                     st.Balances,
		randaoMixes:                  &mixes,
		slashings:                    st.Slashings,
		previousEpochParticipation:   st.PreviousEpochParticipation,
		currentEpochParticipation:    st.CurrentEpochParticipation,
		justificationBits:            st.JustificationBits,
		previousJustifiedCheckpoint:  st.PreviousJustifiedCheckpoint,
		currentJustifiedCheckpoint:   st.CurrentJustifiedCheckpoint,
		finalizedCheckpoint:          st.FinalizedCheckpoint,
		inactivityScores:             st.InactivityScores,
		currentSyncCommittee:         st.CurrentSyncCommittee,
		nextSyncCommittee:            st.NextSyncCommittee,
		latestExecutionPayloadHeader: st.LatestExecutionPayloadHeader,

		dirtyFields:           make(map[nativetypes.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[nativetypes.FieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[nativetypes.FieldIndex]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[nativetypes.FieldIndex]*stateutil.Reference, 11),
		rebuildTrie:           make(map[nativetypes.FieldIndex]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	b.populateFieldIndexes(bellatrixFields)

	for _, f := range bellatrixFields {
		b.dirtyFields[f] = true
		b.rebuildTrie[f] = true
		b.dirtyIndices[f] = []uint64{}
		trie, err := fieldtrie.NewFieldTrie(f, types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
		}
		b.stateFieldLeaves[f] = trie
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[nativetypes.BlockRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.StateRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.HistoricalRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Eth1DataVotes] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Validators] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Balances] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.RandaoMixes] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.Slashings] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.PreviousEpochParticipationBits] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.CurrentEpochParticipationBits] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.InactivityScores] = stateutil.NewRef(1)
	b.sharedFieldReferences[nativetypes.LatestExecutionPayloadHeader] = stateutil.NewRef(1) // New in Bellatrix.

	state.StateCount.Inc()
	return b, nil
}

// Copy returns a deep copy of the beacon state.
func (b *BeaconState) Copy() state.BeaconState {
	b.lock.RLock()
	defer b.lock.RUnlock()

	var fieldCount int
	switch b.version {
	case version.Phase0:
		fieldCount = params.BeaconConfig().BeaconStateFieldCount
	case version.Altair:
		fieldCount = params.BeaconConfig().BeaconStateAltairFieldCount
	case version.Bellatrix:
		fieldCount = params.BeaconConfig().BeaconStateBellatrixFieldCount
	}

	dst := &BeaconState{
		version: b.version,

		// Primitive nativetypes, safe to copy.
		genesisTime:      b.genesisTime,
		slot:             b.slot,
		eth1DepositIndex: b.eth1DepositIndex,

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
		validators:                 b.validators,
		previousEpochParticipation: b.previousEpochParticipation,
		currentEpochParticipation:  b.currentEpochParticipation,
		inactivityScores:           b.inactivityScores,

		// Everything else, too small to be concerned about, constant size.
		genesisValidatorsRoot:        b.genesisValidatorsRoot,
		justificationBits:            b.justificationBitsVal(),
		fork:                         b.forkVal(),
		latestBlockHeader:            b.latestBlockHeaderVal(),
		eth1Data:                     b.eth1DataVal(),
		previousJustifiedCheckpoint:  b.previousJustifiedCheckpointVal(),
		currentJustifiedCheckpoint:   b.currentJustifiedCheckpointVal(),
		finalizedCheckpoint:          b.finalizedCheckpointVal(),
		currentSyncCommittee:         b.currentSyncCommitteeVal(),
		nextSyncCommittee:            b.nextSyncCommitteeVal(),
		latestExecutionPayloadHeader: b.latestExecutionPayloadHeaderVal(),

		dirtyFields:      make(map[nativetypes.FieldIndex]bool, fieldCount),
		dirtyIndices:     make(map[nativetypes.FieldIndex][]uint64, fieldCount),
		rebuildTrie:      make(map[nativetypes.FieldIndex]bool, fieldCount),
		stateFieldLeaves: make(map[nativetypes.FieldIndex]*fieldtrie.FieldTrie, fieldCount),

		// Share the reference to validator index map.
		valMapHandler: b.valMapHandler,
	}

	switch b.version {
	case version.Phase0:
		dst.sharedFieldReferences = make(map[nativetypes.FieldIndex]*stateutil.Reference, 10)
		dst.populateFieldIndexes(phase0Fields)
	case version.Altair:
		dst.sharedFieldReferences = make(map[nativetypes.FieldIndex]*stateutil.Reference, 11)
		dst.populateFieldIndexes(altairFields)
	case version.Bellatrix:
		dst.sharedFieldReferences = make(map[nativetypes.FieldIndex]*stateutil.Reference, 11)
		dst.populateFieldIndexes(bellatrixFields)
	}

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

	state.StateCount.Inc()
	// Finalizer runs when dst is being destroyed in garbage collection.
	runtime.SetFinalizer(dst, func(b *BeaconState) {
		for field, v := range b.sharedFieldReferences {
			v.MinusRef()
			if b.stateFieldLeaves[field].FieldReference() != nil {
				b.stateFieldLeaves[field].FieldReference().MinusRef()
			}

		}
		for i := range b.dirtyFields {
			delete(b.dirtyFields, i)
		}
		for i := range b.rebuildTrie {
			delete(b.rebuildTrie, i)
		}
		for i := range b.dirtyIndices {
			delete(b.dirtyIndices, i)
		}
		for i := range b.sharedFieldReferences {
			delete(b.sharedFieldReferences, i)
		}
		for i := range b.stateFieldLeaves {
			delete(b.stateFieldLeaves, i)
		}
		state.StateCount.Sub(1)
	})
	return dst
}

// HashTreeRoot of the beacon state retrieves the Merkle root of the trie
// representation of the beacon state based on the Ethereum Simple Serialize specification.
func (b *BeaconState) HashTreeRoot(ctx context.Context) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconState.HashTreeRoot")
	defer span.End()

	b.lock.Lock()
	defer b.lock.Unlock()
	if err := b.initializeMerkleLayers(ctx); err != nil {
		return [32]byte{}, err
	}
	if err := b.recomputeDirtyFields(ctx); err != nil {
		return [32]byte{}, err
	}
	return bytesutil.ToBytes32(b.merkleLayers[len(b.merkleLayers)-1][0]), nil
}

// Initializes the Merkle layers for the beacon state if they are empty.
// WARNING: Caller must acquire the mutex before using.
func (b *BeaconState) initializeMerkleLayers(ctx context.Context) error {
	if len(b.merkleLayers) > 0 {
		return nil
	}
	fieldRoots, err := ComputeFieldRootsWithHasher(ctx, b)
	if err != nil {
		return err
	}
	layers := stateutil.Merkleize(fieldRoots)
	b.merkleLayers = layers
	switch b.version {
	case version.Phase0:
		b.dirtyFields = make(map[nativetypes.FieldIndex]bool, params.BeaconConfig().BeaconStateFieldCount)
	case version.Altair:
		b.dirtyFields = make(map[nativetypes.FieldIndex]bool, params.BeaconConfig().BeaconStateAltairFieldCount)
	case version.Bellatrix:
		b.dirtyFields = make(map[nativetypes.FieldIndex]bool, params.BeaconConfig().BeaconStateBellatrixFieldCount)
	}

	return nil
}

// Recomputes the Merkle layers for the dirty fields in the state.
// WARNING: Caller must acquire the mutex before using.
func (b *BeaconState) recomputeDirtyFields(ctx context.Context) error {
	for field := range b.dirtyFields {
		root, err := b.rootSelector(ctx, field)
		if err != nil {
			return err
		}
		idx := b.fieldIndexesRev[field]
		b.merkleLayers[0][idx] = root[:]
		b.recomputeRoot(idx)
		delete(b.dirtyFields, field)
	}
	return nil
}

// FieldReferencesCount returns the reference count held by each field. This
// also includes the field trie held by each field.
func (b *BeaconState) FieldReferencesCount() map[string]uint64 {
	refMap := make(map[string]uint64)
	b.lock.RLock()
	defer b.lock.RUnlock()
	for i, f := range b.sharedFieldReferences {
		refMap[i.String(b.version)] = uint64(f.Refs())
	}
	for i, f := range b.stateFieldLeaves {
		numOfRefs := uint64(f.FieldReference().Refs())
		f.RLock()
		if !f.Empty() {
			refMap[i.String(b.version)+"_trie"] = numOfRefs
		}
		f.RUnlock()
	}
	return refMap
}

// IsNil checks if the state and the underlying proto
// object are nil.
func (b *BeaconState) IsNil() bool {
	return b == nil
}

func (b *BeaconState) rootSelector(ctx context.Context, field nativetypes.FieldIndex) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "beaconState.rootSelector")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("field", field.String(b.version)))

	hasher := hash.CustomSHA256Hasher()
	switch field {
	case nativetypes.GenesisTime:
		return ssz.Uint64Root(b.genesisTime), nil
	case nativetypes.GenesisValidatorsRoot:
		return b.genesisValidatorsRoot, nil
	case nativetypes.Slot:
		return ssz.Uint64Root(uint64(b.slot)), nil
	case nativetypes.Eth1DepositIndex:
		return ssz.Uint64Root(b.eth1DepositIndex), nil
	case nativetypes.Fork:
		return ssz.ForkRoot(b.fork)
	case nativetypes.LatestBlockHeader:
		return stateutil.BlockHeaderRoot(b.latestBlockHeader)
	case nativetypes.BlockRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.blockRoots, fieldparams.BlockRootsLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.blockRoots)
	case nativetypes.StateRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.stateRoots, fieldparams.StateRootsLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.stateRoots)
	case nativetypes.HistoricalRoots:
		hRoots := make([][]byte, len(b.historicalRoots))
		for i := range hRoots {
			hRoots[i] = b.historicalRoots[i][:]
		}
		return ssz.ByteArrayRootWithLimit(hRoots, fieldparams.HistoricalRootsLength)
	case nativetypes.Eth1Data:
		return stateutil.Eth1Root(hasher, b.eth1Data)
	case nativetypes.Eth1DataVotes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.eth1DataVotes,
				fieldparams.Eth1DataVotesLength,
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.eth1DataVotes)
	case nativetypes.Validators:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.validators, fieldparams.ValidatorRegistryLimit)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(11, b.validators)
	case nativetypes.Balances:
		if b.rebuildTrie[field] {
			maxBalCap := uint64(fieldparams.ValidatorRegistryLimit)
			elemSize := uint64(8)
			balLimit := (maxBalCap*elemSize + 31) / 32
			err := b.resetFieldTrie(field, b.balances, balLimit)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(12, b.balances)
	case nativetypes.RandaoMixes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.randaoMixes, fieldparams.RandaoMixesLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(13, b.randaoMixes)
	case nativetypes.Slashings:
		return ssz.SlashingsRoot(b.slashings)
	case nativetypes.PreviousEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.previousEpochAttestations,
				fieldparams.PreviousEpochAttestationsLength,
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.previousEpochAttestations)
	case nativetypes.CurrentEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.currentEpochAttestations,
				fieldparams.CurrentEpochAttestationsLength,
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.currentEpochAttestations)
	case nativetypes.PreviousEpochParticipationBits:
		return stateutil.ParticipationBitsRoot(b.previousEpochParticipation)
	case nativetypes.CurrentEpochParticipationBits:
		return stateutil.ParticipationBitsRoot(b.currentEpochParticipation)
	case nativetypes.JustificationBits:
		return bytesutil.ToBytes32(b.justificationBits), nil
	case nativetypes.PreviousJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.previousJustifiedCheckpoint)
	case nativetypes.CurrentJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.currentJustifiedCheckpoint)
	case nativetypes.FinalizedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.finalizedCheckpoint)
	case nativetypes.InactivityScores:
		return stateutil.Uint64ListRootWithRegistryLimit(b.inactivityScores)
	case nativetypes.CurrentSyncCommittee:
		return stateutil.SyncCommitteeRoot(b.currentSyncCommittee)
	case nativetypes.NextSyncCommittee:
		return stateutil.SyncCommitteeRoot(b.nextSyncCommittee)
	case nativetypes.LatestExecutionPayloadHeader:
		return b.latestExecutionPayloadHeader.HashTreeRoot()
	}
	return [32]byte{}, errors.New("invalid field index provided")
}

func (b *BeaconState) recomputeFieldTrie(index nativetypes.FieldIndex, elements interface{}) ([32]byte, error) {
	fTrie := b.stateFieldLeaves[index]
	// We can't lock the trie directly because the trie's variable gets reassigned,
	// and therefore we would call Unlock() on a different object.
	fTrieMutex := fTrie.RWMutex
	if fTrie.FieldReference().Refs() > 1 {
		fTrieMutex.Lock()
		fTrie.FieldReference().MinusRef()
		newTrie := fTrie.CopyTrie()
		b.stateFieldLeaves[index] = newTrie
		fTrie = newTrie
		fTrieMutex.Unlock()
	}
	// remove duplicate indexes
	b.dirtyIndices[index] = slice.SetUint64(b.dirtyIndices[index])
	// sort indexes again
	sort.Slice(b.dirtyIndices[index], func(i int, j int) bool {
		return b.dirtyIndices[index][i] < b.dirtyIndices[index][j]
	})
	root, err := fTrie.RecomputeTrie(b.dirtyIndices[index], elements)
	if err != nil {
		return [32]byte{}, err
	}
	b.dirtyIndices[index] = []uint64{}
	return root, nil
}

func (b *BeaconState) resetFieldTrie(index nativetypes.FieldIndex, elements interface{}, length uint64) error {
	fTrie, err := fieldtrie.NewFieldTrie(index, fieldMap[index], elements, length)
	if err != nil {
		return err
	}
	b.stateFieldLeaves[index] = fTrie
	b.dirtyIndices[index] = []uint64{}
	return nil
}

// ComputeFieldRootsWithHasher hashes the provided state and returns its respective field roots.
func ComputeFieldRootsWithHasher(ctx context.Context, state *BeaconState) ([][]byte, error) {
	_, span := trace.StartSpan(ctx, "ComputeFieldRootsWithHasher")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	hasher := hash.CustomSHA256Hasher()
	var fieldRoots [][]byte
	switch state.version {
	case version.Phase0:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateFieldCount)
	case version.Altair:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateAltairFieldCount)
	case version.Bellatrix:
		fieldRoots = make([][]byte, params.BeaconConfig().BeaconStateBellatrixFieldCount)
	}

	fieldRootIx := 0

	// Genesis time root.
	genesisRoot := ssz.Uint64Root(state.genesisTime)
	fieldRoots[fieldRootIx] = genesisRoot[:]
	fieldRootIx++

	// Genesis validators root.
	r := [32]byte{}
	copy(r[:], state.genesisValidatorsRoot[:])
	fieldRoots[fieldRootIx] = r[:]
	fieldRootIx++

	// Slot root.
	slotRoot := ssz.Uint64Root(uint64(state.slot))
	fieldRoots[fieldRootIx] = slotRoot[:]
	fieldRootIx++

	// Fork data structure root.
	forkHashTreeRoot, err := ssz.ForkRoot(state.fork)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute fork merkleization")
	}
	fieldRoots[fieldRootIx] = forkHashTreeRoot[:]
	fieldRootIx++

	// BeaconBlockHeader data structure root.
	headerHashTreeRoot, err := stateutil.BlockHeaderRoot(state.latestBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block header merkleization")
	}
	fieldRoots[fieldRootIx] = headerHashTreeRoot[:]
	fieldRootIx++

	// BlockRoots array root.
	bRoots := make([][]byte, len(state.blockRoots))
	for i := range bRoots {
		bRoots[i] = state.blockRoots[i][:]
	}
	blockRootsRoot, err := stateutil.ArraysRoot(bRoots, fieldparams.BlockRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute block roots merkleization")
	}
	fieldRoots[fieldRootIx] = blockRootsRoot[:]
	fieldRootIx++

	// StateRoots array root.
	sRoots := make([][]byte, len(state.stateRoots))
	for i := range sRoots {
		sRoots[i] = state.stateRoots[i][:]
	}
	stateRootsRoot, err := stateutil.ArraysRoot(sRoots, fieldparams.StateRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute state roots merkleization")
	}
	fieldRoots[fieldRootIx] = stateRootsRoot[:]
	fieldRootIx++

	// HistoricalRoots slice root.
	hRoots := make([][]byte, len(state.historicalRoots))
	for i := range hRoots {
		hRoots[i] = state.historicalRoots[i][:]
	}
	historicalRootsRt, err := ssz.ByteArrayRootWithLimit(hRoots, fieldparams.HistoricalRootsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute historical roots merkleization")
	}
	fieldRoots[fieldRootIx] = historicalRootsRt[:]
	fieldRootIx++

	// Eth1Data data structure root.
	eth1HashTreeRoot, err := stateutil.Eth1Root(hasher, state.eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[fieldRootIx] = eth1HashTreeRoot[:]
	fieldRootIx++

	// Eth1DataVotes slice root.
	eth1VotesRoot, err := stateutil.Eth1DataVotesRoot(state.eth1DataVotes)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data votes merkleization")
	}
	fieldRoots[fieldRootIx] = eth1VotesRoot[:]
	fieldRootIx++

	// Eth1DepositIndex root.
	eth1DepositIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(eth1DepositIndexBuf, state.eth1DepositIndex)
	eth1DepositBuf := bytesutil.ToBytes32(eth1DepositIndexBuf)
	fieldRoots[fieldRootIx] = eth1DepositBuf[:]
	fieldRootIx++

	// Validators slice root.
	validatorsRoot, err := stateutil.ValidatorRegistryRoot(state.validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	fieldRoots[fieldRootIx] = validatorsRoot[:]
	fieldRootIx++

	// Balances slice root.
	balancesRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.balances)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute validator balances merkleization")
	}
	fieldRoots[fieldRootIx] = balancesRoot[:]
	fieldRootIx++

	// RandaoMixes array root.
	mixes := make([][]byte, len(state.randaoMixes))
	for i := range mixes {
		mixes[i] = state.randaoMixes[i][:]
	}
	randaoRootsRoot, err := stateutil.ArraysRoot(mixes, fieldparams.RandaoMixesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao roots merkleization")
	}
	fieldRoots[fieldRootIx] = randaoRootsRoot[:]
	fieldRootIx++

	// Slashings array root.
	slashingsRootsRoot, err := ssz.SlashingsRoot(state.slashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute slashings merkleization")
	}
	fieldRoots[fieldRootIx] = slashingsRootsRoot[:]
	fieldRootIx++

	if state.version == version.Phase0 {
		// PreviousEpochAttestations slice root.
		prevAttsRoot, err := stateutil.EpochAttestationsRoot(state.previousEpochAttestations)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch attestations merkleization")
		}
		fieldRoots[fieldRootIx] = prevAttsRoot[:]
		fieldRootIx++

		// CurrentEpochAttestations slice root.
		currAttsRoot, err := stateutil.EpochAttestationsRoot(state.currentEpochAttestations)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch attestations merkleization")
		}
		fieldRoots[fieldRootIx] = currAttsRoot[:]
		fieldRootIx++
	}

	if state.version == version.Altair || state.version == version.Bellatrix {
		// PreviousEpochParticipation slice root.
		prevParticipationRoot, err := stateutil.ParticipationBitsRoot(state.previousEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute previous epoch participation merkleization")
		}
		fieldRoots[fieldRootIx] = prevParticipationRoot[:]
		fieldRootIx++

		// CurrentEpochParticipation slice root.
		currParticipationRoot, err := stateutil.ParticipationBitsRoot(state.currentEpochParticipation)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute current epoch participation merkleization")
		}
		fieldRoots[fieldRootIx] = currParticipationRoot[:]
		fieldRootIx++
	}

	// JustificationBits root.
	justifiedBitsRoot := bytesutil.ToBytes32(state.justificationBits)
	fieldRoots[fieldRootIx] = justifiedBitsRoot[:]
	fieldRootIx++

	// PreviousJustifiedCheckpoint data structure root.
	prevCheckRoot, err := ssz.CheckpointRoot(hasher, state.previousJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute previous justified checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = prevCheckRoot[:]
	fieldRootIx++

	// CurrentJustifiedCheckpoint data structure root.
	currJustRoot, err := ssz.CheckpointRoot(hasher, state.currentJustifiedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute current justified checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = currJustRoot[:]
	fieldRootIx++

	// FinalizedCheckpoint data structure root.
	finalRoot, err := ssz.CheckpointRoot(hasher, state.finalizedCheckpoint)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute finalized checkpoint merkleization")
	}
	fieldRoots[fieldRootIx] = finalRoot[:]
	fieldRootIx++

	if state.version == version.Altair || state.version == version.Bellatrix {
		// Inactivity scores root.
		inactivityScoresRoot, err := stateutil.Uint64ListRootWithRegistryLimit(state.inactivityScores)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute inactivityScoreRoot")
		}
		fieldRoots[fieldRootIx] = inactivityScoresRoot[:]
		fieldRootIx++

		// Current sync committee root.
		currentSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(state.currentSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[fieldRootIx] = currentSyncCommitteeRoot[:]
		fieldRootIx++

		// Next sync committee root.
		nextSyncCommitteeRoot, err := stateutil.SyncCommitteeRoot(state.nextSyncCommittee)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute sync committee merkleization")
		}
		fieldRoots[fieldRootIx] = nextSyncCommitteeRoot[:]
		fieldRootIx++
	}

	if state.version == version.Bellatrix {
		// Execution payload root.
		executionPayloadRoot, err := state.latestExecutionPayloadHeader.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		fieldRoots[fieldRootIx] = executionPayloadRoot[:]
	}

	return fieldRoots, nil
}

func (b *BeaconState) populateFieldIndexes(fields []nativetypes.FieldIndex) {
	b.fieldIndexesRev = make(map[nativetypes.FieldIndex]int, len(fields))
	for i, f := range fields {
		b.fieldIndexesRev[f] = i
	}
}
