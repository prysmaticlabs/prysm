package v2

import (
	"encoding/binary"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

const (
	finalizedRootIndex = 105 // Precomputed value for Altair.
)

// FinalizedRootGeneralizedIndex for the Altair beacon state.
func FinalizedRootGeneralizedIndex() int {
	return finalizedRootIndex
}

func (b *BeaconState) ProveCurrentSyncCommittee() ([][]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return proofFromMerkleLayers(b.merkleLayers, currentSyncCommittee), nil
}

func (b *BeaconState) ProveNextSyncCommittee() ([][]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return proofFromMerkleLayers(b.merkleLayers, nextSyncCommittee), nil
}

// ProveFinalizedRoot crafts a Merkle proof for the finalized root
// contained within the finalized checkpoint of a beacon state.
func (b *BeaconState) ProveFinalizedRoot() ([][]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	cpt := b.state.FinalizedCheckpoint
	// The epoch field of a finalized checkpoint is the neighbor
	// index of the finalized root field in its Merkle tree representation
	// of the checkpoint. This neighbor is the first element added to the proof.
	epochBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBuf, uint64(cpt.Epoch))
	epochRoot := bytesutil.ToBytes32(epochBuf)
	proof := make([][]byte, 0)
	proof = append(proof, epochRoot[:])
	branch := proofFromMerkleLayers(b.merkleLayers, finalizedCheckpoint)
	proof = append(proof, branch...)
	return proof, nil
}

// Creates a proof starting at the leaf index of the state Merkle layers.
// Important: caller should acquire a read-lock before passing in the state Merkle layers
// slice into this function.
func proofFromMerkleLayers(layers [][][]byte, startingLeafIndex types.FieldIndex) [][]byte {
	// The merkle tree structure looks as follows:
	// [[r1, r2, r3, r4], [parent1, parent2], [root]]
	proof := make([][]byte, 0)
	currentIndex := startingLeafIndex
	for i := 0; i < len(layers)-1; i++ {
		neighborIdx := currentIndex ^ 1
		neighbor := layers[i][neighborIdx]
		proof = append(proof, neighbor)
		currentIndex = currentIndex / 2
	}
	return proof
}
