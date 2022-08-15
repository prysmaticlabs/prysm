package v1

import (
	"context"
	"encoding/binary"

	"github.com/pkg/errors"
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

// CurrentSyncCommitteeProof from the state's Merkle trie representation.
func (*BeaconState) CurrentSyncCommitteeProof(_ context.Context) ([][]byte, error) {
	return nil, errors.New("CurrentSyncCommitteeProof() unsupported for v1 beacon state")
}

// NextSyncCommitteeProof from the state's Merkle trie representation.
func (*BeaconState) NextSyncCommitteeProof(_ context.Context) ([][]byte, error) {
	return nil, errors.New("NextSyncCommitteeProof() unsupported for v1 beacon state")
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
