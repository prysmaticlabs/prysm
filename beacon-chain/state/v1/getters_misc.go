package v1

import (
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// GenesisTime of the beacon state as a uint64.
func (b *BeaconState) GenesisTime() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisTimeInternal()
}

// genesisTime of the beacon state as a uint64.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisTimeInternal() uint64 {
	if !b.hasInnerState() {
		return 0
	}

	return b.genesisTime
}

// GenesisValidatorRoot of the beacon state.
func (b *BeaconState) GenesisValidatorRoot() [32]byte {
	if !b.hasInnerState() {
		return params.BeaconConfig().ZeroHash
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.genesisValidatorRootInternal()
}

// genesisValidatorRootInternal of the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) genesisValidatorRootInternal() [32]byte {
	if !b.hasInnerState() {
		return params.BeaconConfig().ZeroHash
	}

	return b.genesisValidatorsRoot
}

// Version of the beacon state. This method
// is strictly meant to be used without a lock
// internally.
func (b *BeaconState) Version() int {
	return version.Phase0
}

// Slot of the current beacon chain state.
func (b *BeaconState) Slot() types.Slot {
	if !b.hasInnerState() {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slotInternal()
}

// slot of the current beacon chain state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) slotInternal() types.Slot {
	if !b.hasInnerState() {
		return 0
	}

	return b.slot
}

// Fork version of the beacon chain.
func (b *BeaconState) Fork() *ethpb.Fork {
	if !b.hasInnerState() {
		return nil
	}
	if b.fork == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.forkInternal()
}

// forkInternal version of the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) forkInternal() *ethpb.Fork {
	if !b.hasInnerState() {
		return nil
	}
	if b.fork == nil {
		return nil
	}

	prevVersion := make([]byte, len(b.fork.PreviousVersion))
	copy(prevVersion, b.fork.PreviousVersion)
	currVersion := make([]byte, len(b.fork.CurrentVersion))
	copy(currVersion, b.fork.CurrentVersion)
	return &ethpb.Fork{
		PreviousVersion: prevVersion,
		CurrentVersion:  currVersion,
		Epoch:           b.fork.Epoch,
	}
}

// HistoricalRoots based on epochs stored in the beacon state.
func (b *BeaconState) HistoricalRoots() [][32]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.historicalRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.historicalRootsInternal()
}

// historicalRootsInternal based on epochs stored in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) historicalRootsInternal() customtypes.HistoricalRoots {
	if !b.hasInnerState() {
		return nil
	}
	return bytesutil.SafeCopy2d32Bytes(b.historicalRoots)
}

// balancesLength returns the length of the balances slice.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) balancesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.balances == nil {
		return 0
	}

	return len(b.balances)
}

// RootsArrayHashTreeRoot computes the Merkle root of arrays of 32-byte hashes, such as [64][32]byte
// according to the Simple Serialize specification of Ethereum.
func RootsArrayHashTreeRoot(vals [][]byte, length uint64, fieldName string) ([32]byte, error) {
	if features.Get().EnableSSZCache {
		return cachedHasher.arraysRoot(vals, length, fieldName)
	}
	return nocachedHasher.arraysRoot(vals, length, fieldName)
}

func (h *stateRootHasher) arraysRoot(input [][]byte, length uint64, fieldName string) ([32]byte, error) {
	lock.Lock()
	defer lock.Unlock()
	hashFunc := hash.CustomSHA256Hasher()
	if _, ok := layersCache[fieldName]; !ok && h.rootsCache != nil {
		depth := ssz.Depth(length)
		layersCache[fieldName] = make([][][32]byte, depth+1)
	}

	leaves := make([][32]byte, length)
	for i, chunk := range input {
		copy(leaves[i][:], chunk)
	}
	bytesProcessed := 0
	changedIndices := make([]int, 0)
	prevLeaves, ok := leavesCache[fieldName]
	if len(prevLeaves) == 0 || h.rootsCache == nil {
		prevLeaves = leaves
	}

	for i := 0; i < len(leaves); i++ {
		// We check if any items changed since the roots were last recomputed.
		notEqual := leaves[i] != prevLeaves[i]
		if ok && h.rootsCache != nil && notEqual {
			changedIndices = append(changedIndices, i)
		}
		bytesProcessed += 32
	}
	if len(changedIndices) > 0 && h.rootsCache != nil {
		var rt [32]byte
		var err error
		// If indices did change since last computation, we only recompute
		// the modified branches in the cached Merkle tree for this state field.
		chunks := leaves

		// We need to ensure we recompute indices of the Merkle tree which
		// changed in-between calls to this function. This check adds an offset
		// to the recomputed indices to ensure we do so evenly.
		maxChangedIndex := changedIndices[len(changedIndices)-1]
		if maxChangedIndex+2 == len(chunks) && maxChangedIndex%2 != 0 {
			changedIndices = append(changedIndices, maxChangedIndex+1)
		}
		for i := 0; i < len(changedIndices); i++ {
			rt, err = recomputeRoot(changedIndices[i], chunks, fieldName, hashFunc)
			if err != nil {
				return [32]byte{}, err
			}
		}
		leavesCache[fieldName] = chunks
		return rt, nil
	}

	res, err := h.merkleizeWithCache(leaves, length, fieldName, hashFunc)
	if err != nil {
		return [32]byte{}, err
	}
	if h.rootsCache != nil {
		leavesCache[fieldName] = leaves
	}
	return res, nil
}

func recomputeRoot(idx int, chunks [][32]byte, fieldName string, hasher func([]byte) [32]byte) ([32]byte, error) {
	items, ok := layersCache[fieldName]
	if !ok {
		return [32]byte{}, errors.New("could not recompute root as there was no cache found")
	}
	if items == nil {
		return [32]byte{}, errors.New("could not recompute root as there were no items found in the layers cache")
	}
	layers := items
	root := chunks[idx]
	layers[0] = chunks
	// The merkle tree structure looks as follows:
	// [[r1, r2, r3, r4], [parent1, parent2], [root]]
	// Using information about the index which changed, idx, we recompute
	// only its branch up the tree.
	currentIndex := idx
	for i := 0; i < len(layers)-1; i++ {
		isLeft := currentIndex%2 == 0
		neighborIdx := currentIndex ^ 1

		neighbor := [32]byte{}
		if layers[i] != nil && len(layers[i]) != 0 && neighborIdx < len(layers[i]) {
			neighbor = layers[i][neighborIdx]
		}
		if isLeft {
			parentHash := hasher(append(root[:], neighbor[:]...))
			root = parentHash
		} else {
			parentHash := hasher(append(neighbor[:], root[:]...))
			root = parentHash
		}
		parentIdx := currentIndex / 2
		// Update the cached layers at the parent index.
		if len(layers[i+1]) == 0 {
			layers[i+1] = append(layers[i+1], root)
		} else {
			layers[i+1][parentIdx] = root
		}
		currentIndex = parentIdx
	}
	layersCache[fieldName] = layers
	// If there is only a single leaf, we return it (the identity element).
	if len(layers[0]) == 1 {
		return layers[0][0], nil
	}
	return root, nil
}

func (h *stateRootHasher) merkleizeWithCache(leaves [][32]byte, length uint64,
	fieldName string, hasher func([]byte) [32]byte) ([32]byte, error) {
	if len(leaves) == 0 {
		return [32]byte{}, errors.New("zero leaves provided")
	}
	if len(leaves) == 1 {
		return leaves[0], nil
	}
	hashLayer := leaves
	layers := make([][][32]byte, ssz.Depth(length)+1)
	if items, ok := layersCache[fieldName]; ok && h.rootsCache != nil {
		if len(items[0]) == len(leaves) {
			layers = items
		}
	}
	layers[0] = hashLayer
	var err error
	layers, hashLayer, err = stateutil.MerkleizeTrieLeaves(layers, hashLayer, hasher)
	if err != nil {
		return [32]byte{}, err
	}
	root := hashLayer[0]
	if h.rootsCache != nil {
		layersCache[fieldName] = layers
	}
	return root, nil
}
