package v3

import (
	"context"
	"runtime"
	"sort"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
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
	return InitializeFromProtoUnsafe(proto.Clone(st).(*ethpb.BeaconStateMerge))
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf pointer
// and sets it as the inner state of the BeaconState type.
func InitializeFromProtoUnsafe(st *ethpb.BeaconStateMerge) (*BeaconState, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	fieldCount := params.BeaconConfig().BeaconStateMergeFieldCount
	b := &BeaconState{
		state:                 st,
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

// Copy returns a deep copy of the beacon state.
func (b *BeaconState) Copy() state.BeaconState {
	if !b.hasInnerState() {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()
	fieldCount := params.BeaconConfig().BeaconStateMergeFieldCount

	dst := &BeaconState{
		state: &ethpb.BeaconStateMerge{
			// Primitive types, safe to copy.
			GenesisTime:      b.state.GenesisTime,
			Slot:             b.state.Slot,
			Eth1DepositIndex: b.state.Eth1DepositIndex,

			// Large arrays, infrequently changed, constant size.
			RandaoMixes:   b.state.RandaoMixes,
			StateRoots:    b.state.StateRoots,
			BlockRoots:    b.state.BlockRoots,
			Slashings:     b.state.Slashings,
			Eth1DataVotes: b.state.Eth1DataVotes,

			// Large arrays, increases over time.
			Validators:                 b.state.Validators,
			Balances:                   b.state.Balances,
			HistoricalRoots:            b.state.HistoricalRoots,
			PreviousEpochParticipation: b.state.PreviousEpochParticipation,
			CurrentEpochParticipation:  b.state.CurrentEpochParticipation,
			InactivityScores:           b.state.InactivityScores,

			// Everything else, too small to be concerned about, constant size.
			Fork:                         b.fork(),
			LatestBlockHeader:            b.latestBlockHeader(),
			Eth1Data:                     b.eth1Data(),
			JustificationBits:            b.justificationBits(),
			PreviousJustifiedCheckpoint:  b.previousJustifiedCheckpoint(),
			CurrentJustifiedCheckpoint:   b.currentJustifiedCheckpoint(),
			FinalizedCheckpoint:          b.finalizedCheckpoint(),
			GenesisValidatorsRoot:        b.genesisValidatorRoot(),
			CurrentSyncCommittee:         b.currentSyncCommittee(),
			NextSyncCommittee:            b.nextSyncCommittee(),
			LatestExecutionPayloadHeader: b.latestExecutionPayloadHeader(),
		},
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
		fieldRoots, err := computeFieldRoots(ctx, b.state)
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
	return b == nil || b.state == nil
}

func (b *BeaconState) rootSelector(field types.FieldIndex) ([32]byte, error) {
	hasher := hash.CustomSHA256Hasher()
	switch field {
	case genesisTime:
		return ssz.Uint64Root(b.state.GenesisTime), nil
	case genesisValidatorRoot:
		return bytesutil.ToBytes32(b.state.GenesisValidatorsRoot), nil
	case slot:
		return ssz.Uint64Root(uint64(b.state.Slot)), nil
	case eth1DepositIndex:
		return ssz.Uint64Root(b.state.Eth1DepositIndex), nil
	case fork:
		return ssz.ForkRoot(b.state.Fork)
	case latestBlockHeader:
		return stateutil.BlockHeaderRoot(b.state.LatestBlockHeader)
	case blockRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.BlockRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(blockRoots, b.state.BlockRoots)
	case stateRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.StateRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(stateRoots, b.state.StateRoots)
	case historicalRoots:
		return ssz.ByteArrayRootWithLimit(b.state.HistoricalRoots, params.BeaconConfig().HistoricalRootsLimit)
	case eth1Data:
		return stateutil.Eth1Root(hasher, b.state.Eth1Data)
	case eth1DataVotes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.Eth1DataVotes, uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.Eth1DataVotes)
	case validators:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.Validators, params.BeaconConfig().ValidatorRegistryLimit)
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[validators] = []uint64{}
			delete(b.rebuildTrie, validators)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(validators, b.state.Validators)
	case balances:
		return stateutil.Uint64ListRootWithRegistryLimit(b.state.Balances)
	case randaoMixes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.RandaoMixes, uint64(params.BeaconConfig().EpochsPerHistoricalVector))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(randaoMixes, b.state.RandaoMixes)
	case slashings:
		return ssz.SlashingsRoot(b.state.Slashings)
	case previousEpochParticipationBits:
		return stateutil.ParticipationBitsRoot(b.state.PreviousEpochParticipation)
	case currentEpochParticipationBits:
		return stateutil.ParticipationBitsRoot(b.state.CurrentEpochParticipation)
	case justificationBits:
		return bytesutil.ToBytes32(b.state.JustificationBits), nil
	case previousJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.state.PreviousJustifiedCheckpoint)
	case currentJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.state.CurrentJustifiedCheckpoint)
	case finalizedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.state.FinalizedCheckpoint)
	case inactivityScores:
		return stateutil.Uint64ListRootWithRegistryLimit(b.state.InactivityScores)
	case currentSyncCommittee:
		return stateutil.SyncCommitteeRoot(b.state.CurrentSyncCommittee)
	case nextSyncCommittee:
		return stateutil.SyncCommitteeRoot(b.state.NextSyncCommittee)
	case latestExecutionPayloadHeader:
		return b.state.LatestExecutionPayloadHeader.HashTreeRoot()
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
