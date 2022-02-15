package v1

import (
	"context"
	"runtime"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/fieldtrie"
	statenative "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/container/slice"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *ethpb.BeaconState) (state.BeaconState, error) {
	if features.Get().EnableNativeState {
		return statenative.InitializeFromProtoUnsafe(proto.Clone(st).(*ethpb.BeaconState))
	}
	return InitializeFromProtoUnsafe(proto.Clone(st).(*ethpb.BeaconState))
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf pointer
// and sets it as the inner state of the BeaconState type.
func InitializeFromProtoUnsafe(st *ethpb.BeaconState) (state.BeaconState, error) {
	if features.Get().EnableNativeState {
		return statenative.InitializeFromProtoUnsafe(st)
	}

	if st == nil {
		return nil, errors.New("received nil state")
	}

	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	b := &BeaconState{
		state:                 st,
		dirtyFields:           make(map[types.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[types.FieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[types.FieldIndex]*fieldtrie.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[types.FieldIndex]*stateutil.Reference, 10),
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
	b.sharedFieldReferences[previousEpochAttestations] = stateutil.NewRef(1)
	b.sharedFieldReferences[currentEpochAttestations] = stateutil.NewRef(1)
	b.sharedFieldReferences[slashings] = stateutil.NewRef(1)
	b.sharedFieldReferences[eth1DataVotes] = stateutil.NewRef(1)
	b.sharedFieldReferences[validators] = stateutil.NewRef(1)
	b.sharedFieldReferences[balances] = stateutil.NewRef(1)
	b.sharedFieldReferences[historicalRoots] = stateutil.NewRef(1)

	state.StateCount.Inc()
	return b, nil
}

// Copy returns a deep copy of the beacon state.
func (b *BeaconState) Copy() state.BeaconState {
	if !b.hasInnerState() {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()
	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	dst := &BeaconState{
		state: &ethpb.BeaconState{
			// Primitive types, safe to copy.
			GenesisTime:      b.state.GenesisTime,
			Slot:             b.state.Slot,
			Eth1DepositIndex: b.state.Eth1DepositIndex,

			// Large arrays, infrequently changed, constant size.
			RandaoMixes:               b.state.RandaoMixes,
			StateRoots:                b.state.StateRoots,
			BlockRoots:                b.state.BlockRoots,
			PreviousEpochAttestations: b.state.PreviousEpochAttestations,
			CurrentEpochAttestations:  b.state.CurrentEpochAttestations,
			Slashings:                 b.state.Slashings,
			Eth1DataVotes:             b.state.Eth1DataVotes,

			// Large arrays, increases over time.
			Validators:      b.state.Validators,
			Balances:        b.state.Balances,
			HistoricalRoots: b.state.HistoricalRoots,

			// Everything else, too small to be concerned about, constant size.
			Fork:                        b.fork(),
			LatestBlockHeader:           b.latestBlockHeader(),
			Eth1Data:                    b.eth1Data(),
			JustificationBits:           b.justificationBits(),
			PreviousJustifiedCheckpoint: b.previousJustifiedCheckpoint(),
			CurrentJustifiedCheckpoint:  b.currentJustifiedCheckpoint(),
			FinalizedCheckpoint:         b.finalizedCheckpoint(),
			GenesisValidatorsRoot:       b.genesisValidatorsRoot(),
		},
		dirtyFields:           make(map[types.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[types.FieldIndex][]uint64, fieldCount),
		rebuildTrie:           make(map[types.FieldIndex]bool, fieldCount),
		sharedFieldReferences: make(map[types.FieldIndex]*stateutil.Reference, 10),
		stateFieldLeaves:      make(map[types.FieldIndex]*fieldtrie.FieldTrie, fieldCount),

		// Share the reference to validator index map.
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

	state.StateCount.Inc()
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
	fieldRoots, err := computeFieldRoots(ctx, b.state)
	if err != nil {
		return err
	}
	layers := stateutil.Merkleize(fieldRoots)
	b.merkleLayers = layers
	b.dirtyFields = make(map[types.FieldIndex]bool, params.BeaconConfig().BeaconStateFieldCount)
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
		b.merkleLayers[0][field] = root[:]
		b.recomputeRoot(int(field))
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

func (b *BeaconState) rootSelector(ctx context.Context, field types.FieldIndex) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "beaconState.rootSelector")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("field", field.String(b.Version())))

	hasher := hash.CustomSHA256Hasher()
	switch field {
	case genesisTime:
		return ssz.Uint64Root(b.state.GenesisTime), nil
	case genesisValidatorsRoot:
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
			err := b.resetFieldTrie(field, b.state.BlockRoots, fieldparams.BlockRootsLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(blockRoots, b.state.BlockRoots)
	case stateRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.StateRoots, fieldparams.StateRootsLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(stateRoots, b.state.StateRoots)
	case historicalRoots:
		return ssz.ByteArrayRootWithLimit(b.state.HistoricalRoots, fieldparams.HistoricalRootsLength)
	case eth1Data:
		return stateutil.Eth1Root(hasher, b.state.Eth1Data)
	case eth1DataVotes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.state.Eth1DataVotes,
				fieldparams.Eth1DataVotesLength,
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.Eth1DataVotes)
	case validators:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.Validators, fieldparams.ValidatorRegistryLimit)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, validators)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(validators, b.state.Validators)
	case balances:
		if features.Get().EnableBalanceTrieComputation {
			if b.rebuildTrie[field] {
				maxBalCap := uint64(fieldparams.ValidatorRegistryLimit)
				elemSize := uint64(8)
				balLimit := (maxBalCap*elemSize + 31) / 32
				err := b.resetFieldTrie(field, b.state.Balances, balLimit)
				if err != nil {
					return [32]byte{}, err
				}
				delete(b.rebuildTrie, field)
				return b.stateFieldLeaves[field].TrieRoot()
			}
			return b.recomputeFieldTrie(balances, b.state.Balances)
		}
		return stateutil.Uint64ListRootWithRegistryLimit(b.state.Balances)
	case randaoMixes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.RandaoMixes, fieldparams.RandaoMixesLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(randaoMixes, b.state.RandaoMixes)
	case slashings:
		return ssz.SlashingsRoot(b.state.Slashings)
	case previousEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.state.PreviousEpochAttestations,
				fieldparams.PreviousEpochAttestationsLength,
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.PreviousEpochAttestations)
	case currentEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.state.CurrentEpochAttestations,
				fieldparams.CurrentEpochAttestationsLength,
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.CurrentEpochAttestations)
	case justificationBits:
		return bytesutil.ToBytes32(b.state.JustificationBits), nil
	case previousJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.state.PreviousJustifiedCheckpoint)
	case currentJustifiedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.state.CurrentJustifiedCheckpoint)
	case finalizedCheckpoint:
		return ssz.CheckpointRoot(hasher, b.state.FinalizedCheckpoint)
	}
	return [32]byte{}, errors.New("invalid field index provided")
}

func (b *BeaconState) recomputeFieldTrie(index types.FieldIndex, elements interface{}) ([32]byte, error) {
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

func (b *BeaconState) resetFieldTrie(index types.FieldIndex, elements interface{}, length uint64) error {
	fTrie, err := fieldtrie.NewFieldTrie(index, fieldMap[index], elements, length)
	if err != nil {
		return err
	}
	b.stateFieldLeaves[index] = fTrie
	b.dirtyIndices[index] = []uint64{}
	return nil
}
