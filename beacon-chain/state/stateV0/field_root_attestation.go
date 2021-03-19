package stateV0

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func (h *stateRootHasher) epochAttestationsRoot(atts []*pb.PendingAttestation) ([32]byte, error) {
	hasher := hashutil.CustomSHA256Hasher()
	roots := make([][]byte, len(atts))
	for i := 0; i < len(atts); i++ {
		pendingRoot, err := h.pendingAttestationRoot(hasher, atts[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not attestation merkleization")
		}
		roots[i] = pendingRoot[:]
	}

	attsRootsRoot, err := htrutils.BitwiseMerkleize(
		hasher,
		roots,
		uint64(len(roots)),
		uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxAttestations)),
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
	res := htrutils.MixInLength(attsRootsRoot, attsLenRoot)
	return res, nil
}

func (h *stateRootHasher) pendingAttestationRoot(hasher htrutils.HashFn, att *pb.PendingAttestation) ([32]byte, error) {
	if att == nil {
		return [32]byte{}, errors.New("nil pending attestation")
	}
	// Marshal attestation to determine if it exists in the cache.
	enc := stateutil.PendingAttEncKey(att)

	// Check if it exists in cache:
	if h.rootsCache != nil {
		if found, ok := h.rootsCache.Get(string(enc)); found != nil && ok {
			return found.([32]byte), nil
		}
	}

	res, err := stateutil.PendingAttRootWithHasher(hasher, att)
	if err != nil {
		return [32]byte{}, err
	}
	if h.rootsCache != nil {
		h.rootsCache.Set(string(enc), res, 32)
	}
	return res, nil
}
