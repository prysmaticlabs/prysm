package v2

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stateutil"
	stateTypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/types"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// For our setters, we have a field reference counter through
// which we can track shared field references. This helps when
// performing state copies, as we simply copy the reference to the
// field. When we do need to do need to modify these fields, we
// perform a full copy of the field. This is true of most of our
// fields except for the following below.
// 1) BlockRoots
// 2) StateRoots
// 3) Eth1DataVotes
// 4) RandaoMixes
// 5) HistoricalRoots
// 6) CurrentParticipationBits
// 7) PreviousParticipationBits
//
// The fields referred to above are instead copied by reference, where
// we simply copy the reference to the underlying object instead of the
// whole object. This is possible due to how we have structured our state
// as we copy the value on read, so as to ensure the underlying object is
// not mutated while it is being accessed during a state read.

const (
	// This specifies the limit till which we process all dirty indices for a certain field.
	// If we have more dirty indices than the threshold, then we rebuild the whole trie. This
	// comes due to the fact that O(alogn) > O(n) beyond a certain value of a.
	indicesLimit = 8000
)

// SetGenesisTime for the beacon state.
func (b *BeaconState) SetGenesisTime(val uint64) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.GenesisTime = val
	b.markFieldAsDirty(genesisTime)
	return nil
}

// SetGenesisValidatorsRoot for the beacon state.
func (b *BeaconState) SetGenesisValidatorsRoot(val []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.GenesisValidatorsRoot = val
	b.markFieldAsDirty(genesisValidatorsRoot)
	return nil
}

// SetSlot for the beacon state.
func (b *BeaconState) SetSlot(val types.Slot) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.state.Slot = val
	b.markFieldAsDirty(slot)
	return nil
}

// SetFork version for the beacon chain.
func (b *BeaconState) SetFork(val *ethpb.Fork) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	fk, ok := proto.Clone(val).(*ethpb.Fork)
	if !ok {
		return errors.New("proto.Clone did not return a fork proto")
	}
	b.state.Fork = fk
	b.markFieldAsDirty(fork)
	return nil
}

// SetHistoricalRoots for the beacon state. Updates the entire
// list to a new value by overwriting the previous one.
func (b *BeaconState) SetHistoricalRoots(val [][]byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sharedFieldReferences[historicalRoots].MinusRef()
	b.sharedFieldReferences[historicalRoots] = stateutil.NewRef(1)

	b.state.HistoricalRoots = val
	b.markFieldAsDirty(historicalRoots)
	return nil
}

// AppendHistoricalRoots for the beacon state. Appends the new value
// to the the end of list.
func (b *BeaconState) AppendHistoricalRoots(root [32]byte) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	b.lock.Lock()
	defer b.lock.Unlock()

	roots := b.state.HistoricalRoots
	if b.sharedFieldReferences[historicalRoots].Refs() > 1 {
		roots = make([][]byte, len(b.state.HistoricalRoots))
		copy(roots, b.state.HistoricalRoots)
		b.sharedFieldReferences[historicalRoots].MinusRef()
		b.sharedFieldReferences[historicalRoots] = stateutil.NewRef(1)
	}

	b.state.HistoricalRoots = append(roots, root[:])
	b.markFieldAsDirty(historicalRoots)
	return nil
}

// Recomputes the branch up the index in the Merkle trie representation
// of the beacon state. This method performs slice reads and the caller MUST
// hold the lock before calling this method.
func (b *BeaconState) recomputeRoot(idx int) {
	hashFunc := hash.CustomSHA256Hasher()
	layers := b.merkleLayers
	// The merkle tree structure looks as follows:
	// [[r1, r2, r3, r4], [parent1, parent2], [root]]
	// Using information about the index which changed, idx, we recompute
	// only its branch up the tree.
	currentIndex := idx
	root := b.merkleLayers[0][idx]
	for i := 0; i < len(layers)-1; i++ {
		isLeft := currentIndex%2 == 0
		neighborIdx := currentIndex ^ 1

		neighbor := make([]byte, 32)
		if layers[i] != nil && len(layers[i]) != 0 && neighborIdx < len(layers[i]) {
			neighbor = layers[i][neighborIdx]
		}
		if isLeft {
			parentHash := hashFunc(append(root, neighbor...))
			root = parentHash[:]
		} else {
			parentHash := hashFunc(append(neighbor, root...))
			root = parentHash[:]
		}
		parentIdx := currentIndex / 2
		// Update the cached layers at the parent index.
		layers[i+1][parentIdx] = root
		currentIndex = parentIdx
	}
	b.merkleLayers = layers
}

func (b *BeaconState) markFieldAsDirty(field stateTypes.FieldIndex) {
	b.dirtyFields[field] = true
}

// addDirtyIndices adds the relevant dirty field indices, so that they
// can be recomputed.
func (b *BeaconState) addDirtyIndices(index stateTypes.FieldIndex, indices []uint64) {
	if b.rebuildTrie[index] {
		return
	}
	totalIndicesLen := len(b.dirtyIndices[index]) + len(indices)
	if totalIndicesLen > indicesLimit {
		b.rebuildTrie[index] = true
		b.dirtyIndices[index] = []uint64{}
	} else {
		b.dirtyIndices[index] = append(b.dirtyIndices[index], indices...)
	}
}
