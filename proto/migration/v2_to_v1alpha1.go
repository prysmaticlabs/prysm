package migration

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// AltairToV1Alpha1SignedBlock converts a v2 SignedBeaconBlockAltair proto to a v1alpha1 proto.
func AltairToV1Alpha1SignedBlock(altairBlk *ethpbv2.SignedBeaconBlockAltair) (*ethpbalpha.SignedBeaconBlockAltair, error) {
	marshaledBlk, err := proto.Marshal(altairBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBeaconBlockAltair{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// BellatrixToV1Alpha1SignedBlock converts a v2 SignedBeaconBlockBellatrix proto to a v1alpha1 proto.
func BellatrixToV1Alpha1SignedBlock(bellatrixBlk *ethpbv2.SignedBeaconBlockBellatrix) (*ethpbalpha.SignedBeaconBlockBellatrix, error) {
	marshaledBlk, err := proto.Marshal(bellatrixBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBeaconBlockBellatrix{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// CapellaToV1Alpha1SignedBlock converts a v2 SignedBeaconBlockCapella proto to a v1alpha1 proto.
func CapellaToV1Alpha1SignedBlock(capellaBlk *ethpbv2.SignedBeaconBlockCapella) (*ethpbalpha.SignedBeaconBlockCapella, error) {
	marshaledBlk, err := proto.Marshal(capellaBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBeaconBlockCapella{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// DenebToV1Alpha1SignedBlock converts a v2 SignedBeaconBlockDeneb proto to a v1alpha1 proto.
func DenebToV1Alpha1SignedBlock(denebBlk *ethpbv2.SignedBeaconBlockDeneb) (*ethpbalpha.SignedBeaconBlockDeneb, error) {
	marshaledBlk, err := proto.Marshal(denebBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBeaconBlockDeneb{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// V2BeaconBlockDenebToV1Alpha1 converts a v2 Deneb beacon block to a v1alpha1
// Deneb block.
func V2BeaconBlockDenebToV1Alpha1(v2block *ethpbv2.BeaconBlockDeneb) (*ethpbalpha.BeaconBlockDeneb, error) {
	marshaledBlk, err := proto.Marshal(v2block)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1block := &ethpbalpha.BeaconBlockDeneb{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1block, nil
}

// BlindedBellatrixToV1Alpha1SignedBlock converts a v2 SignedBlindedBeaconBlockBellatrix proto to a v1alpha1 proto.
func BlindedBellatrixToV1Alpha1SignedBlock(bellatrixBlk *ethpbv2.SignedBlindedBeaconBlockBellatrix) (*ethpbalpha.SignedBlindedBeaconBlockBellatrix, error) {
	marshaledBlk, err := proto.Marshal(bellatrixBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBlindedBeaconBlockBellatrix{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// BlindedCapellaToV1Alpha1SignedBlock converts a v2 SignedBlindedBeaconBlockCapella proto to a v1alpha1 proto.
func BlindedCapellaToV1Alpha1SignedBlock(capellaBlk *ethpbv2.SignedBlindedBeaconBlockCapella) (*ethpbalpha.SignedBlindedBeaconBlockCapella, error) {
	marshaledBlk, err := proto.Marshal(capellaBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBlindedBeaconBlockCapella{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// BlindedDenebToV1Alpha1SignedBlock converts a v2 SignedBlindedBeaconBlockDeneb proto to a v1alpha1 proto.
func BlindedDenebToV1Alpha1SignedBlock(denebBlk *ethpbv2.SignedBlindedBeaconBlockDeneb) (*ethpbalpha.SignedBlindedBeaconBlockDeneb, error) {
	marshaledBlk, err := proto.Marshal(denebBlk)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpbalpha.SignedBlindedBeaconBlockDeneb{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}

// SignedBlindedBlobsToV1Alpha1SignedBlindedBlobs converts an array of v2 SignedBlindedBlobSidecar objects to its v1alpha1 equivalent.
func SignedBlindedBlobsToV1Alpha1SignedBlindedBlobs(sidecars []*ethpbv2.SignedBlindedBlobSidecar) []*ethpbalpha.SignedBlindedBlobSidecar {
	result := make([]*ethpbalpha.SignedBlindedBlobSidecar, len(sidecars))
	for i, sc := range sidecars {
		result[i] = &ethpbalpha.SignedBlindedBlobSidecar{
			Message: &ethpbalpha.BlindedBlobSidecar{
				BlockRoot:       bytesutil.SafeCopyBytes(sc.Message.BlockRoot),
				Index:           sc.Message.Index,
				Slot:            sc.Message.Slot,
				BlockParentRoot: bytesutil.SafeCopyBytes(sc.Message.BlockParentRoot),
				ProposerIndex:   sc.Message.ProposerIndex,
				BlobRoot:        bytesutil.SafeCopyBytes(sc.Message.BlobRoot),
				KzgCommitment:   bytesutil.SafeCopyBytes(sc.Message.KzgCommitment),
				KzgProof:        bytesutil.SafeCopyBytes(sc.Message.KzgProof),
			},
			Signature: bytesutil.SafeCopyBytes(sc.Signature),
		}
	}
	return result
}

// SignedBlobsToV1Alpha1SignedBlobs converts an array of v2 SignedBlobSidecar objects to its v1alpha1 equivalent.
func SignedBlobsToV1Alpha1SignedBlobs(sidecars []*ethpbv2.SignedBlobSidecar) []*ethpbalpha.SignedBlobSidecar {
	result := make([]*ethpbalpha.SignedBlobSidecar, len(sidecars))
	for i, sc := range sidecars {
		result[i] = &ethpbalpha.SignedBlobSidecar{
			Message: &ethpbalpha.BlobSidecar{
				BlockRoot:       bytesutil.SafeCopyBytes(sc.Message.BlockRoot),
				Index:           sc.Message.Index,
				Slot:            sc.Message.Slot,
				BlockParentRoot: bytesutil.SafeCopyBytes(sc.Message.BlockParentRoot),
				ProposerIndex:   sc.Message.ProposerIndex,
				Blob:            bytesutil.SafeCopyBytes(sc.Message.Blob),
				KzgCommitment:   bytesutil.SafeCopyBytes(sc.Message.KzgCommitment),
				KzgProof:        bytesutil.SafeCopyBytes(sc.Message.KzgProof),
			},
			Signature: bytesutil.SafeCopyBytes(sc.Signature),
		}
	}
	return result
}

// DenebBlockContentsToV1Alpha1 converts signed deneb block contents to signed beacon block and blobs deneb
func DenebBlockContentsToV1Alpha1(blockcontents *ethpbv2.SignedBeaconBlockContentsDeneb) (*ethpbalpha.SignedBeaconBlockAndBlobsDeneb, error) {
	block, err := DenebToV1Alpha1SignedBlock(blockcontents.SignedBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert block")
	}
	blobs := SignedBlobsToV1Alpha1SignedBlobs(blockcontents.SignedBlobSidecars)
	return &ethpbalpha.SignedBeaconBlockAndBlobsDeneb{Block: block, Blobs: blobs}, nil
}

// BlindedDenebBlockContentsToV1Alpha1 converts signed blinded deneb block contents to signed blinded beacon block and blobs deneb
func BlindedDenebBlockContentsToV1Alpha1(blockcontents *ethpbv2.SignedBlindedBeaconBlockContentsDeneb) (*ethpbalpha.SignedBlindedBeaconBlockAndBlobsDeneb, error) {
	block, err := BlindedDenebToV1Alpha1SignedBlock(blockcontents.SignedBlindedBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert block")
	}
	blobs := SignedBlindedBlobsToV1Alpha1SignedBlindedBlobs(blockcontents.SignedBlindedBlobSidecars)
	return &ethpbalpha.SignedBlindedBeaconBlockAndBlobsDeneb{SignedBlindedBlock: block, SignedBlindedBlobSidecars: blobs}, nil
}
