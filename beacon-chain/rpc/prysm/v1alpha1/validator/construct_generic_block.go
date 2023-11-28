package validator

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"google.golang.org/protobuf/proto"
)

// constructGenericBeaconBlock constructs a `GenericBeaconBlock` based on the block version and other parameters.
func (vs *Server) constructGenericBeaconBlock(sBlk interfaces.SignedBeaconBlock, blindBlobs []*ethpb.BlindedBlobSidecar) (*ethpb.GenericBeaconBlock, error) {
	if sBlk == nil || sBlk.Block() == nil {
		return nil, fmt.Errorf("block cannot be nil")
	}

	blockProto, err := sBlk.Block().Proto()
	if err != nil {
		return nil, err
	}

	isBlinded := sBlk.IsBlinded()
	payloadValue := sBlk.ValueInGwei()

	switch sBlk.Version() {
	case version.Deneb:
		return vs.constructDenebBlock(blockProto, isBlinded, payloadValue, blindBlobs), nil
	case version.Capella:
		return vs.constructCapellaBlock(blockProto, isBlinded, payloadValue), nil
	case version.Bellatrix:
		return vs.constructBellatrixBlock(blockProto, isBlinded, payloadValue), nil
	case version.Altair:
		return vs.constructAltairBlock(blockProto), nil
	case version.Phase0:
		return vs.constructPhase0Block(blockProto), nil
	default:
		return nil, fmt.Errorf("unknown block version: %d", sBlk.Version())
	}
}

// Helper functions for constructing blocks for each version
func (vs *Server) constructDenebBlock(blockProto proto.Message, isBlinded bool, payloadValue uint64, blindBlobs []*ethpb.BlindedBlobSidecar) *ethpb.GenericBeaconBlock {
	if isBlinded {
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedDeneb{BlindedDeneb: &ethpb.BlindedBeaconBlockAndBlobsDeneb{Block: blockProto.(*ethpb.BlindedBeaconBlockDeneb), Blobs: blindBlobs}}, IsBlinded: true, PayloadValue: payloadValue}
	}
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Deneb{Deneb: blockProto.(*ethpb.BeaconBlockDeneb)}, IsBlinded: false, PayloadValue: payloadValue}
}

func (vs *Server) constructCapellaBlock(pb proto.Message, isBlinded bool, payloadValue uint64) *ethpb.GenericBeaconBlock {
	if isBlinded {
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedCapella{BlindedCapella: pb.(*ethpb.BlindedBeaconBlockCapella)}, IsBlinded: true, PayloadValue: payloadValue}
	}
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Capella{Capella: pb.(*ethpb.BeaconBlockCapella)}, IsBlinded: false, PayloadValue: payloadValue}
}

func (vs *Server) constructBellatrixBlock(pb proto.Message, isBlinded bool, payloadValue uint64) *ethpb.GenericBeaconBlock {
	if isBlinded {
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: pb.(*ethpb.BlindedBeaconBlockBellatrix)}, IsBlinded: true, PayloadValue: payloadValue}
	}
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Bellatrix{Bellatrix: pb.(*ethpb.BeaconBlockBellatrix)}, IsBlinded: false, PayloadValue: payloadValue}
}

func (vs *Server) constructAltairBlock(pb proto.Message) *ethpb.GenericBeaconBlock {
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Altair{Altair: pb.(*ethpb.BeaconBlockAltair)}}
}

func (vs *Server) constructPhase0Block(pb proto.Message) *ethpb.GenericBeaconBlock {
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Phase0{Phase0: pb.(*ethpb.BeaconBlock)}}
}
