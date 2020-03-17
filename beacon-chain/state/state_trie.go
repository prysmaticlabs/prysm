package state

import (
	"runtime"
	"sort"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/protolambda/zssz/merkle"
	coreutils "github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/memorypool"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *pbp2p.BeaconState) (*BeaconState, error) {
	return InitializeFromProtoUnsafe(proto.Clone(st).(*pbp2p.BeaconState))
}

// InitializeFromProtoUnsafe directly uses the beacon state protobuf pointer
// and sets it as the inner state of the BeaconState type.
func InitializeFromProtoUnsafe(st *pbp2p.BeaconState) (*BeaconState, error) {
	b := &BeaconState{
		state:                 st,
		dirtyFields:           make(map[fieldIndex]interface{}, 20),
		dirtyIndexes:          make(map[fieldIndex][]uint64, 20),
		stateFieldLeaves:      make(map[fieldIndex]*FieldTrie, 20),
		sharedFieldReferences: make(map[fieldIndex]*reference, 10),
		rebuildTrie:           make(map[fieldIndex]bool, 20),
		valIdxMap:             coreutils.ValidatorIndexMap(st.Validators),
	}

	for i := 0; i < 20; i++ {
		b.dirtyFields[fieldIndex(i)] = true
		b.rebuildTrie[fieldIndex(i)] = true
		b.dirtyIndexes[fieldIndex(i)] = []uint64{}
		b.stateFieldLeaves[fieldIndex(i)] = &FieldTrie{
			field:     fieldIndex(i),
			reference: &reference{1},
			Mutex:     new(sync.Mutex),
		}

	}
	var err error
	if featureconfig.Get().EnableSSZCache {
		b.stateFieldLeaves[blockRoots], err = NewFieldTrie(blockRoots, b.state.BlockRoots, params.BeaconConfig().SlotsPerHistoricalRoot)
		if err != nil {
			panic(err)
		}
		b.stateFieldLeaves[stateRoots], err = NewFieldTrie(stateRoots, b.state.StateRoots, params.BeaconConfig().SlotsPerHistoricalRoot)
		if err != nil {
			panic(err)
		}
		//b.stateFieldLeaves[randaoMixes] = NewFieldTrie(randaoMixes, b.state.RandaoMixes, params.BeaconConfig().EpochsPerHistoricalVector)
		/*layers, err := stateutil.Eth1DataVotesRootWithTrie(b.state.Eth1DataVotes)
		if err != nil {
			panic(err)
		}
		b.stateFieldLeaves[eth1DataVotes] = &FieldTrie{
			fieldLayers: layers,
			field:       eth1DataVotes,
			reference:   &reference{1},
			Mutex:       new(sync.Mutex),
		}*/

		/*layers, err = stateutil.ValidatorsRootWithTrie(b.state.Validators)
		if err != nil {
			panic(err)
		}

		b.stateFieldLeaves[validators] = &FieldTrie{
			fieldLayers: layers,
			field:       validators,
			reference:   &reference{1},
			Mutex:       new(sync.Mutex),
		}*/
	}

	// Initialize field reference tracking for shared data.
	b.sharedFieldReferences[randaoMixes] = &reference{refs: 1}
	b.sharedFieldReferences[stateRoots] = &reference{refs: 1}
	b.sharedFieldReferences[blockRoots] = &reference{refs: 1}
	b.sharedFieldReferences[previousEpochAttestations] = &reference{refs: 1}
	b.sharedFieldReferences[currentEpochAttestations] = &reference{refs: 1}
	b.sharedFieldReferences[slashings] = &reference{refs: 1}
	b.sharedFieldReferences[eth1DataVotes] = &reference{refs: 1}
	b.sharedFieldReferences[validators] = &reference{refs: 1}
	b.sharedFieldReferences[balances] = &reference{refs: 1}
	b.sharedFieldReferences[historicalRoots] = &reference{refs: 1}

	return b, nil
}

// Copy returns a deep copy of the beacon state.
func (b *BeaconState) Copy() *BeaconState {
	if !b.HasInnerState() {
		return nil
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
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
			Fork:                        b.Fork(),
			LatestBlockHeader:           b.LatestBlockHeader(),
			Eth1Data:                    b.Eth1Data(),
			JustificationBits:           b.JustificationBits(),
			PreviousJustifiedCheckpoint: b.PreviousJustifiedCheckpoint(),
			CurrentJustifiedCheckpoint:  b.CurrentJustifiedCheckpoint(),
			FinalizedCheckpoint:         b.FinalizedCheckpoint(),
		},
		dirtyFields:           make(map[fieldIndex]interface{}, 20),
		dirtyIndexes:          make(map[fieldIndex][]uint64, 20),
		rebuildTrie:           make(map[fieldIndex]bool, 20),
		sharedFieldReferences: make(map[fieldIndex]*reference, 10),
		stateFieldLeaves:      make(map[fieldIndex]*FieldTrie, 20),

		// Copy on write validator index map.
		valIdxMap: b.valIdxMap,
	}

	for field, ref := range b.sharedFieldReferences {
		ref.refs++
		dst.sharedFieldReferences[field] = ref
	}

	for i := range b.dirtyFields {
		dst.dirtyFields[i] = true
	}

	for i := range b.dirtyIndexes {
		indices := make([]uint64, len(b.dirtyIndexes[i]))
		copy(indices, b.dirtyIndexes[i])
		dst.dirtyIndexes[i] = indices
	}

	for i := range b.rebuildTrie {
		dst.rebuildTrie[i] = true
	}

	for fldIdx, fieldTrie := range b.stateFieldLeaves {
		dst.stateFieldLeaves[fldIdx] = fieldTrie
		if fieldTrie.reference != nil {
			fieldTrie.Lock()
			fieldTrie.AddRef()
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
			v.refs--
			if b.stateFieldLeaves[field].reference != nil {
				b.stateFieldLeaves[field].MinusRef()
			}
			if field == randaoMixes && v.refs == 0 {
				memorypool.PutDoubleByteSlice(b.state.RandaoMixes)
				if b.stateFieldLeaves[field].refs == 0 {
					memorypool.PutTripleByteSliceRandaoMixes(b.stateFieldLeaves[randaoMixes].fieldLayers)
				}
			}

			if field == blockRoots && v.refs == 0 && b.stateFieldLeaves[field].refs == 0 {
				memorypool.PutTripleByteSliceBlockRoots(b.stateFieldLeaves[blockRoots].fieldLayers)
			}

			if field == stateRoots && v.refs == 0 && b.stateFieldLeaves[field].refs == 0 {
				memorypool.PutTripleByteSliceStateRoots(b.stateFieldLeaves[stateRoots].fieldLayers)
			}

			if field == stateRoots && v.refs == 0 && b.stateFieldLeaves[field].refs == 0 {
				memorypool.PutTripleByteSliceStateRoots(b.stateFieldLeaves[stateRoots].fieldLayers)
			}
		}
	})

	return dst
}

// HashTreeRoot of the beacon state retrieves the Merkle root of the trie
// representation of the beacon state based on the eth2 Simple Serialize specification.
func (b *BeaconState) HashTreeRoot() ([32]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.merkleLayers == nil || len(b.merkleLayers) == 0 {
		fieldRoots, err := stateutil.ComputeFieldRoots(b.state)
		if err != nil {
			return [32]byte{}, err
		}
		layers := merkleize(fieldRoots)
		b.merkleLayers = layers
		b.dirtyFields = make(map[fieldIndex]interface{})
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

func (b *BeaconState) InnerMerkleTrie(field fieldIndex) [][][32]byte {
	refLayers := make([][][32]byte, len(b.stateFieldLeaves[field].fieldLayers))
	for i, val := range b.stateFieldLeaves[field].fieldLayers {
		refLayers[i] = make([][32]byte, len(val))
		for j, innerVal := range val {
			refLayers[i][j] = *innerVal
		}
	}
	return refLayers
}

// Merkleize 32-byte leaves into a Merkle trie for its adequate depth, returning
// the resulting layers of the trie based on the appropriate depth. This function
// pads the leaves to a power-of-two length.
func merkleize(leaves [][]byte) [][][]byte {
	hashFunc := hashutil.CustomSHA256Hasher()
	layers := make([][][]byte, merkle.GetDepth(uint64(len(leaves)))+1)
	for len(leaves) != 32 {
		leaves = append(leaves, make([]byte, 32))
	}
	currentLayer := leaves
	layers[0] = currentLayer

	// We keep track of the hash layers of a Merkle trie until we reach
	// the top layer of length 1, which contains the single root element.
	//        [Root]      -> Top layer has length 1.
	//    [E]       [F]   -> This layer has length 2.
	// [A]  [B]  [C]  [D] -> The bottom layer has length 4 (needs to be a power of two).
	i := 1
	for len(currentLayer) > 1 && i < len(layers) {
		layer := make([][]byte, 0)
		for i := 0; i < len(currentLayer); i += 2 {
			hashedChunk := hashFunc(append(currentLayer[i], currentLayer[i+1]...))
			layer = append(layer, hashedChunk[:])
		}
		currentLayer = layer
		layers[i] = currentLayer
		i++
	}
	return layers
}

func (b *BeaconState) rootSelector(field fieldIndex) ([32]byte, error) {
	switch field {
	case genesisTime:
		return stateutil.Uint64Root(b.state.GenesisTime), nil
	case slot:
		return stateutil.Uint64Root(b.state.Slot), nil
	case eth1DepositIndex:
		return stateutil.Uint64Root(b.state.Eth1DepositIndex), nil
	case fork:
		return stateutil.ForkRoot(b.state.Fork)
	case latestBlockHeader:
		return stateutil.BlockHeaderRoot(b.state.LatestBlockHeader)
	case blockRoots:
		if featureconfig.Get().EnableSSZCache {
			root, err := b.recomputeFieldTrie(blockRoots, b.state.BlockRoots)
			if err != nil {
				return [32]byte{}, err
			}
			/*
				newRoot, _ := stateutil.RootsArrayHashTreeRoot(b.state.BlockRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "BlockRoots")
				if newRoot != root {
					refLayers := make([][][32]byte, len(b.stateFieldLeaves[blockRoots].fieldLayers))
					for i, val := range b.stateFieldLeaves[blockRoots].fieldLayers {
						refLayers[i] = make([][32]byte, len(val))
						for j, innerVal := range val {
							refLayers[i][j] = *innerVal
						}
					}
					diff, _ := messagediff.PrettyDiff(stateutil.LayerCache("BlockRoots")[0], refLayers[0])
					log.Errorf("different roots for field %d and diff %s", field, diff)
				}*/
			return root, nil
		}
		return stateutil.RootsArrayHashTreeRoot(b.state.BlockRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "BlockRoots")
	case stateRoots:
		if featureconfig.Get().EnableSSZCache {
			root, err := b.recomputeFieldTrie(stateRoots, b.state.StateRoots)
			if err != nil {
				return [32]byte{}, err
			}
			/*
				newRoot, _ := stateutil.RootsArrayHashTreeRoot(b.state.StateRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "StateRoots")
				if newRoot != root {
					refLayers := make([][][32]byte, len(b.stateFieldLeaves[stateRoots].fieldLayers))
					for i, val := range b.stateFieldLeaves[stateRoots].fieldLayers {
						refLayers[i] = make([][32]byte, len(val))
						for j, innerVal := range val {
							refLayers[i][j] = *innerVal
						}
					}
					diff, _ := messagediff.PrettyDiff(stateutil.LayerCache("StateRoots")[0], refLayers[0])
					log.Errorf("different roots for field %d and diff %s", field, diff)
				}*/
			return root, nil
		}
		return stateutil.RootsArrayHashTreeRoot(b.state.StateRoots, params.BeaconConfig().SlotsPerHistoricalRoot, "StateRoots")
	case historicalRoots:
		return stateutil.HistoricalRootsRoot(b.state.HistoricalRoots)
	case eth1Data:
		return stateutil.Eth1Root(b.state.Eth1Data)
	case eth1DataVotes:
		if featureconfig.Get().EnableSSZCache {
			if b.rebuildTrie[field] {
				err := b.resetFieldTrie(field, b.state.Eth1DataVotes, params.BeaconConfig().SlotsPerEth1VotingPeriod)
				if err != nil {
					return [32]byte{}, err
				}
				b.dirtyIndexes[field] = []uint64{}
				delete(b.rebuildTrie, field)
				return b.stateFieldLeaves[field].TrieRoot()
			}
			return b.recomputeFieldTrie(field, b.state.Eth1DataVotes)
		}
		return stateutil.Eth1DataVotesRoot(b.state.Eth1DataVotes)
	case validators:
		if featureconfig.Get().EnableSSZCache {
			if b.rebuildTrie[field] {
				err := b.resetFieldTrie(field, b.state.Validators, params.BeaconConfig().ValidatorRegistryLimit)
				if err != nil {
					return [32]byte{}, err
				}
				b.dirtyIndexes[validators] = []uint64{}
				delete(b.rebuildTrie, validators)
				return b.stateFieldLeaves[field].TrieRoot()
			}
			return b.recomputeFieldTrie(validators, b.state.Validators)
		}
		return stateutil.ValidatorRegistryRoot(b.state.Validators)
	case balances:
		return stateutil.ValidatorBalancesRoot(b.state.Balances)
	case randaoMixes:
		if featureconfig.Get().EnableSSZCache {
			if b.rebuildTrie[field] {
				err := b.resetFieldTrie(field, b.state.RandaoMixes, params.BeaconConfig().EpochsPerHistoricalVector)
				if err != nil {
					return [32]byte{}, err
				}
				b.dirtyIndexes[field] = []uint64{}
				delete(b.rebuildTrie, field)
				return b.stateFieldLeaves[field].TrieRoot()
			}
			root, err := b.recomputeFieldTrie(randaoMixes, b.state.RandaoMixes)
			if err != nil {
				return [32]byte{}, err
			}
			/*
				newRoot, _ := stateutil.RootsArrayHashTreeRoot(b.state.RandaoMixes, params.BeaconConfig().EpochsPerHistoricalVector, "RandaoMixes")
				if newRoot != root {
					refLayers := make([][][32]byte, len(b.stateFieldLeaves[randaoMixes].fieldLayers))
					for i, val := range b.stateFieldLeaves[randaoMixes].fieldLayers {
						refLayers[i] = make([][32]byte, len(val))
						for j, innerVal := range val {
							refLayers[i][j] = *innerVal
						}
					}
					diff, _ := messagediff.PrettyDiff(stateutil.LayerCache("RandaoMixes")[0], refLayers[0])
					log.Errorf("different roots for field %d and diff %s", field, diff)
				}*/
			return root, nil
		}
		return stateutil.RootsArrayHashTreeRoot(b.state.RandaoMixes, params.BeaconConfig().EpochsPerHistoricalVector, "RandaoMixes")
	case slashings:
		return stateutil.SlashingsRoot(b.state.Slashings)
	case previousEpochAttestations:
		return stateutil.EpochAttestationsRoot(b.state.PreviousEpochAttestations)
	case currentEpochAttestations:
		return stateutil.EpochAttestationsRoot(b.state.CurrentEpochAttestations)
	case justificationBits:
		return bytesutil.ToBytes32(b.state.JustificationBits), nil
	case previousJustifiedCheckpoint:
		return stateutil.CheckpointRoot(b.state.PreviousJustifiedCheckpoint)
	case currentJustifiedCheckpoint:
		return stateutil.CheckpointRoot(b.state.CurrentJustifiedCheckpoint)
	case finalizedCheckpoint:
		return stateutil.CheckpointRoot(b.state.FinalizedCheckpoint)
	}
	return [32]byte{}, errors.New("invalid field index provided")
}

func (b *BeaconState) recomputeFieldTrie(index fieldIndex, elements interface{}) ([32]byte, error) {
	fTrie := b.stateFieldLeaves[index]
	if fTrie.refs > 1 {
		fTrie.Lock()
		defer fTrie.Unlock()
		fTrie.MinusRef()
		newTrie := fTrie.CopyTrie()
		b.stateFieldLeaves[index] = newTrie
		fTrie = newTrie
	}
	sort.Slice(b.dirtyIndexes[index], func(i int, j int) bool {
		return b.dirtyIndexes[index][i] < b.dirtyIndexes[index][j]
	})
	root, err := fTrie.RecomputeTrie(b.dirtyIndexes[index], elements)
	if err != nil {
		return [32]byte{}, err
	}
	b.dirtyIndexes[index] = []uint64{}

	return root, nil
}

/*
func (b *BeaconState) recomputeFieldTrieTest(index fieldIndex, elements []*ethpb.Eth1Data) ([32]byte, error) {
	fTrie := b.stateFieldLeaves[index]
	if fTrie.refs > 1 {
		fTrie.Lock()
		defer fTrie.Unlock()
		fTrie.MinusRef()
		newTrie := fTrie.CopyTrie()
		b.stateFieldLeaves[index] = newTrie
		fTrie = newTrie
	}
	changedRts, err := stateutil.ReturnChangedEth1Data(elements, b.dirtyIndexes[index])
	if err != nil {
		return [32]byte{}, err
	}
	root, err := fTrie.RecomputeTrieVariable(b.dirtyIndexes[index], changedRts)
	if err != nil {
		return [32]byte{}, err
	}
	b.dirtyIndexes[index] = []uint64{}

	return stateutil.AddInMixin(root, uint64(len(elements)))
} */

func (b *BeaconState) resetFieldTrie(index fieldIndex, elements interface{}, length uint64) error {
	fTrie := b.stateFieldLeaves[index]
	var err error
	fTrie, err = NewFieldTrie(index, elements, length)
	if err != nil {
		return err
	}
	b.stateFieldLeaves[index] = fTrie
	b.dirtyIndexes[index] = []uint64{}
	return nil
}
