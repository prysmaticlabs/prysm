package v3

import (
	"context"
	"runtime"
	"sort"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/fieldtrie"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/slice"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

var (
	stateCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacon_state_merge_count",
		Help: "Count the number of active beacon state objects.",
	})
)

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *ethpb.BeaconStateMerge) (*BeaconState, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	st, ok := proto.Clone(st).(*ethpb.BeaconStateMerge)
	if !ok {
		return nil, errors.New("could not clone proto beacon state")
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

	fieldCount := params.BeaconConfig().BeaconStateMergeFieldCount
	b := &BeaconState{
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

		dirtyFields:           make(map[types.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[types.FieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[types.FieldIndex]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[types.FieldIndex]*stateutil.Reference, 11),
		rebuildTrie:           make(map[types.FieldIndex]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	var err error
	for i := 0; i < fieldCount; i++ {
		b.dirtyFields[types.FieldIndex(i)] = true
		b.rebuildTrie[types.FieldIndex(i)] = true
		b.dirtyIndices[types.FieldIndex(i)] = []uint64{}
		b.stateFieldLeaves[types.FieldIndex(i)], err = fieldtrie.NewFieldTrie(types.FieldIndex(i), types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
		}
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[randaoMixes] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[blockRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[previousEpochParticipationBits] = stateutil.NewRef(1) // New in Altair.
	b.sharedFieldReferences[currentEpochParticipationBits] = stateutil.NewRef(1)  // New in Altair.
	b.sharedFieldReferences[slashings] = stateutil.NewRef(1)
	b.sharedFieldReferences[eth1DataVotes] = stateutil.NewRef(1)
	b.sharedFieldReferences[validators] = stateutil.NewRef(1)
	b.sharedFieldReferences[balances] = stateutil.NewRef(1)
	b.sharedFieldReferences[inactivityScores] = stateutil.NewRef(1) // New in Altair.
	b.sharedFieldReferences[historicalRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[latestExecutionPayloadHeader] = stateutil.NewRef(1) // New in Merge.

	stateCount.Inc()

	return b, nil
}

func Initialize() (*BeaconState, error) {
	fieldCount := params.BeaconConfig().BeaconStateAltairFieldCount
	sRoots := customtypes.StateRoots([customtypes.StateRootsSize][32]byte{})
	bRoots := customtypes.BlockRoots([customtypes.BlockRootsSize][32]byte{})
	mixes := customtypes.RandaoMixes([customtypes.RandaoMixesSize][32]byte{})
	b := &BeaconState{
		dirtyFields:           make(map[types.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[types.FieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[types.FieldIndex]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[types.FieldIndex]*stateutil.Reference, 11),
		rebuildTrie:           make(map[types.FieldIndex]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler([]*ethpb.Validator{}),
		stateRoots:            &sRoots,
		blockRoots:            &bRoots,
		randaoMixes:           &mixes,
	}

	var err error
	for i := 0; i < fieldCount; i++ {
		b.dirtyFields[types.FieldIndex(i)] = true
		b.rebuildTrie[types.FieldIndex(i)] = true
		b.dirtyIndices[types.FieldIndex(i)] = []uint64{}
		b.stateFieldLeaves[types.FieldIndex(i)], err = fieldtrie.NewFieldTrie(types.FieldIndex(i), types.BasicArray, nil, 0)
		if err != nil {
			return nil, err
		}
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[randaoMixes] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[blockRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[previousEpochParticipationBits] = stateutil.NewRef(1) // New in Altair.
	b.sharedFieldReferences[currentEpochParticipationBits] = stateutil.NewRef(1)  // New in Altair.
	b.sharedFieldReferences[slashings] = stateutil.NewRef(1)
	b.sharedFieldReferences[eth1DataVotes] = stateutil.NewRef(1)
	b.sharedFieldReferences[validators] = stateutil.NewRef(1)
	b.sharedFieldReferences[balances] = stateutil.NewRef(1)
	b.sharedFieldReferences[inactivityScores] = stateutil.NewRef(1) // New in Altair.
	b.sharedFieldReferences[historicalRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[latestExecutionPayloadHeader] = stateutil.NewRef(1) // New in Merge.

	stateCount.Inc()
	return b, nil
}

// Copy returns a deep copy of the beacon state.
func (b *BeaconState) Copy() state.BeaconState {
	b.lock.RLock()
	defer b.lock.RUnlock()
	fieldCount := params.BeaconConfig().BeaconStateMergeFieldCount

	dst := &BeaconState{
		// Primitive types, safe to copy.
		genesisTime:      b.genesisTime,
		slot:             b.slot,
		eth1DepositIndex: b.eth1DepositIndex,

		// Large arrays, infrequently changed, constant size.
		randaoMixes:   b.randaoMixes,
		stateRoots:    b.stateRoots,
		blockRoots:    b.blockRoots,
		slashings:     b.slashings,
		eth1DataVotes: b.eth1DataVotes,

		// Large arrays, increases over time.
		validators:                 b.validators,
		balances:                   b.balances,
		historicalRoots:            b.historicalRoots,
		previousEpochParticipation: b.previousEpochParticipation,
		currentEpochParticipation:  b.currentEpochParticipation,
		inactivityScores:           b.inactivityScores,

		// Everything else, too small to be concerned about, constant size.
		fork:                         b.fork,
		latestBlockHeader:            b.latestBlockHeader,
		eth1Data:                     b.eth1Data,
		justificationBits:            b.justificationBits,
		previousJustifiedCheckpoint:  b.previousJustifiedCheckpoint,
		currentJustifiedCheckpoint:   b.currentJustifiedCheckpoint,
		finalizedCheckpoint:          b.finalizedCheckpoint,
		genesisValidatorsRoot:        b.genesisValidatorsRoot,
		currentSyncCommittee:         b.currentSyncCommittee,
		nextSyncCommittee:            b.nextSyncCommittee,
		latestExecutionPayloadHeader: b.latestExecutionPayloadHeader,

		dirtyFields:           make(map[types.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[types.FieldIndex][]uint64, fieldCount),
		rebuildTrie:           make(map[types.FieldIndex]bool, fieldCount),
		sharedFieldReferences: make(map[types.FieldIndex]*stateutil.Reference, 11),
		stateFieldLeaves:      make(map[types.FieldIndex]*fieldtrie.FieldTrie, fieldCount),

		// Copy on write validator index map.
		valMapHandler: b.valMapHandler,
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
	stateCount.Inc()
	// Finalizer runs when dst is being destroyed in garbage collection.
	runtime.SetFinalizer(dst, func(b *BeaconState) {
		for field, v := range b.sharedFieldReferences {
			v.MinusRef()
			if b.stateFieldLeaves[field].FieldReference() != nil {
				b.stateFieldLeaves[field].FieldReference().MinusRef()
			}
		}
		for i := 0; i < fieldCount; i++ {
			field := types.FieldIndex(i)
			delete(b.stateFieldLeaves, field)
			delete(b.dirtyIndices, field)
			delete(b.dirtyFields, field)
			delete(b.sharedFieldReferences, field)
			delete(b.stateFieldLeaves, field)
		}
		stateCount.Sub(1)
	})

	return dst
}

// HashTreeRoot of the beacon state retrieves the Merkle root of the trie
// representation of the beacon state based on the eth2 Simple Serialize specification.
func (b *BeaconState) HashTreeRoot(ctx context.Context) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "BeaconStateMerge.HashTreeRoot")
	defer span.End()

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.merkleLayers == nil || len(b.merkleLayers) == 0 {
		fieldRoots, err := computeFieldRoots(ctx, b)
		if err != nil {
			return [32]byte{}, err
		}
		layers := stateutil.Merkleize(fieldRoots)
		b.merkleLayers = layers
		b.dirtyFields = make(map[types.FieldIndex]bool, params.BeaconConfig().BeaconStateMergeFieldCount)
	}

	for field := range b.dirtyFields {
		root, err := b.rootSelector(field)
		if err != nil {
			return [32]byte{}, err
		}
		b.merkleLayers[0][field] = root[:]
		b.recomputeRoot(int(field))
		delete(b.dirtyFields, field)
	}
	return bytesutil.ToBytes32(b.merkleLayers[len(b.merkleLayers)-1][0]), nil
}

// FieldReferencesCount returns the reference count held by each field. This
// also includes the field trie held by each field.
func (b *BeaconState) FieldReferencesCount() map[string]uint64 {
	refMap := make(map[string]uint64)
	b.lock.RLock()
	defer b.lock.RUnlock()
	for i, f := range b.sharedFieldReferences {
		refMap[i.String(b.Version())] = uint64(f.Refs())
	}
	for i, f := range b.stateFieldLeaves {
		numOfRefs := uint64(f.FieldReference().Refs())
		f.RLock()
		if !f.Empty() {
			refMap[i.String(b.Version())+"_trie"] = numOfRefs
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

func (b *BeaconState) rootSelector(field types.FieldIndex) ([32]byte, error) {
	hasher := hash.CustomSHA256Hasher()
	switch field {
	case genesisTime:
		return ssz.Uint64Root(b.genesisTime), nil
	case genesisValidatorRoot:
		return b.genesisValidatorsRoot, nil
	case slot:
		return ssz.Uint64Root(uint64(b.slot)), nil
	case eth1DepositIndex:
		return ssz.Uint64Root(b.eth1DepositIndex), nil
	case fork:
		return ssz.ForkRoot(b.fork)
	case latestBlockHeader:
		return stateutil.BlockHeaderRoot(b.latestBlockHeader)
	case blockRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.blockRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(blockRoots, b.blockRoots)
	case stateRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.stateRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(stateRoots, b.stateRoots)
	case historicalRoots:
		hRoots := make([][]byte, len(b.historicalRoots))
		for i := range hRoots {
			hRoots[i] = b.historicalRoots[i][:]
		}
		return ssz.ByteArrayRootWithLimit(hRoots, params.BeaconConfig().HistoricalRootsLimit)
	case eth1Data:
		return stateutil.Eth1Root(hasher, b.eth1Data)
	case eth1DataVotes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.eth1DataVotes, uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.eth1DataVotes)
	case validators:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.validators, params.BeaconConfig().ValidatorRegistryLimit)
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[validators] = []uint64{}
			delete(b.rebuildTrie, validators)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(validators, b.validators)
	case balances:
		return stateutil.Uint64ListRootWithRegistryLimit(b.balances)
	case randaoMixes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.randaoMixes, uint64(params.BeaconConfig().EpochsPerHistoricalVector))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(randaoMixes, b.randaoMixes)
	case slashings:
		return ssz.SlashingsRoot(b.slashings)
	case previousEpochParticipationBits:
		return stateutil.ParticipationBitsRoot(b.previousEpochParticipation)
	case currentEpochParticipationBits:
		return stateutil.ParticipationBitsRoot(b.currentEpochParticipation)
	case justificationBits:
		return bytesutil.ToBytes32(b.justificationBits), nil
	case previousJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.previousJustifiedCheckpoint)
	case currentJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.currentJustifiedCheckpoint)
	case finalizedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.finalizedCheckpoint)
	case inactivityScores:
		return stateutil.Uint64ListRootWithRegistryLimit(b.inactivityScores)
	case currentSyncCommittee:
		return stateutil.SyncCommitteeRoot(b.currentSyncCommittee)
	case nextSyncCommittee:
		return stateutil.SyncCommitteeRoot(b.nextSyncCommittee)
	case latestExecutionPayloadHeader:
		return b.latestExecutionPayloadHeader.HashTreeRoot()
	}
	return [32]byte{}, errors.New("invalid field index provided")
}

func (b *BeaconState) recomputeFieldTrie(index types.FieldIndex, elements interface{}) ([32]byte, error) {
	fTrie := b.stateFieldLeaves[index]
	if fTrie.FieldReference().Refs() > 1 {
		fTrie.Lock()
		defer fTrie.Unlock()
		fTrie.FieldReference().MinusRef()
		newTrie := fTrie.CopyTrie()
		b.stateFieldLeaves[index] = newTrie
		fTrie = newTrie
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

func (b *BeaconState) resetFieldTrie(index types.FieldIndex, elements interface{}, length uint64) error {
	fTrie, err := fieldtrie.NewFieldTrie(index, fieldMap[index], elements, length)
	if err != nil {
		return err
	}
	b.stateFieldLeaves[index] = fTrie
	b.dirtyIndices[index] = []uint64{}
	return nil
}
