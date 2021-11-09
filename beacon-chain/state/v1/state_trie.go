package v1

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
		Name: "beacon_state_count",
		Help: "Count the number of active beacon state objects.",
	})
)

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *ethpb.BeaconState) (*BeaconState, error) {
	return InitializeFromProtoUnsafe(proto.Clone(st).(*ethpb.BeaconState))
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf pointer
// and sets it as the inner state of the BeaconState type.
func InitializeFromProtoUnsafe(st *ethpb.BeaconState) (*BeaconState, error) {
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
	fieldCount := params.BeaconConfig().BeaconStateFieldCount
	dst := &BeaconState{
		// Primitive types, safe to copy.
		genesisTime:      b.genesisTime,
		slot:             b.slot,
		eth1DepositIndex: b.eth1DepositIndex,

		// Large arrays, infrequently changed, constant size.
		slashings: b.slashings,

		// Large arrays, infrequently changed, constant size.
		blockRoots:  b.blockRoots,
		stateRoots:  b.stateRoots,
		randaoMixes: b.randaoMixes,

		// Large arrays, increases over time.
		balances:        b.balances,
		historicalRoots: b.historicalRoots,

		// Everything else, too small to be concerned about, constant size.
		genesisValidatorsRoot: b.genesisValidatorsRoot,
		justificationBits:     b.justificationBits,

		state: &ethpb.BeaconState{
			// Large arrays, infrequently changed, constant size.
			PreviousEpochAttestations: b.state.PreviousEpochAttestations,
			CurrentEpochAttestations:  b.state.CurrentEpochAttestations,
			Eth1DataVotes:             b.state.Eth1DataVotes,

			// Large arrays, increases over time.
			Validators: b.state.Validators,

			// Everything else, too small to be concerned about, constant size.
			Fork:                        b.fork(),
			LatestBlockHeader:           b.latestBlockHeader(),
			Eth1Data:                    b.eth1Data(),
			PreviousJustifiedCheckpoint: b.previousJustifiedCheckpoint(),
			CurrentJustifiedCheckpoint:  b.currentJustifiedCheckpoint(),
			FinalizedCheckpoint:         b.finalizedCheckpoint(),
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
// representation of the beacon state based on the Ethereum Simple Serialize specification.
func (b *BeaconState) HashTreeRoot(ctx context.Context) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconState.HashTreeRoot")
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
		b.dirtyFields = make(map[types.FieldIndex]bool, params.BeaconConfig().BeaconStateFieldCount)
	}

	for field := range b.dirtyFields {
		root, err := b.rootSelector(ctx, field)
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

func (b *BeaconState) rootSelector(ctx context.Context, field types.FieldIndex) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "beaconState.rootSelector")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("field", field.String(b.Version())))

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
		return ssz.ForkRoot(b.state.Fork)
	case latestBlockHeader:
		return stateutil.BlockHeaderRoot(b.state.LatestBlockHeader)
	case blockRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.blockRoots, uint64(params.BeaconConfig().SlotsPerHistoricalRoot))
			if err != nil {
				return [32]byte{}, err
			}
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
		return eth1Root(hasher, b.state.Eth1Data)
	case eth1DataVotes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.state.Eth1DataVotes,
				uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))),
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
			err := b.resetFieldTrie(field, b.state.Validators, params.BeaconConfig().ValidatorRegistryLimit)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, validators)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(validators, b.state.Validators)
	case balances:
		return stateutil.Uint64ListRootWithRegistryLimit(b.balances)
	case randaoMixes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.randaoMixes, uint64(params.BeaconConfig().EpochsPerHistoricalVector))
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(randaoMixes, b.randaoMixes)
	case slashings:
		return ssz.SlashingsRoot(b.slashings)
	case previousEpochAttestations:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(
				field,
				b.state.PreviousEpochAttestations,
				uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)),
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
				uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)),
			)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(field, b.state.CurrentEpochAttestations)
	case justificationBits:
		return bytesutil.ToBytes32(b.justificationBits), nil
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
