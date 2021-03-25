package stateV0

import (
	"context"
	"runtime"
	"sort"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"go.opencensus.io/trace"
)

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *pbp2p.BeaconState) (*BeaconState, error) {
	return InitializeFromProtoUnsafe(proto.Clone(st).(*pbp2p.BeaconState))
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf pointer
// and sets it as the inner state of the BeaconState type.
func InitializeFromProtoUnsafe(st *pbp2p.BeaconState) (*BeaconState, error) {
	if st == nil {
		return nil, errors.New("received nil state")
	}

	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	b := &BeaconState{
		state:                 st,
		dirtyFields:           make(map[stateutil.FieldIndex]interface{}, fieldCount),
		dirtyIndices:          make(map[stateutil.FieldIndex][]uint64, fieldCount),
		stateFieldLeaves:      make(map[stateutil.FieldIndex]*stateutil.FieldTrie, fieldCount),
		sharedFieldReferences: make(map[stateutil.FieldIndex]*stateutil.Reference, 10),
		rebuildTrie:           make(map[stateutil.FieldIndex]bool, fieldCount),
		valMapHandler:         stateutil.NewValMapHandler(st.Validators),
	}

	for i := 0; i < fieldCount; i++ {
		b.dirtyFields[stateutil.FieldIndex(i)] = true
		b.rebuildTrie[stateutil.FieldIndex(i)] = true
		b.dirtyIndices[stateutil.FieldIndex(i)] = []uint64{}
		ft, err := stateutil.NewFieldTrie(stateutil.FieldIndex(i), nil, 0)
		if err != nil {
			return nil, err
		}
		b.stateFieldLeaves[stateutil.FieldIndex(i)] = ft
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[stateutil.RandaoMixes] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateutil.StateRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateutil.BlockRoots] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateutil.PreviousEpochAttestations] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateutil.CurrentEpochAttestations] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateutil.Slashings] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateutil.Eth1DataVotes] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateutil.Validators] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateutil.Balances] = stateutil.NewRef(1)
	b.sharedFieldReferences[stateutil.HistoricalRoots] = stateutil.NewRef(1)

	return b, nil
}

// Copy returns a deep copy of the beacon state.
func (b *BeaconState) Copy() iface.BeaconState {
	if !b.hasInnerState() {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()
	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	dst := &BeaconState{
		state: &pbp2p.BeaconState{
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
			GenesisValidatorsRoot:       b.genesisValidatorRoot(),
		},
		dirtyFields:           make(map[stateutil.FieldIndex]interface{}, fieldCount),
		dirtyIndices:          make(map[stateutil.FieldIndex][]uint64, fieldCount),
		rebuildTrie:           make(map[stateutil.FieldIndex]bool, fieldCount),
		sharedFieldReferences: make(map[stateutil.FieldIndex]*stateutil.Reference, 10),
		stateFieldLeaves:      make(map[stateutil.FieldIndex]*stateutil.FieldTrie, fieldCount),

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
		if fieldTrie.Reference() != nil {
			fieldTrie.Lock()
			fieldTrie.Reference().AddRef()
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

	// Finalizer runs when dst is being destroyed in garbage collection.
	runtime.SetFinalizer(dst, func(b *BeaconState) {
		for field, v := range b.sharedFieldReferences {
			v.MinusRef()
			if b.stateFieldLeaves[field].Reference() != nil {
				b.stateFieldLeaves[field].Reference().MinusRef()
			}
		}
	})

	return dst
}

// HashTreeRoot of the beacon state retrieves the Merkle root of the trie
// representation of the beacon state based on the eth2 Simple Serialize specification.
func (b *BeaconState) HashTreeRoot(ctx context.Context) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "beaconState.HashTreeRoot")
	defer span.End()

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.merkleLayers == nil || len(b.merkleLayers) == 0 {
		fieldRoots, err := computeFieldRoots(b.state)
		if err != nil {
			return [32]byte{}, err
		}
		layers := stateutil.Merkleize(fieldRoots)
		b.merkleLayers = layers
		b.dirtyFields = make(map[stateutil.FieldIndex]interface{}, params.BeaconConfig().BeaconStateFieldCount)
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
		refMap[i.String()] = uint64(f.Refs())
	}
	for i, f := range b.stateFieldLeaves {
		numOfRefs := uint64(f.Reference().Refs())
		f.RLock()
		if len(f.FieldLayers()) != 0 {
			refMap[i.String()+"_trie"] = numOfRefs
		}
		f.RUnlock()
	}
	return refMap
}

func (b *BeaconState) rootSelector(field stateutil.FieldIndex) ([32]byte, error) {
	hasher := hashutil.CustomSHA256Hasher()
	switch field {
	case stateutil.GenesisTime:
		return htrutils.Uint64Root(b.state.GenesisTime), nil
	case stateutil.GenesisValidatorRoot:
		return bytesutil.ToBytes32(b.state.GenesisValidatorsRoot), nil
	case stateutil.Slot:
		return htrutils.Uint64Root(uint64(b.state.Slot)), nil
	case stateutil.Eth1DepositIndex:
		return htrutils.Uint64Root(b.state.Eth1DepositIndex), nil
	case stateutil.Fork:
		return htrutils.ForkRoot(b.state.Fork)
	case stateutil.LatestBlockHeader:
		return stateutil.BlockHeaderRoot(b.state.LatestBlockHeader)
	case stateutil.BlockRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.BlockRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(stateutil.BlockRoots, b.state.BlockRoots)
	case stateutil.StateRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.StateRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(stateutil.StateRoots, b.state.StateRoots)
	case stateutil.HistoricalRoots:
		return htrutils.HistoricalRootsRoot(b.state.HistoricalRoots)
	case stateutil.Eth1Data:
		return eth1Root(hasher, b.state.Eth1Data)
	case stateutil.Eth1DataVotes:
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
	case stateutil.Validators:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.Validators, params.BeaconConfig().ValidatorRegistryLimit)
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[stateutil.Validators] = []uint64{}
			delete(b.rebuildTrie, stateutil.Validators)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(stateutil.Validators, b.state.Validators)
	case stateutil.Balances:
		return stateutil.Uint64ListRootWithRegistryLimit(b.state.Balances)
	case stateutil.RandaoMixes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.RandaoMixes, uint64(params.BeaconConfig().EpochsPerHistoricalVector))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(stateutil.RandaoMixes, b.state.RandaoMixes)
	case stateutil.Slashings:
		return htrutils.SlashingsRoot(b.state.Slashings)
	case stateutil.PreviousEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.PreviousEpochAttestations, uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.PreviousEpochAttestations)
	case stateutil.CurrentEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.state.CurrentEpochAttestations, uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)))
			if err != nil {
				return [32]byte{}, err
			}
			b.dirtyIndices[field] = []uint64{}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.CurrentEpochAttestations)
	case stateutil.JustificationBits:
		return bytesutil.ToBytes32(b.state.JustificationBits), nil
	case stateutil.PreviousJustifiedCheckpoint:
		return htrutils.CheckpointRoot(hasher, b.state.PreviousJustifiedCheckpoint)
	case stateutil.CurrentJustifiedCheckpoint:
		return htrutils.CheckpointRoot(hasher, b.state.CurrentJustifiedCheckpoint)
	case stateutil.FinalizedCheckpoint:
		return htrutils.CheckpointRoot(hasher, b.state.FinalizedCheckpoint)
	}
	return [32]byte{}, errors.New("invalid field index provided")
}

func (b *BeaconState) recomputeFieldTrie(index stateutil.FieldIndex, elements interface{}) ([32]byte, error) {
	fTrie := b.stateFieldLeaves[index]
	if fTrie.Reference().Refs() > 1 {
		fTrie.Lock()
		defer fTrie.Unlock()
		fTrie.Reference().MinusRef()
		newTrie := fTrie.CopyTrie()
		b.stateFieldLeaves[index] = newTrie
		fTrie = newTrie
	}
	// remove duplicate indexes
	b.dirtyIndices[index] = sliceutil.SetUint64(b.dirtyIndices[index])
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

func (b *BeaconState) resetFieldTrie(index stateutil.FieldIndex, elements interface{}, length uint64) error {
	fTrie, err := stateutil.NewFieldTrie(index, elements, length)
	if err != nil {
		return err
	}
	b.stateFieldLeaves[index] = fTrie
	b.dirtyIndices[index] = []uint64{}
	return nil
}
