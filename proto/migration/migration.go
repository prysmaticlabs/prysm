package migration

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// V1Alpha1BlockToV1BlockHeader converts a v1alpha1 SignedBeaconBlock proto to a v1 SignedBeaconBlockHeader proto.
func V1Alpha1BlockToV1BlockHeader(block *ethpb_alpha.SignedBeaconBlock) (*ethpb.SignedBeaconBlockHeader, error) {
	bodyRoot, err := block.Block.Body.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get body root of block")
	}
	return &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          block.Block.Slot,
			ProposerIndex: block.Block.ProposerIndex,
			ParentRoot:    block.Block.ParentRoot,
			StateRoot:     block.Block.StateRoot,
			BodyRoot:      bodyRoot[:],
		},
		Signature: block.Signature,
	}, nil
}

// V1Alpha1BlockToV1Block converts a v1alpha1 SignedBeaconBlock proto to a v1 proto.
func V1Alpha1ToV1Block(alphaBlk *ethpb_alpha.SignedBeaconBlock) (*ethpb.SignedBeaconBlock, error) {
	marshaledBlk, err := alphaBlk.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1Block := &ethpb.SignedBeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1Block, nil
}

// V1ToV1Alpha1Block converts a v1 SignedBeaconBlock proto to a v1alpha1 proto.
func V1ToV1Alpha1Block(alphaBlk *ethpb.SignedBeaconBlock) (*ethpb_alpha.SignedBeaconBlock, error) {
	marshaledBlk, err := alphaBlk.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal block")
	}
	v1alpha1Block := &ethpb_alpha.SignedBeaconBlock{}
	if err := proto.Unmarshal(marshaledBlk, v1alpha1Block); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal block")
	}
	return v1alpha1Block, nil
}
