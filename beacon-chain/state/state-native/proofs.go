package state_native

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/state/fieldtrie"
	nativetypes "github.com/prysmaticlabs/prysm/beacon-chain/state/state-native/types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

const (
	finalizedRootIndex = uint64(105) // Precomputed value.
)

// FinalizedRootGeneralizedIndex for the beacon state.
func FinalizedRootGeneralizedIndex() uint64 {
	return finalizedRootIndex
}

// CurrentSyncCommitteeGeneralizedIndex for the beacon state.
func (b *BeaconState) CurrentSyncCommitteeGeneralizedIndex() (uint64, error) {
	if b.version == version.Phase0 {
		return 0, fmt.Errorf("CurrentSyncCommitteeGeneralizedIndex is not supported for %s", version.String(b.version))
	}

	return uint64(b.fieldIndexesRev[nativetypes.CurrentSyncCommittee]), nil
}

// NextSyncCommitteeGeneralizedIndex for the beacon state.
func (b *BeaconState) NextSyncCommitteeGeneralizedIndex() (uint64, error) {
	if b.version == version.Phase0 {
		return 0, fmt.Errorf("NextSyncCommitteeGeneralizedIndex is not supported for %s", version.String(b.version))
	}

	return uint64(b.fieldIndexesRev[nativetypes.NextSyncCommittee]), nil
}

// CurrentSyncCommitteeProof from the state's Merkle trie representation.
func (b *BeaconState) CurrentSyncCommitteeProof(ctx context.Context) ([][]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return nil, fmt.Errorf("CurrentSyncCommitteeProof is not supported for %s", version.String(b.version))
	}

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
	return fieldtrie.ProofFromMerkleLayers(b.merkleLayers, int(b.fieldIndexesRev[nativetypes.CurrentSyncCommittee])), nil
}

// NextSyncCommitteeProof from the state's Merkle trie representation.
func (b *BeaconState) NextSyncCommitteeProof(ctx context.Context) ([][]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return nil, fmt.Errorf("NextSyncCommitteeProof is not supported for %s", version.String(b.version))
	}

	if err := b.initializeMerkleLayers(ctx); err != nil {
		return nil, err
	}
	if err := b.recomputeDirtyFields(ctx); err != nil {
		return nil, err
	}
	return fieldtrie.ProofFromMerkleLayers(b.merkleLayers, int(b.fieldIndexesRev[nativetypes.NextSyncCommittee])), nil
}

// FinalizedRootProof crafts a Merkle proof for the finalized root
// contained within the finalized checkpoint of a beacon state.
func (b *BeaconState) FinalizedRootProof(ctx context.Context) ([][]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.version == version.Phase0 {
		return nil, fmt.Errorf("FinalizedRootProof is not supported for %s", version.String(b.version))
	}

	if err := b.initializeMerkleLayers(ctx); err != nil {
		return nil, err
	}
	if err := b.recomputeDirtyFields(ctx); err != nil {
		return nil, err
	}
	cpt := b.finalizedCheckpointVal()
	// The epoch field of a finalized checkpoint is the neighbor
	// index of the finalized root field in its Merkle tree representation
	// of the checkpoint. This neighbor is the first element added to the proof.
	epochBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBuf, uint64(cpt.Epoch))
	epochRoot := bytesutil.ToBytes32(epochBuf)
	proof := make([][]byte, 0)
	proof = append(proof, epochRoot[:])
	branch := fieldtrie.ProofFromMerkleLayers(b.merkleLayers, int(b.fieldIndexesRev[nativetypes.FinalizedCheckpoint]))
	proof = append(proof, branch...)
	return proof, nil
}
