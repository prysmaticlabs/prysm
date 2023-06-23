package validator

import (
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// setKzgCommitments sets the KZG commitment on the block.
// Return early if the block version is older than deneb or block slot has not passed deneb epoch.
// Depends on the blk is blind or not, set the KZG commitment from the corresponding bundle.
func setKzgCommitments(blk interfaces.SignedBeaconBlock, bundle *enginev1.BlobsBundle, blindBundle *enginev1.BlindedBlobsBundle) error {
	if blk.Version() < version.Deneb {
		return nil
	}
	slot := blk.Block().Slot()
	if slots.ToEpoch(slot) < params.BeaconConfig().DenebForkEpoch {
		return nil
	}

	if blk.IsBlinded() {
		return blk.SetBlobKzgCommitments(blindBundle.KzgCommitments)
	}

	return blk.SetBlobKzgCommitments(bundle.KzgCommitments)
}

// coverts a blobs bundle to a sidecar format.
func blobsBundleToSidecars(bundle *enginev1.BlobsBundle, blk interfaces.ReadOnlyBeaconBlock) ([]*ethpb.BlobSidecar, error) {
	r, err := blk.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	pr := blk.ParentRoot()

	sidecars := make([]*ethpb.BlobSidecar, len(bundle.Blobs))
	for i := 0; i < len(bundle.Blobs); i++ {
		sidecars[i] = &ethpb.BlobSidecar{
			BlockRoot:       r[:],
			Index:           uint64(i),
			Slot:            blk.Slot(),
			BlockParentRoot: pr[:],
			ProposerIndex:   blk.ProposerIndex(),
			Blob:            bundle.Blobs[i],
			KzgCommitment:   bundle.KzgCommitments[i],
			KzgProof:        bundle.Proofs[i],
		}
	}

	return sidecars, nil
}

// coverts a blinds blobs bundle to a sidecar format.
func blindBlobsBundleToSidecars(bundle *enginev1.BlindedBlobsBundle, blk interfaces.ReadOnlyBeaconBlock) ([]*ethpb.BlindedBlobSidecar, error) {
	r, err := blk.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	pr := blk.ParentRoot()

	sidecars := make([]*ethpb.BlindedBlobSidecar, len(bundle.BlobRoots))
	for i := 0; i < len(bundle.BlobRoots); i++ {
		sidecars[i] = &ethpb.BlindedBlobSidecar{
			BlockRoot:       r[:],
			Index:           uint64(i),
			Slot:            blk.Slot(),
			BlockParentRoot: pr[:],
			ProposerIndex:   blk.ProposerIndex(),
			BlobRoot:        bundle.BlobRoots[i],
			KzgCommitment:   bundle.KzgCommitments[i],
			KzgProof:        bundle.Proofs[i],
		}
	}

	return sidecars, nil
}
