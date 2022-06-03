package blocks

import (
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"google.golang.org/protobuf/proto"
)

// Proto returns the underlying protobuf signed beacon block.
func (b *SignedBeaconBlock) Proto() (proto.Message, error) {
	if b == nil {
		return nil, errNilBlock
	}

	blockMessage, err := b.block.Proto()
	if err != nil {
		return nil, err
	}

	switch b.version {
	case version.Phase0:
		block, ok := blockMessage.(*eth.BeaconBlock)
		if !ok {
			return nil, errors.Wrap(err, incorrectBlockVersion)
		}
		return &eth.SignedBeaconBlock{
			Block:     block,
			Signature: b.signature,
		}, nil
	case version.Altair:
		block, ok := blockMessage.(*eth.BeaconBlockAltair)
		if !ok {
			return nil, errors.Wrap(err, incorrectBlockVersion)
		}
		return &eth.SignedBeaconBlockAltair{
			Block:     block,
			Signature: b.signature,
		}, nil
	case version.Bellatrix:
		block, ok := blockMessage.(*eth.BeaconBlockBellatrix)
		if !ok {
			return nil, errors.Wrap(err, incorrectBlockVersion)
		}
		return &eth.SignedBeaconBlockBellatrix{
			Block:     block,
			Signature: b.signature,
		}, nil
	case version.BellatrixBlind:
		block, ok := blockMessage.(*eth.BlindedBeaconBlockBellatrix)
		if !ok {
			return nil, errors.Wrap(err, incorrectBlockVersion)
		}
		return &eth.SignedBlindedBeaconBlockBellatrix{
			Block:     block,
			Signature: b.signature,
		}, nil
	default:
		return nil, errors.New("unsupported signed beacon block version")
	}
}

// Proto returns the underlying protobuf beacon block.
func (b *BeaconBlock) Proto() (proto.Message, error) {
	if b == nil {
		return nil, errNilBlock
	}

	bodyMessage, err := b.body.Proto()
	if err != nil {
		return nil, err
	}

	switch b.version {
	case version.Phase0:
		body, ok := bodyMessage.(*eth.BeaconBlockBody)
		if !ok {
			return nil, errors.Wrap(err, incorrectBodyVersion)
		}
		return &eth.BeaconBlock{
			Slot:          b.slot,
			ProposerIndex: b.proposerIndex,
			ParentRoot:    b.parentRoot,
			StateRoot:     b.stateRoot,
			Body:          body,
		}, nil
	case version.Altair:
		body, ok := bodyMessage.(*eth.BeaconBlockBodyAltair)
		if !ok {
			return nil, errors.Wrap(err, incorrectBodyVersion)
		}
		return &eth.BeaconBlockAltair{
			Slot:          b.slot,
			ProposerIndex: b.proposerIndex,
			ParentRoot:    b.parentRoot,
			StateRoot:     b.stateRoot,
			Body:          body,
		}, nil
	case version.Bellatrix:
		body, ok := bodyMessage.(*eth.BeaconBlockBodyBellatrix)
		if !ok {
			return nil, errors.Wrap(err, incorrectBodyVersion)
		}
		return &eth.BeaconBlockBellatrix{
			Slot:          b.slot,
			ProposerIndex: b.proposerIndex,
			ParentRoot:    b.parentRoot,
			StateRoot:     b.stateRoot,
			Body:          body,
		}, nil
	case version.BellatrixBlind:
		body, ok := bodyMessage.(*eth.BlindedBeaconBlockBodyBellatrix)
		if !ok {
			return nil, errors.Wrap(err, incorrectBodyVersion)
		}
		return &eth.BlindedBeaconBlockBellatrix{
			Slot:          b.slot,
			ProposerIndex: b.proposerIndex,
			ParentRoot:    b.parentRoot,
			StateRoot:     b.stateRoot,
			Body:          body,
		}, nil
	default:
		return nil, errors.New("unsupported beacon block version")
	}
}

// Proto returns the underlying protobuf beacon block body.
func (b *BeaconBlockBody) Proto() (proto.Message, error) {
	if b == nil {
		return nil, errNilBody
	}

	switch b.version {
	case version.Phase0:
		return &eth.BeaconBlockBody{
			RandaoReveal:      b.randaoReveal,
			Eth1Data:          b.eth1Data,
			Graffiti:          b.graffiti,
			ProposerSlashings: b.proposerSlashings,
			AttesterSlashings: b.attesterSlashings,
			Attestations:      b.attestations,
			Deposits:          b.deposits,
			VoluntaryExits:    b.voluntaryExits,
		}, nil
	case version.Altair:
		return &eth.BeaconBlockBodyAltair{
			RandaoReveal:      b.randaoReveal,
			Eth1Data:          b.eth1Data,
			Graffiti:          b.graffiti,
			ProposerSlashings: b.proposerSlashings,
			AttesterSlashings: b.attesterSlashings,
			Attestations:      b.attestations,
			Deposits:          b.deposits,
			VoluntaryExits:    b.voluntaryExits,
			SyncAggregate:     b.syncAggregate,
		}, nil
	case version.Bellatrix:
		return &eth.BeaconBlockBodyBellatrix{
			RandaoReveal:      b.randaoReveal,
			Eth1Data:          b.eth1Data,
			Graffiti:          b.graffiti,
			ProposerSlashings: b.proposerSlashings,
			AttesterSlashings: b.attesterSlashings,
			Attestations:      b.attestations,
			Deposits:          b.deposits,
			VoluntaryExits:    b.voluntaryExits,
			SyncAggregate:     b.syncAggregate,
			ExecutionPayload:  b.executionPayload,
		}, nil
	case version.BellatrixBlind:
		return &eth.BlindedBeaconBlockBodyBellatrix{
			RandaoReveal:           b.randaoReveal,
			Eth1Data:               b.eth1Data,
			Graffiti:               b.graffiti,
			ProposerSlashings:      b.proposerSlashings,
			AttesterSlashings:      b.attesterSlashings,
			Attestations:           b.attestations,
			Deposits:               b.deposits,
			VoluntaryExits:         b.voluntaryExits,
			SyncAggregate:          b.syncAggregate,
			ExecutionPayloadHeader: b.executionPayloadHeader,
		}, nil
	default:
		return nil, errors.New("unsupported beacon block body version")
	}
}

func initSignedBlockFromProtoPhase0(pb *eth.SignedBeaconBlock) (*SignedBeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb, ok := proto.Clone(pb).(*eth.SignedBeaconBlock)
	if !ok {
		return nil, errCloningFailed
	}
	block, err := initBlockFromProtoPhase0(pb.Block)
	if err != nil {
		return nil, err
	}
	b := &SignedBeaconBlock{
		version:   version.Phase0,
		block:     block,
		signature: pb.Signature,
	}
	return b, nil
}

func initSignedBlockFromProtoAltair(pb *eth.SignedBeaconBlockAltair) (*SignedBeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb, ok := proto.Clone(pb).(*eth.SignedBeaconBlockAltair)
	if !ok {
		return nil, errCloningFailed
	}
	block, err := initBlockFromProtoAltair(pb.Block)
	if err != nil {
		return nil, err
	}
	b := &SignedBeaconBlock{
		version:   version.Altair,
		block:     block,
		signature: pb.Signature,
	}
	return b, nil
}

func initSignedBlockFromProtoBellatrix(pb *eth.SignedBeaconBlockBellatrix) (*SignedBeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb, ok := proto.Clone(pb).(*eth.SignedBeaconBlockBellatrix)
	if !ok {
		return nil, errCloningFailed
	}
	block, err := initBlockFromProtoBellatrix(pb.Block)
	if err != nil {
		return nil, err
	}
	b := &SignedBeaconBlock{
		version:   version.Bellatrix,
		block:     block,
		signature: pb.Signature,
	}
	return b, nil
}

func initBlindedSignedBlockFromProtoBellatrix(pb *eth.SignedBlindedBeaconBlockBellatrix) (*SignedBeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb, ok := proto.Clone(pb).(*eth.SignedBlindedBeaconBlockBellatrix)
	if !ok {
		return nil, errCloningFailed
	}
	block, err := initBlindedBlockFromProtoBellatrix(pb.Block)
	if err != nil {
		return nil, err
	}
	b := &SignedBeaconBlock{
		version:   version.BellatrixBlind,
		block:     block,
		signature: pb.Signature,
	}
	return b, nil
}

func initBlockFromProtoPhase0(pb *eth.BeaconBlock) (*BeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb, ok := proto.Clone(pb).(*eth.BeaconBlock)
	if !ok {
		return nil, errCloningFailed
	}
	body, err := initBlockBodyFromProtoPhase0(pb.Body)
	if err != nil {
		return nil, err
	}
	b := &BeaconBlock{
		version:       version.Phase0,
		slot:          pb.Slot,
		proposerIndex: pb.ProposerIndex,
		parentRoot:    pb.ParentRoot,
		stateRoot:     pb.StateRoot,
		body:          body,
	}
	return b, nil
}

func initBlockFromProtoAltair(pb *eth.BeaconBlockAltair) (*BeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb, ok := proto.Clone(pb).(*eth.BeaconBlockAltair)
	if !ok {
		return nil, errCloningFailed
	}
	body, err := initBlockBodyFromProtoAltair(pb.Body)
	if err != nil {
		return nil, err
	}
	b := &BeaconBlock{
		version:       version.Altair,
		slot:          pb.Slot,
		proposerIndex: pb.ProposerIndex,
		parentRoot:    pb.ParentRoot,
		stateRoot:     pb.StateRoot,
		body:          body,
	}
	return b, nil
}

func initBlockFromProtoBellatrix(pb *eth.BeaconBlockBellatrix) (*BeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb, ok := proto.Clone(pb).(*eth.BeaconBlockBellatrix)
	if !ok {
		return nil, errCloningFailed
	}
	body, err := initBlockBodyFromProtoBellatrix(pb.Body)
	if err != nil {
		return nil, err
	}
	b := &BeaconBlock{
		version:       version.Bellatrix,
		slot:          pb.Slot,
		proposerIndex: pb.ProposerIndex,
		parentRoot:    pb.ParentRoot,
		stateRoot:     pb.StateRoot,
		body:          body,
	}
	return b, nil
}

func initBlindedBlockFromProtoBellatrix(pb *eth.BlindedBeaconBlockBellatrix) (*BeaconBlock, error) {
	if pb == nil {
		return nil, errNilBlock
	}

	pb, ok := proto.Clone(pb).(*eth.BlindedBeaconBlockBellatrix)
	if !ok {
		return nil, errCloningFailed
	}
	body, err := initBlindedBlockBodyFromProtoBellatrix(pb.Body)
	if err != nil {
		return nil, err
	}
	b := &BeaconBlock{
		version:       version.BellatrixBlind,
		slot:          pb.Slot,
		proposerIndex: pb.ProposerIndex,
		parentRoot:    pb.ParentRoot,
		stateRoot:     pb.StateRoot,
		body:          body,
	}
	return b, nil
}

func initBlockBodyFromProtoPhase0(pb *eth.BeaconBlockBody) (*BeaconBlockBody, error) {
	if pb == nil {
		return nil, errNilBody
	}

	pb, ok := proto.Clone(pb).(*eth.BeaconBlockBody)
	if !ok {
		return nil, errCloningFailed
	}
	b := &BeaconBlockBody{
		version:           version.Phase0,
		randaoReveal:      pb.RandaoReveal,
		eth1Data:          pb.Eth1Data,
		graffiti:          pb.Graffiti,
		proposerSlashings: pb.ProposerSlashings,
		attesterSlashings: pb.AttesterSlashings,
		attestations:      pb.Attestations,
		deposits:          pb.Deposits,
		voluntaryExits:    pb.VoluntaryExits,
	}
	return b, nil
}

func initBlockBodyFromProtoAltair(pb *eth.BeaconBlockBodyAltair) (*BeaconBlockBody, error) {
	if pb == nil {
		return nil, errNilBody
	}

	pb, ok := proto.Clone(pb).(*eth.BeaconBlockBodyAltair)
	if !ok {
		return nil, errCloningFailed
	}
	b := &BeaconBlockBody{
		version:           version.Altair,
		randaoReveal:      pb.RandaoReveal,
		eth1Data:          pb.Eth1Data,
		graffiti:          pb.Graffiti,
		proposerSlashings: pb.ProposerSlashings,
		attesterSlashings: pb.AttesterSlashings,
		attestations:      pb.Attestations,
		deposits:          pb.Deposits,
		voluntaryExits:    pb.VoluntaryExits,
		syncAggregate:     pb.SyncAggregate,
	}
	return b, nil
}

func initBlockBodyFromProtoBellatrix(pb *eth.BeaconBlockBodyBellatrix) (*BeaconBlockBody, error) {
	if pb == nil {
		return nil, errNilBody
	}

	pb, ok := proto.Clone(pb).(*eth.BeaconBlockBodyBellatrix)
	if !ok {
		return nil, errCloningFailed
	}
	b := &BeaconBlockBody{
		version:           version.Bellatrix,
		randaoReveal:      pb.RandaoReveal,
		eth1Data:          pb.Eth1Data,
		graffiti:          pb.Graffiti,
		proposerSlashings: pb.ProposerSlashings,
		attesterSlashings: pb.AttesterSlashings,
		attestations:      pb.Attestations,
		deposits:          pb.Deposits,
		voluntaryExits:    pb.VoluntaryExits,
		syncAggregate:     pb.SyncAggregate,
		executionPayload:  pb.ExecutionPayload,
	}
	return b, nil
}

func initBlindedBlockBodyFromProtoBellatrix(pb *eth.BlindedBeaconBlockBodyBellatrix) (*BeaconBlockBody, error) {
	if pb == nil {
		return nil, errNilBody
	}

	pb, ok := proto.Clone(pb).(*eth.BlindedBeaconBlockBodyBellatrix)
	if !ok {
		return nil, errCloningFailed
	}
	b := &BeaconBlockBody{
		version:                version.BellatrixBlind,
		randaoReveal:           pb.RandaoReveal,
		eth1Data:               pb.Eth1Data,
		graffiti:               pb.Graffiti,
		proposerSlashings:      pb.ProposerSlashings,
		attesterSlashings:      pb.AttesterSlashings,
		attestations:           pb.Attestations,
		deposits:               pb.Deposits,
		voluntaryExits:         pb.VoluntaryExits,
		syncAggregate:          pb.SyncAggregate,
		executionPayloadHeader: pb.ExecutionPayloadHeader,
	}
	return b, nil
}
