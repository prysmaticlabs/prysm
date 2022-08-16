package stateutil

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/crypto/hash"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// RootsArrayHashTreeRoot computes the Merkle root of arrays of 32-byte hashes, such as [64][32]byte
// according to the Simple Serialize specification of Ethereum.
func RootsArrayHashTreeRoot(vals [][]byte, length uint64) ([32]byte, error) {
	return ArraysRoot(vals, length)
}

func EpochAttestationsRoot(atts []*ethpb.PendingAttestation) ([32]byte, error) {
	max := uint64(fieldparams.CurrentEpochAttestationsLength)
	if uint64(len(atts)) > max {
		return [32]byte{}, fmt.Errorf("epoch attestation exceeds max length %d", max)
	}

	hasher := hash.CustomSHA256Hasher()
	roots := make([][32]byte, len(atts))
	for i := 0; i < len(atts); i++ {
		pendingRoot, err := pendingAttestationRoot(hasher, atts[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not attestation merkleization")
		}
		roots[i] = pendingRoot
	}

	attsRootsRoot, err := ssz.BitwiseMerkleize(
		hasher,
		roots,
		uint64(len(roots)),
		fieldparams.CurrentEpochAttestationsLength,
	)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute epoch attestations merkleization")
	}
	attsLenBuf := new(bytes.Buffer)
	if err := binary.Write(attsLenBuf, binary.LittleEndian, uint64(len(atts))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal epoch attestations length")
	}
	// We need to mix in the length of the slice.
	attsLenRoot := make([]byte, 32)
	copy(attsLenRoot, attsLenBuf.Bytes())
	res := ssz.MixInLength(attsRootsRoot, attsLenRoot)
	return res, nil
}

func pendingAttestationRoot(hasher ssz.HashFn, att *ethpb.PendingAttestation) ([32]byte, error) {
	if att == nil {
		return [32]byte{}, errors.New("nil pending attestation")
	}
	return PendingAttRootWithHasher(hasher, att)
}
