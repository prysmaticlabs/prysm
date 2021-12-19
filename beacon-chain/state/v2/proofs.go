package v2

import (
	"encoding/binary"

	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

const (
	finalizedRootIndex = 105
)

// FinalizedRootGeneralizedIndex for the Altair beacon state.
func (b *BeaconState) FinalizedRootGeneralizedIndex() (int, error) {
	return finalizedRootIndex, nil
}

// ProveFinalizedRoot crafts a Merkle proof for the finalized root
// contained within the finalized checkpoint of a beacon state.
func (b *BeaconState) ProveFinalizedRoot() ([][]byte, error) {
	cpt := b.state.FinalizedCheckpoint
	epochBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBuf, uint64(cpt.Epoch))
	epochRoot := bytesutil.ToBytes32(epochBuf)
	proof := make([][]byte, 0)
	proof = append(proof, epochRoot[:])

	// The merkle tree structure looks as follows:
	// [[r1, r2, r3, r4], [parent1, parent2], [root]]
	layers := b.merkleLayers
	currentIndex := finalizedCheckpoint // Start at the finalized checkpoint root generalized index.
	for i := 0; i < len(layers)-1; i++ {
		neighborIdx := currentIndex ^ 1
		neighbor := layers[i][neighborIdx]
		proof = append(proof, neighbor)
		currentIndex = currentIndex / 2
	}
	return proof, nil
}
