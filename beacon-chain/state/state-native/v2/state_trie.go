package v2

import (
	"context"
	"io"
	"io/ioutil"
	"runtime"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/fieldtrie"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/custom-types"
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
func InitializeFromProto(st *ethpb.BeaconStateAltair) (*BeaconState, error) {
	return InitializeFromProtoUnsafe(proto.Clone(st).(*ethpb.BeaconStateAltair))
}

// InitializeFromSSZReader can be used when the source for a serialized BeaconState object
// is an io.Reader. This allows client code to remain agnostic about whether the data comes
// from the network or a file without needing to read the entire state into mem as a large byte slice.
func InitializeFromSSZReader(r io.Reader) (*BeaconState, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return InitializeFromSSZBytes(b)
}

// InitializeFromSSZBytes is a convenience method to obtain a BeaconState by unmarshaling
// a slice of bytes containing the ssz-serialized representation of the state.
func InitializeFromSSZBytes(marshaled []byte) (*BeaconState, error) {
	st := &ethpb.BeaconStateAltair{}
	if err := st.UnmarshalSSZ(marshaled); err != nil {
		return nil, err
	}
	return InitializeFromProtoUnsafe(st)
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf fields
// and sets them as fields of the BeaconState type.
func InitializeFromProtoUnsafe(st *ethpb.BeaconStateAltair) (*BeaconState, error) {
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

	state.StateCount.Inc()
	return b, nil
}

// Copy returns a deep copy of the beacon state.
func (b *BeaconState) Copy() state.BeaconState {
	b.lock.RLock()
	defer b.lock.RUnlock()
	fieldCount := params.BeaconConfig().BeaconStateAltairFieldCount

	dst := &BeaconState{
		// Primitive types, safe to copy.
		genesisTime:      b.genesisTime,
		slot:             b.slot,
		eth1DepositIndex: b.eth1DepositIndex,

		// Large arrays, infrequently changed, constant size.
		blockRoots:    b.blockRoots,
		stateRoots:    b.stateRoots,
		randaoMixes:   b.randaoMixes,
		slashings:     b.slashings,
		eth1DataVotes: b.eth1DataVotes,

		// Large arrays, increases over time.
		historicalRoots:            b.historicalRoots,
		balances:                   b.balances,
		previousEpochParticipation: b.previousEpochParticipation,
		currentEpochParticipation:  b.currentEpochParticipation,
		inactivityScores:           b.inactivityScores,
		validators:                 b.validators,

		// Everything else, too small to be concerned about, constant size.
		genesisValidatorsRoot:       b.genesisValidatorsRoot,
		justificationBits:           b.justificationBitsVal(),
		fork:                        b.forkVal(),
		latestBlockHeader:           b.latestBlockHeaderVal(),
		eth1Data:                    b.eth1DataVal(),
		previousJustifiedCheckpoint: b.previousJustifiedCheckpointVal(),
		currentJustifiedCheckpoint:  b.currentJustifiedCheckpointVal(),
		finalizedCheckpoint:         b.finalizedCheckpointVal(),
		currentSyncCommittee:        b.currentSyncCommitteeVal(),
		nextSyncCommittee:           b.nextSyncCommitteeVal(),

		dirtyFields:           make(map[types.FieldIndex]bool, fieldCount),
		dirtyIndices:          make(map[types.FieldIndex][]uint64, fieldCount),
		rebuildTrie:           make(map[types.FieldIndex]bool, fieldCount),
		sharedFieldReferences: make(map[types.FieldIndex]*stateutil.Reference, 11),
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
// representation of the beacon state based on the eth2 Simple Serialize specification.
func (b *BeaconState) HashTreeRoot(ctx context.Context) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "beaconStateAltair.HashTreeRoot")
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
	protoState, ok := b.ToProtoUnsafe().(*ethpb.BeaconStateAltair)
	if !ok {
		return errors.New("state is of the wrong type")
	}
	fieldRoots, err := computeFieldRoots(ctx, protoState)
	if err != nil {
		return err
	}
	layers := stateutil.Merkleize(fieldRoots)
	b.merkleLayers = layers
	b.dirtyFields = make(map[types.FieldIndex]bool, params.BeaconConfig().BeaconStateAltairFieldCount)
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
	return b == nil
}

func (b *BeaconState) rootSelector(ctx context.Context, field types.FieldIndex) ([32]byte, error) {
	_, span := trace.StartSpan(ctx, "beaconState.rootSelector")
	defer span.End()
	span.AddAttributes(trace.StringAttribute("field", field.String(b.Version())))

	hasher := hash.CustomSHA256Hasher()
	switch field {
	case genesisTime:
		return ssz.Uint64Root(b.genesisTime), nil
	case genesisValidatorsRoot:
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
			err := b.resetFieldTrie(field, b.blockRoots, fieldparams.BlockRootsLength)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, field)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(blockRoots, b.blockRoots)
	case stateRoots:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.stateRoots, fieldparams.StateRootsLength)
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
		return ssz.ByteArrayRootWithLimit(hRoots, fieldparams.HistoricalRootsLength)
	case eth1Data:
		return stateutil.Eth1Root(hasher, b.eth1Data)
	case eth1DataVotes:
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
	case validators:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.validators, fieldparams.ValidatorRegistryLimit)
			if err != nil {
				return [32]byte{}, err
			}
			delete(b.rebuildTrie, validators)
			return b.stateFieldLeaves[field].TrieRoot()
		}
		return b.recomputeFieldTrie(validators, b.validators)
	case balances:
		if features.Get().EnableBalanceTrieComputation {
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
			return b.recomputeFieldTrie(balances, b.balances)
		}
		return stateutil.Uint64ListRootWithRegistryLimit(b.balances)
	case randaoMixes:
		if b.rebuildTrie[field] {
			err := b.resetFieldTrie(field, b.randaoMixes, fieldparams.RandaoMixesLength)
			if err != nil {
				return [32]byte{}, err
			}
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
