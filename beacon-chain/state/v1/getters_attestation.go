package v1

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// PreviousEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) PreviousEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	if !b.hasInnerState() {
		return nil, nil
	}
	if b.state.PreviousEpochAttestations == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.previousEpochAttestations(), nil
}

// previousEpochAttestations corresponding to blocks on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) previousEpochAttestations() []*ethpb.PendingAttestation {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyPendingAttestationSlice(b.state.PreviousEpochAttestations)
}

// CurrentEpochAttestations corresponding to blocks on the beacon chain.
func (b *BeaconState) CurrentEpochAttestations() ([]*ethpb.PendingAttestation, error) {
	if !b.hasInnerState() {
		return nil, nil
	}
	if b.state.CurrentEpochAttestations == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.currentEpochAttestations(), nil
}

// currentEpochAttestations corresponding to blocks on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) currentEpochAttestations() []*ethpb.PendingAttestation {
	if !b.hasInnerState() {
		return nil
	}

	return ethpb.CopyPendingAttestationSlice(b.state.CurrentEpochAttestations)
}

func (h *stateRootHasher) epochAttestationsRoot(atts []*ethpb.PendingAttestation) ([32]byte, error) {
	max := uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().MaxAttestations
	if uint64(len(atts)) > max {
		return [32]byte{}, fmt.Errorf("epoch attestation exceeds max length %d", max)
	}

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

func (h *stateRootHasher) pendingAttestationRoot(hasher htrutils.HashFn, att *ethpb.PendingAttestation) ([32]byte, error) {
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
