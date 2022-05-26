package blocks

import (
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"google.golang.org/protobuf/proto"
)

var errNilBlock = errors.New("received nil beacon block")
var errNilBody = errors.New("received nil beacon block body")

const incorrectBlockType = "incorrect beacon block type"
const incorrectBodyType = "incorrect beacon block body type"

func InitSignedBlockFromProtoPhase0(pb *eth.SignedBeaconBlock) (*SingedBeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb = proto.Clone(pb).(*eth.SignedBeaconBlock)
	block, err := InitBlockFromProtoPhase0(pb.Block)
	if err != nil {
		return nil, err
	}
	b := &SingedBeaconBlock{
		version:   version.Phase0,
		blinded:   false,
		Block:     block,
		Signature: pb.Signature,
	}
	return b, nil
}

func InitSignedBlockFromProtoAltair(pb *eth.SignedBeaconBlockAltair) (*SingedBeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb = proto.Clone(pb).(*eth.SignedBeaconBlockAltair)
	block, err := InitBlockFromProtoAltair(pb.Block)
	if err != nil {
		return nil, err
	}
	b := &SingedBeaconBlock{
		version:   version.Altair,
		blinded:   false,
		Block:     block,
		Signature: pb.Signature,
	}
	return b, nil
}

func InitSignedBlockFromProtoBellatrix(pb *eth.SignedBeaconBlockBellatrix) (*SingedBeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb = proto.Clone(pb).(*eth.SignedBeaconBlockBellatrix)
	block, err := InitBlockFromProtoBellatrix(pb.Block)
	if err != nil {
		return nil, err
	}
	b := &SingedBeaconBlock{
		version:   version.Bellatrix,
		blinded:   false,
		Block:     block,
		Signature: pb.Signature,
	}
	return b, nil
}

func InitBlindedSignedBlockFromProtoBellatrix(pb *eth.SignedBlindedBeaconBlockBellatrix) (*SingedBeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb = proto.Clone(pb).(*eth.SignedBlindedBeaconBlockBellatrix)
	block, err := InitBlindedBlockFromProtoBellatrix(pb.Block)
	if err != nil {
		return nil, err
	}
	b := &SingedBeaconBlock{
		version:   version.Bellatrix,
		blinded:   true,
		Block:     block,
		Signature: pb.Signature,
	}
	return b, nil
}

// TODO: InitFromProtoUnsafe?
func InitBlockFromProtoPhase0(pb *eth.BeaconBlock) (*BeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb = proto.Clone(pb).(*eth.BeaconBlock)
	body, err := InitBlockBodyFromProtoPhase0(pb.Body)
	if err != nil {
		return nil, err
	}
	b := &BeaconBlock{
		version:       version.Phase0,
		blinded:       false,
		Slot:          pb.Slot,
		ProposerIndex: pb.ProposerIndex,
		ParentRoot:    pb.ParentRoot,
		StateRoot:     pb.StateRoot,
		Body:          body,
	}
	return b, nil
}

func InitBlockFromProtoAltair(pb *eth.BeaconBlockAltair) (*BeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb = proto.Clone(pb).(*eth.BeaconBlockAltair)
	body, err := InitBlockBodyFromProtoAltair(pb.Body)
	if err != nil {
		return nil, err
	}
	b := &BeaconBlock{
		version:       version.Altair,
		blinded:       false,
		Slot:          pb.Slot,
		ProposerIndex: pb.ProposerIndex,
		ParentRoot:    pb.ParentRoot,
		StateRoot:     pb.StateRoot,
		Body:          body,
	}
	return b, nil
}

func InitBlockFromProtoBellatrix(pb *eth.BeaconBlockBellatrix) (*BeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb = proto.Clone(pb).(*eth.BeaconBlockBellatrix)
	body, err := InitBlockBodyFromProtoBellatrix(pb.Body)
	if err != nil {
		return nil, err
	}
	b := &BeaconBlock{
		version:       version.Bellatrix,
		blinded:       false,
		Slot:          pb.Slot,
		ProposerIndex: pb.ProposerIndex,
		ParentRoot:    pb.ParentRoot,
		StateRoot:     pb.StateRoot,
		Body:          body,
	}
	return b, nil
}

func InitBlindedBlockFromProtoBellatrix(pb *eth.BlindedBeaconBlockBellatrix) (*BeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb = proto.Clone(pb).(*eth.BlindedBeaconBlockBellatrix)
	body, err := InitBlindedBlockBodyFromProtoBellatrix(pb.Body)
	if err != nil {
		return nil, err
	}
	b := &BeaconBlock{
		version:       version.Bellatrix,
		blinded:       true,
		Slot:          pb.Slot,
		ProposerIndex: pb.ProposerIndex,
		ParentRoot:    pb.ParentRoot,
		StateRoot:     pb.StateRoot,
		Body:          body,
	}
	return b, nil
}

func InitBlockBodyFromProtoPhase0(pb *eth.BeaconBlockBody) (*BeaconBlockBody, error) {
	if pb == nil {
		return nil, errNilBody
	}

	pb = proto.Clone(pb).(*eth.BeaconBlockBody)
	b := &BeaconBlockBody{
		version:           version.Phase0,
		blinded:           false,
		RandaoReveal:      pb.RandaoReveal,
		Eth1Data:          pb.Eth1Data,
		Graffiti:          pb.Graffiti,
		ProposerSlashings: pb.ProposerSlashings,
		AttesterSlashings: pb.AttesterSlashings,
		Attestations:      pb.Attestations,
		Deposits:          pb.Deposits,
		VoluntaryExits:    pb.VoluntaryExits,
	}
	return b, nil
}

func InitBlockBodyFromProtoAltair(pb *eth.BeaconBlockBodyAltair) (*BeaconBlockBody, error) {
	if pb == nil {
		return nil, errNilBody
	}

	pb = proto.Clone(pb).(*eth.BeaconBlockBodyAltair)
	b := &BeaconBlockBody{
		version:           version.Altair,
		blinded:           false,
		RandaoReveal:      pb.RandaoReveal,
		Eth1Data:          pb.Eth1Data,
		Graffiti:          pb.Graffiti,
		ProposerSlashings: pb.ProposerSlashings,
		AttesterSlashings: pb.AttesterSlashings,
		Attestations:      pb.Attestations,
		Deposits:          pb.Deposits,
		VoluntaryExits:    pb.VoluntaryExits,
		SyncAggregate:     pb.SyncAggregate,
	}
	return b, nil
}

func InitBlockBodyFromProtoBellatrix(pb *eth.BeaconBlockBodyBellatrix) (*BeaconBlockBody, error) {
	if pb == nil {
		return nil, errNilBody
	}

	pb = proto.Clone(pb).(*eth.BeaconBlockBodyBellatrix)
	b := &BeaconBlockBody{
		version:           version.Bellatrix,
		blinded:           false,
		RandaoReveal:      pb.RandaoReveal,
		Eth1Data:          pb.Eth1Data,
		Graffiti:          pb.Graffiti,
		ProposerSlashings: pb.ProposerSlashings,
		AttesterSlashings: pb.AttesterSlashings,
		Attestations:      pb.Attestations,
		Deposits:          pb.Deposits,
		VoluntaryExits:    pb.VoluntaryExits,
		SyncAggregate:     pb.SyncAggregate,
		ExecutionPayload:  pb.ExecutionPayload,
	}
	return b, nil
}

func InitBlindedBlockBodyFromProtoBellatrix(pb *eth.BlindedBeaconBlockBodyBellatrix) (*BeaconBlockBody, error) {
	if pb == nil {
		return nil, errNilBody
	}

	pb = proto.Clone(pb).(*eth.BlindedBeaconBlockBodyBellatrix)
	b := &BeaconBlockBody{
		version:                version.Bellatrix,
		blinded:                true,
		RandaoReveal:           pb.RandaoReveal,
		Eth1Data:               pb.Eth1Data,
		Graffiti:               pb.Graffiti,
		ProposerSlashings:      pb.ProposerSlashings,
		AttesterSlashings:      pb.AttesterSlashings,
		Attestations:           pb.Attestations,
		Deposits:               pb.Deposits,
		VoluntaryExits:         pb.VoluntaryExits,
		SyncAggregate:          pb.SyncAggregate,
		ExecutionPayloadHeader: pb.ExecutionPayloadHeader,
	}
	return b, nil
}

func (b *SingedBeaconBlock) ToProto() (proto.Message, error) {
	if b == nil {
		return nil, nil
	}

	blockMessage, err := b.Block.ToProto()
	if err != nil {
		return nil, err
	}

	switch b.version {
	case version.Phase0:
		block, ok := blockMessage.(*eth.BeaconBlock)
		if !ok {
			return nil, errors.Wrap(err, incorrectBlockType)
		}
		return &eth.SignedBeaconBlock{
			Block:     block,
			Signature: b.Signature,
		}, nil
	case version.Altair:
		block, ok := blockMessage.(*eth.BeaconBlockAltair)
		if !ok {
			return nil, errors.Wrap(err, incorrectBlockType)
		}
		return &eth.SignedBeaconBlockAltair{
			Block:     block,
			Signature: b.Signature,
		}, nil
	case version.Bellatrix:
		if b.blinded {
			block, ok := blockMessage.(*eth.BlindedBeaconBlockBellatrix)
			if !ok {
				return nil, errors.Wrap(err, incorrectBlockType)
			}
			return &eth.SignedBlindedBeaconBlockBellatrix{
				Block:     block,
				Signature: b.Signature,
			}, nil
		}
		block, ok := blockMessage.(*eth.BeaconBlockBellatrix)
		if !ok {
			return nil, errors.Wrap(err, incorrectBlockType)
		}
		return &eth.SignedBeaconBlockBellatrix{
			Block:     block,
			Signature: b.Signature,
		}, nil
	default:
		return nil, errors.New("unsupported signed beacon block version")
	}
}

func (b *BeaconBlock) ToProto() (proto.Message, error) {
	if b == nil {
		return nil, nil
	}

	bodyMessage, err := b.Body.ToProto()
	if err != nil {
		return nil, err
	}

	switch b.version {
	case version.Phase0:
		body, ok := bodyMessage.(*eth.BeaconBlockBody)
		if !ok {
			return nil, errors.Wrap(err, incorrectBodyType)
		}
		return &eth.BeaconBlock{
			Slot:          b.Slot,
			ProposerIndex: b.ProposerIndex,
			ParentRoot:    b.ParentRoot,
			StateRoot:     b.StateRoot,
			Body:          body,
		}, nil
	case version.Altair:
		body, ok := bodyMessage.(*eth.BeaconBlockBodyAltair)
		if !ok {
			return nil, errors.Wrap(err, incorrectBodyType)
		}
		return &eth.BeaconBlockAltair{
			Slot:          b.Slot,
			ProposerIndex: b.ProposerIndex,
			ParentRoot:    b.ParentRoot,
			StateRoot:     b.StateRoot,
			Body:          body,
		}, nil
	case version.Bellatrix:
		if b.blinded {
			body, ok := bodyMessage.(*eth.BlindedBeaconBlockBodyBellatrix)
			if !ok {
				return nil, errors.Wrap(err, incorrectBodyType)
			}
			return &eth.BlindedBeaconBlockBellatrix{
				Slot:          b.Slot,
				ProposerIndex: b.ProposerIndex,
				ParentRoot:    b.ParentRoot,
				StateRoot:     b.StateRoot,
				Body:          body,
			}, nil
		}
		body, ok := bodyMessage.(*eth.BeaconBlockBodyBellatrix)
		if !ok {
			return nil, errors.Wrap(err, incorrectBodyType)
		}
		return &eth.BeaconBlockBellatrix{
			Slot:          b.Slot,
			ProposerIndex: b.ProposerIndex,
			ParentRoot:    b.ParentRoot,
			StateRoot:     b.StateRoot,
			Body:          body,
		}, nil
	default:
		return nil, errors.New("unsupported beacon block version")
	}
}

func (b *BeaconBlockBody) ToProto() (proto.Message, error) {
	if b == nil {
		return nil, nil
	}

	switch b.version {
	case version.Phase0:
		return &eth.BeaconBlockBody{
			RandaoReveal:      b.RandaoReveal,
			Eth1Data:          b.Eth1Data,
			Graffiti:          b.Graffiti,
			ProposerSlashings: b.ProposerSlashings,
			AttesterSlashings: b.AttesterSlashings,
			Attestations:      b.Attestations,
			Deposits:          b.Deposits,
			VoluntaryExits:    b.VoluntaryExits,
		}, nil
	case version.Altair:
		return &eth.BeaconBlockBodyAltair{
			RandaoReveal:      b.RandaoReveal,
			Eth1Data:          b.Eth1Data,
			Graffiti:          b.Graffiti,
			ProposerSlashings: b.ProposerSlashings,
			AttesterSlashings: b.AttesterSlashings,
			Attestations:      b.Attestations,
			Deposits:          b.Deposits,
			VoluntaryExits:    b.VoluntaryExits,
			SyncAggregate:     b.SyncAggregate,
		}, nil
	case version.Bellatrix:
		if b.blinded {
			return &eth.BlindedBeaconBlockBodyBellatrix{
				RandaoReveal:           b.RandaoReveal,
				Eth1Data:               b.Eth1Data,
				Graffiti:               b.Graffiti,
				ProposerSlashings:      b.ProposerSlashings,
				AttesterSlashings:      b.AttesterSlashings,
				Attestations:           b.Attestations,
				Deposits:               b.Deposits,
				VoluntaryExits:         b.VoluntaryExits,
				SyncAggregate:          b.SyncAggregate,
				ExecutionPayloadHeader: b.ExecutionPayloadHeader,
			}, nil
		}
		return &eth.BeaconBlockBodyBellatrix{
			RandaoReveal:      b.RandaoReveal,
			Eth1Data:          b.Eth1Data,
			Graffiti:          b.Graffiti,
			ProposerSlashings: b.ProposerSlashings,
			AttesterSlashings: b.AttesterSlashings,
			Attestations:      b.Attestations,
			Deposits:          b.Deposits,
			VoluntaryExits:    b.VoluntaryExits,
			SyncAggregate:     b.SyncAggregate,
			ExecutionPayload:  b.ExecutionPayload,
		}, nil
	default:
		return nil, errors.New("unsupported beacon block body version")
	}
}
