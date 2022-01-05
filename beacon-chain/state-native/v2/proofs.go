package v2

import (
	"encoding/binary"

	"github.com/prysmaticlabs/prysm/beacon-chain/state-native/fieldtrie"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

const (
	finalizedRootIndex = uint64(105) // Precomputed value.
)

// FinalizedRootGeneralizedIndex for the beacon state.
func FinalizedRootGeneralizedIndex() uint64 {
	return finalizedRootIndex
}

// CurrentSyncCommitteeGeneralizedIndex for the beacon state.
func CurrentSyncCommitteeGeneralizedIndex() uint64 {
	return uint64(currentSyncCommittee)
}

// NextSyncCommitteeGeneralizedIndex for the beacon state.
func NextSyncCommitteeGeneralizedIndex() uint64 {
	return uint64(nextSyncCommittee)
}

// CurrentSyncCommitteeProof from the state's Merkle trie representation.
func (b *BeaconState) CurrentSyncCommitteeProof() ([][]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return fieldtrie.ProofFromMerkleLayers(b.merkleLayers, currentSyncCommittee), nil
}

// NextSyncCommitteeProof from the state's Merkle trie representation.
func (b *BeaconState) NextSyncCommitteeProof() ([][]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return fieldtrie.ProofFromMerkleLayers(b.merkleLayers, nextSyncCommittee), nil
}

// FinalizedRootProof crafts a Merkle proof for the finalized root
// contained within the finalized checkpoint of a beacon state.
func (b *BeaconState) FinalizedRootProof() ([][]byte, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	cpt := b.finalizedCheckpoint
	// The epoch field of a finalized checkpoint is the neighbor
	// index of the finalized root field in its Merkle tree representation
	// of the checkpoint. This neighbor is the first element added to the proof.
	epochBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBuf, uint64(cpt.Epoch))
	epochRoot := bytesutil.ToBytes32(epochBuf)
	proof := make([][]byte, 0)
	proof = append(proof, epochRoot[:])
	branch := fieldtrie.ProofFromMerkleLayers(b.merkleLayers, finalizedCheckpoint)
	proof = append(proof, branch...)
	return proof, nil
}
