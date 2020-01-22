package state

import (
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/protolambda/zssz/merkle"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/stateutil"
)

// BeaconState defines a struct containing utilities for the eth2 chain state, defining
// getters and setters for its respective values and helpful functions such as HashTreeRoot().
type BeaconState struct {
	state        *pbp2p.BeaconState
	lock         sync.RWMutex
	merkleLayers [][][]byte
}

// InitializeFromProto the beacon state from a protobuf representation.
func InitializeFromProto(st *pbp2p.BeaconState) (*BeaconState, error) {
	fieldRoots, err := stateutil.ComputeFieldRoots(st)
	if err != nil {
		return nil, err
	}
	layers := merkleize(fieldRoots)
	return &BeaconState{
		state:        proto.Clone(st).(*pbp2p.BeaconState),
		merkleLayers: layers,
	}, nil
}

// HashTreeRoot of the beacon state retrieves the Merkle root of the trie
// representation of the beacon state based on the eth2 Simple Serialize specification.
func (b *BeaconState) HashTreeRoot() [32]byte {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return bytesutil.ToBytes32(b.merkleLayers[len(b.merkleLayers)-1][0])
}

// Merkleize 32-byte leaves into a Merkle trie for its adequate depth, returning
// the resulting layers of the trie based on the appropriate depth. This function
// pads the leaves to a power-of-two length.
func merkleize(leaves [][]byte) [][][]byte {
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
			hashedChunk := hashutil.Hash(append(currentLayer[i], currentLayer[i+1]...))
			layer = append(layer, hashedChunk[:])
		}
		currentLayer = layer
		layers[i] = currentLayer
		i++
	}
	return layers
}
