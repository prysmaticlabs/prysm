package v3

import (
	"context"
	"encoding/binary"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/fieldtrie"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
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
func (b *BeaconState) CurrentSyncCommitteeProof(ctx context.Context) ([][]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	// In case the Merkle layers of the trie are not populated, we need
	// to perform some initialization.
	if err := b.initializeMerkleLayers(ctx); err != nil {
		return nil, err
	}
	// Our beacon state uses a "dirty" fields pattern which requires us to
	// recompute branches of the Merkle layers that are marked as dirty.
	if err := b.recomputeDirtyFields(ctx); err != nil {
		return nil, err
	}
	return fieldtrie.ProofFromMerkleLayers(b.merkleLayers, int(currentSyncCommittee)), nil
}

// NextSyncCommitteeProof from the state's Merkle trie representation.
func (b *BeaconState) NextSyncCommitteeProof(ctx context.Context) ([][]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if err := b.initializeMerkleLayers(ctx); err != nil {
		return nil, err
	}
	if err := b.recomputeDirtyFields(ctx); err != nil {
		return nil, err
	}
	return fieldtrie.ProofFromMerkleLayers(b.merkleLayers, int(nextSyncCommittee)), nil
}

// FinalizedRootProof crafts a Merkle proof for the finalized root
// contained within the finalized checkpoint of a beacon state.
func (b *BeaconState) FinalizedRootProof(ctx context.Context) ([][]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if err := b.initializeMerkleLayers(ctx); err != nil {
		return nil, err
	}
	if err := b.recomputeDirtyFields(ctx); err != nil {
		return nil, err
	}
	cpt := b.state.FinalizedCheckpoint
	// The epoch field of a finalized checkpoint is the neighbor
	// index of the finalized root field in its Merkle tree representation
	// of the checkpoint. This neighbor is the first element added to the proof.
	epochBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBuf, uint64(cpt.Epoch))
	epochRoot := bytesutil.ToBytes32(epochBuf)
	proof := make([][]byte, 0)
	proof = append(proof, epochRoot[:])
	branch := fieldtrie.ProofFromMerkleLayers(b.merkleLayers, int(finalizedCheckpoint))
	proof = append(proof, branch...)
	return proof, nil
}
