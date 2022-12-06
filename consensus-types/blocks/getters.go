package blocks

import (
	"fmt"

	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	field_params "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
)

// BeaconBlockIsNil checks if any composite field of input signed beacon block is nil.
// Access to these nil fields will result in run time panic,
// it is recommended to run these checks as first line of defense.
func BeaconBlockIsNil(b interfaces.SignedBeaconBlock) error {
	if b == nil || b.IsNil() {
		return ErrNilSignedBeaconBlock
	}
	return nil
}

// Signature returns the respective block signature.
func (b *SignedBeaconBlock) Signature() [field_params.BLSSignatureLength]byte {
	return b.signature
}

// SetSignature sets the signature of the signed beacon block.
func (b *SignedBeaconBlock) SetSignature(sig []byte) {
	copy(b.signature[:], sig)
}

// Block returns the underlying beacon block object.
func (b *SignedBeaconBlock) Block() interfaces.BeaconBlock {
	return b.block
}

// SetBlock sets the underlying beacon block object.
func (b *SignedBeaconBlock) SetBlock(blk interfaces.BeaconBlock) error {
	b.block.slot = blk.Slot()
	b.block.parentRoot = blk.ParentRoot()
	b.block.stateRoot = blk.StateRoot()
	b.block.stateRoot = blk.StateRoot()
	b.block.proposerIndex = blk.ProposerIndex()
	b.block.body.randaoReveal = blk.Body().RandaoReveal()
	b.block.body.eth1Data = blk.Body().Eth1Data()
	b.block.body.graffiti = blk.Body().Graffiti()
	b.block.body.proposerSlashings = blk.Body().ProposerSlashings()
	b.block.body.attesterSlashings = blk.Body().AttesterSlashings()
	b.block.body.attestations = blk.Body().Attestations()
	b.block.body.deposits = blk.Body().Deposits()
	b.block.body.voluntaryExits = blk.Body().VoluntaryExits()
	if b.version >= version.Altair {
		syncAggregate, err := blk.Body().SyncAggregate()
		if err != nil {
			return err
		}
		b.block.body.syncAggregate = syncAggregate
	}
	if b.version >= version.Bellatrix {
		executionData, err := blk.Body().Execution()
		if err != nil {
			return err
		}
		if b.block.body.isBlinded {
			b.block.body.executionPayloadHeader = executionData
		} else {
			b.block.body.executionPayload = executionData
		}
	}
	if b.version >= version.Capella {
		changes, err := blk.Body().BLSToExecutionChanges()
		if err != nil {
			return err
		}
		b.block.body.blsToExecutionChanges = changes
	}
	return nil
}

// IsNil checks if the underlying beacon block is nil.
func (b *SignedBeaconBlock) IsNil() bool {
	return b == nil || b.block.IsNil()
}

// Copy performs a deep copy of the signed beacon block object.
func (b *SignedBeaconBlock) Copy() (interfaces.SignedBeaconBlock, error) {
	if b == nil {
		return nil, nil
	}

	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	switch b.version {
	case version.Phase0:
		cp := eth.CopySignedBeaconBlock(pb.(*eth.SignedBeaconBlock))
		return initSignedBlockFromProtoPhase0(cp)
	case version.Altair:
		cp := eth.CopySignedBeaconBlockAltair(pb.(*eth.SignedBeaconBlockAltair))
		return initSignedBlockFromProtoAltair(cp)
	case version.Bellatrix:
		if b.IsBlinded() {
			cp := eth.CopySignedBlindedBeaconBlockBellatrix(pb.(*eth.SignedBlindedBeaconBlockBellatrix))
			return initBlindedSignedBlockFromProtoBellatrix(cp)
		}
		cp := eth.CopySignedBeaconBlockBellatrix(pb.(*eth.SignedBeaconBlockBellatrix))
		return initSignedBlockFromProtoBellatrix(cp)
	case version.Capella:
		if b.IsBlinded() {
			cp := eth.CopySignedBlindedBeaconBlockCapella(pb.(*eth.SignedBlindedBeaconBlockCapella))
			return initBlindedSignedBlockFromProtoCapella(cp)
		}
		cp := eth.CopySignedBeaconBlockCapella(pb.(*eth.SignedBeaconBlockCapella))
		return initSignedBlockFromProtoCapella(cp)
	case version.EIP4844:
		if b.IsBlinded() {
			cp := eth.CopySignedBlindedBeaconBlock4844(pb.(*eth.SignedBlindedBeaconBlock4844))
			return initBlindedSignedBlockFromProto4844(cp)
		}
		cp := eth.CopySignedBeaconBlock4844(pb.(*eth.SignedBeaconBlock4844))
		return initSignedBlockFromProto4844(cp)
	default:
		return nil, errIncorrectBlockVersion
	}
}

// PbGenericBlock returns a generic signed beacon block.
func (b *SignedBeaconBlock) PbGenericBlock() (*eth.GenericSignedBeaconBlock, error) {
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	switch b.version {
	case version.Phase0:
		return &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Phase0{Phase0: pb.(*eth.SignedBeaconBlock)},
		}, nil
	case version.Altair:
		return &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Altair{Altair: pb.(*eth.SignedBeaconBlockAltair)},
		}, nil
	case version.Bellatrix:
		if b.IsBlinded() {
			return &eth.GenericSignedBeaconBlock{
				Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: pb.(*eth.SignedBlindedBeaconBlockBellatrix)},
			}, nil
		}
		return &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Bellatrix{Bellatrix: pb.(*eth.SignedBeaconBlockBellatrix)},
		}, nil
	case version.Capella:
		if b.IsBlinded() {
			return &eth.GenericSignedBeaconBlock{
				Block: &eth.GenericSignedBeaconBlock_BlindedCapella{BlindedCapella: pb.(*eth.SignedBlindedBeaconBlockCapella)},
			}, nil
		}
		return &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Capella{Capella: pb.(*eth.SignedBeaconBlockCapella)},
		}, nil
	case version.EIP4844:
		if b.IsBlinded() {
			return &eth.GenericSignedBeaconBlock{
				Block: &eth.GenericSignedBeaconBlock_Blinded_EIP4844{Blinded_EIP4844: pb.(*eth.SignedBlindedBeaconBlock4844)},
			}, nil
		}
		return &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_EIP4844{EIP4844: pb.(*eth.SignedBeaconBlock4844)},
		}, nil
	default:
		return nil, errIncorrectBlockVersion
	}

}

// PbPhase0Block returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	if b.version != version.Phase0 {
		return nil, ErrNotSupported("PbPhase0Block", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBeaconBlock), nil
}

// PbAltairBlock returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbAltairBlock() (*eth.SignedBeaconBlockAltair, error) {
	if b.version != version.Altair {
		return nil, ErrNotSupported("PbAltairBlock", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBeaconBlockAltair), nil
}

// PbBellatrixBlock returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	if b.version != version.Bellatrix || b.IsBlinded() {
		return nil, ErrNotSupported("PbBellatrixBlock", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBeaconBlockBellatrix), nil
}

// PbBlindedBellatrixBlock returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbBlindedBellatrixBlock() (*eth.SignedBlindedBeaconBlockBellatrix, error) {
	if b.version != version.Bellatrix || !b.IsBlinded() {
		return nil, ErrNotSupported("PbBlindedBellatrixBlock", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBlindedBeaconBlockBellatrix), nil
}

// PbCapellaBlock returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbCapellaBlock() (*eth.SignedBeaconBlockCapella, error) {
	if b.version != version.Capella || b.IsBlinded() {
		return nil, ErrNotSupported("PbCapellaBlock", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBeaconBlockCapella), nil
}

// PbBlindedCapellaBlock returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbBlindedCapellaBlock() (*eth.SignedBlindedBeaconBlockCapella, error) {
	if b.version != version.Capella || !b.IsBlinded() {
		return nil, ErrNotSupported("PbBlindedCapellaBlock", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBlindedBeaconBlockCapella), nil
}

// Pb4844Block returns the underlying protobuf object.
func (b *SignedBeaconBlock) Pb4844Block() (*eth.SignedBeaconBlock4844, error) {
	if b.version != version.EIP4844 || b.IsBlinded() {
		return nil, ErrNotSupported("Pb4844Block", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBeaconBlock4844), nil
}

// PbBlinded4844Block returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbBlinded4844Block() (*eth.SignedBlindedBeaconBlock4844, error) {
	if b.version != version.EIP4844 || !b.IsBlinded() {
		return nil, ErrNotSupported("PbBlinded4844Block", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBlindedBeaconBlock4844), nil
}

// ToBlinded converts a non-blinded block to its blinded equivalent.
func (b *SignedBeaconBlock) ToBlinded() (interfaces.SignedBeaconBlock, error) {
	if b.version < version.Bellatrix {
		return nil, ErrUnsupportedVersion
	}
	if b.IsBlinded() {
		return b, nil
	}
	if b.block.IsNil() {
		return nil, errors.New("cannot convert nil block to blinded format")
	}
	payload, err := b.block.Body().Execution()
	if err != nil {
		return nil, err
	}

	switch p := payload.Proto().(type) {
	case *enginev1.ExecutionPayload:
		header, err := PayloadToHeader(payload)
		if err != nil {
			return nil, err
		}
		return initBlindedSignedBlockFromProtoBellatrix(
			&eth.SignedBlindedBeaconBlockBellatrix{
				Block: &eth.BlindedBeaconBlockBellatrix{
					Slot:          b.block.slot,
					ProposerIndex: b.block.proposerIndex,
					ParentRoot:    b.block.parentRoot[:],
					StateRoot:     b.block.stateRoot[:],
					Body: &eth.BlindedBeaconBlockBodyBellatrix{
						RandaoReveal:           b.block.body.randaoReveal[:],
						Eth1Data:               b.block.body.eth1Data,
						Graffiti:               b.block.body.graffiti[:],
						ProposerSlashings:      b.block.body.proposerSlashings,
						AttesterSlashings:      b.block.body.attesterSlashings,
						Attestations:           b.block.body.attestations,
						Deposits:               b.block.body.deposits,
						VoluntaryExits:         b.block.body.voluntaryExits,
						SyncAggregate:          b.block.body.syncAggregate,
						ExecutionPayloadHeader: header,
					},
				},
				Signature: b.signature[:],
			})
	case *enginev1.ExecutionPayloadCapella:
		header, err := PayloadToHeaderCapella(payload)
		if err != nil {
			return nil, err
		}
		return initBlindedSignedBlockFromProtoCapella(
			&eth.SignedBlindedBeaconBlockCapella{
				Block: &eth.BlindedBeaconBlockCapella{
					Slot:          b.block.slot,
					ProposerIndex: b.block.proposerIndex,
					ParentRoot:    b.block.parentRoot[:],
					StateRoot:     b.block.stateRoot[:],
					Body: &eth.BlindedBeaconBlockBodyCapella{
						RandaoReveal:           b.block.body.randaoReveal[:],
						Eth1Data:               b.block.body.eth1Data,
						Graffiti:               b.block.body.graffiti[:],
						ProposerSlashings:      b.block.body.proposerSlashings,
						AttesterSlashings:      b.block.body.attesterSlashings,
						Attestations:           b.block.body.attestations,
						Deposits:               b.block.body.deposits,
						VoluntaryExits:         b.block.body.voluntaryExits,
						SyncAggregate:          b.block.body.syncAggregate,
						ExecutionPayloadHeader: header,
						BlsToExecutionChanges:  b.block.body.blsToExecutionChanges,
					},
				},
				Signature: b.signature[:],
			})
	case *enginev1.ExecutionPayload4844:
		header, err := PayloadToHeaderEIP4844(payload)
		if err != nil {
			return nil, err
		}
		return initBlindedSignedBlockFromProto4844(
			&eth.SignedBlindedBeaconBlock4844{
				Block: &eth.BlindedBeaconBlock4844{
					Slot:          b.block.slot,
					ProposerIndex: b.block.proposerIndex,
					ParentRoot:    b.block.parentRoot[:],
					StateRoot:     b.block.stateRoot[:],
					Body: &eth.BlindedBeaconBlockBody4844{
						RandaoReveal:           b.block.body.randaoReveal[:],
						Eth1Data:               b.block.body.eth1Data,
						Graffiti:               b.block.body.graffiti[:],
						ProposerSlashings:      b.block.body.proposerSlashings,
						AttesterSlashings:      b.block.body.attesterSlashings,
						Attestations:           b.block.body.attestations,
						Deposits:               b.block.body.deposits,
						VoluntaryExits:         b.block.body.voluntaryExits,
						SyncAggregate:          b.block.body.syncAggregate,
						ExecutionPayloadHeader: header,
						BlsToExecutionChanges:  b.block.body.blsToExecutionChanges,
					},
				},
				Signature: b.signature[:],
			})
	default:
		return nil, fmt.Errorf("%T is not an execution payload header", p)
	}
}

// Version of the underlying protobuf object.
func (b *SignedBeaconBlock) Version() int {
	return b.version
}

func (b *SignedBeaconBlock) IsBlinded() bool {
	return b.block.body.isBlinded
}

// Header converts the underlying protobuf object from blinded block to header format.
func (b *SignedBeaconBlock) Header() (*eth.SignedBeaconBlockHeader, error) {
	if b.IsNil() {
		return nil, errNilBlock
	}
	root, err := b.block.body.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not hash block body")
	}

	return &eth.SignedBeaconBlockHeader{
		Header: &eth.BeaconBlockHeader{
			Slot:          b.block.slot,
			ProposerIndex: b.block.proposerIndex,
			ParentRoot:    b.block.parentRoot[:],
			StateRoot:     b.block.stateRoot[:],
			BodyRoot:      root[:],
		},
		Signature: b.signature[:],
	}, nil
}

// MarshalSSZ marshals the signed beacon block to its relevant ssz form.
func (b *SignedBeaconBlock) MarshalSSZ() ([]byte, error) {
	pb, err := b.Proto()
	if err != nil {
		return []byte{}, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.SignedBeaconBlock).MarshalSSZ()
	case version.Altair:
		return pb.(*eth.SignedBeaconBlockAltair).MarshalSSZ()
	case version.Bellatrix:
		if b.IsBlinded() {
			return pb.(*eth.SignedBlindedBeaconBlockBellatrix).MarshalSSZ()
		}
		return pb.(*eth.SignedBeaconBlockBellatrix).MarshalSSZ()
	case version.Capella:
		if b.IsBlinded() {
			return pb.(*eth.SignedBlindedBeaconBlockCapella).MarshalSSZ()
		}
		return pb.(*eth.SignedBeaconBlockCapella).MarshalSSZ()
	case version.EIP4844:
		if b.IsBlinded() {
			return pb.(*eth.SignedBlindedBeaconBlock4844).MarshalSSZ()
		}
		return pb.(*eth.SignedBeaconBlock4844).MarshalSSZ()
	default:
		return []byte{}, errIncorrectBlockVersion
	}
}

// MarshalSSZTo marshals the signed beacon block's ssz
// form to the provided byte buffer.
func (b *SignedBeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	pb, err := b.Proto()
	if err != nil {
		return []byte{}, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.SignedBeaconBlock).MarshalSSZTo(dst)
	case version.Altair:
		return pb.(*eth.SignedBeaconBlockAltair).MarshalSSZTo(dst)
	case version.Bellatrix:
		if b.IsBlinded() {
			return pb.(*eth.SignedBlindedBeaconBlockBellatrix).MarshalSSZTo(dst)
		}
		return pb.(*eth.SignedBeaconBlockBellatrix).MarshalSSZTo(dst)
	case version.Capella:
		if b.IsBlinded() {
			return pb.(*eth.SignedBlindedBeaconBlockCapella).MarshalSSZTo(dst)
		}
		return pb.(*eth.SignedBeaconBlockCapella).MarshalSSZTo(dst)
	case version.EIP4844:
		if b.IsBlinded() {
			return pb.(*eth.SignedBlindedBeaconBlock4844).MarshalSSZTo(dst)
		}
		return pb.(*eth.SignedBeaconBlock4844).MarshalSSZTo(dst)
	default:
		return []byte{}, errIncorrectBlockVersion
	}
}

// SizeSSZ returns the size of the serialized signed block
//
// WARNING: This function panics. It is required to change the signature
// of fastssz's SizeSSZ() interface function to avoid panicking.
// Changing the signature causes very problematic issues with wealdtech deps.
// For the time being panicking is preferable.
func (b *SignedBeaconBlock) SizeSSZ() int {
	pb, err := b.Proto()
	if err != nil {
		panic(err)
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.SignedBeaconBlock).SizeSSZ()
	case version.Altair:
		return pb.(*eth.SignedBeaconBlockAltair).SizeSSZ()
	case version.Bellatrix:
		if b.IsBlinded() {
			return pb.(*eth.SignedBlindedBeaconBlockBellatrix).SizeSSZ()
		}
		return pb.(*eth.SignedBeaconBlockBellatrix).SizeSSZ()
	case version.Capella:
		if b.IsBlinded() {
			return pb.(*eth.SignedBlindedBeaconBlockCapella).SizeSSZ()
		}
		return pb.(*eth.SignedBeaconBlockCapella).SizeSSZ()
	case version.EIP4844:
		if b.IsBlinded() {
			return pb.(*eth.SignedBlindedBeaconBlock4844).SizeSSZ()
		}
		return pb.(*eth.SignedBeaconBlock4844).SizeSSZ()
	default:
		panic(incorrectBlockVersion)
	}
}

// UnmarshalSSZ unmarshals the signed beacon block from its relevant ssz form.
func (b *SignedBeaconBlock) UnmarshalSSZ(buf []byte) error {
	var newBlock *SignedBeaconBlock
	switch b.version {
	case version.Phase0:
		pb := &eth.SignedBeaconBlock{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newBlock, err = initSignedBlockFromProtoPhase0(pb)
		if err != nil {
			return err
		}
	case version.Altair:
		pb := &eth.SignedBeaconBlockAltair{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newBlock, err = initSignedBlockFromProtoAltair(pb)
		if err != nil {
			return err
		}
	case version.Bellatrix:
		if b.IsBlinded() {
			pb := &eth.SignedBlindedBeaconBlockBellatrix{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initBlindedSignedBlockFromProtoBellatrix(pb)
			if err != nil {
				return err
			}
		} else {
			pb := &eth.SignedBeaconBlockBellatrix{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initSignedBlockFromProtoBellatrix(pb)
			if err != nil {
				return err
			}
		}
	case version.Capella:
		if b.IsBlinded() {
			pb := &eth.SignedBlindedBeaconBlockCapella{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initBlindedSignedBlockFromProtoCapella(pb)
			if err != nil {
				return err
			}
		} else {
			pb := &eth.SignedBeaconBlockCapella{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initSignedBlockFromProtoCapella(pb)
			if err != nil {
				return err
			}
		}
	case version.EIP4844:
		if b.IsBlinded() {
			pb := &eth.SignedBlindedBeaconBlock4844{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initBlindedSignedBlockFromProto4844(pb)
			if err != nil {
				return err
			}
		} else {
			pb := &eth.SignedBeaconBlock4844{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initSignedBlockFromProto4844(pb)
			if err != nil {
				return err
			}
		}
	default:
		return errIncorrectBlockVersion
	}
	*b = *newBlock
	return nil
}

// Slot returns the respective slot of the block.
func (b *BeaconBlock) Slot() types.Slot {
	return b.slot
}

// SetSlot sets the respective slot of the block.
// This function is not thread safe, it is only used during block creation.
func (b *BeaconBlock) SetSlot(slot types.Slot) {
	b.slot = slot
}

// ProposerIndex returns the proposer index of the beacon block.
func (b *BeaconBlock) ProposerIndex() types.ValidatorIndex {
	return b.proposerIndex
}

// SetProposerIndex sets the proposer index of the beacon block.
// This function is not thread safe, it is only used during block creation.
func (b *BeaconBlock) SetProposerIndex(proposerIndex types.ValidatorIndex) {
	b.proposerIndex = proposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (b *BeaconBlock) ParentRoot() [field_params.RootLength]byte {
	return b.parentRoot
}

// SetParentRoot sets the parent root of beacon block.
// This function is not thread safe, it is only used during block creation.
func (b *BeaconBlock) SetParentRoot(parentRoot []byte) {
	copy(b.parentRoot[:], parentRoot)
}

// StateRoot returns the state root of the beacon block.
func (b *BeaconBlock) StateRoot() [field_params.RootLength]byte {
	return b.stateRoot
}

// SetStateRoot sets the state root of the underlying beacon block
// This function is not thread safe, it is only used during block
// proposal to set the state root of a new block
func (b *BeaconBlock) SetStateRoot(root []byte) {
	copy(b.stateRoot[:], root)
}

// Body returns the underlying block body.
func (b *BeaconBlock) Body() interfaces.BeaconBlockBody {
	return b.body
}

// IsNil checks if the beacon block is nil.
func (b *BeaconBlock) IsNil() bool {
	return b == nil || b.Body().IsNil()
}

// IsBlinded checks if the beacon block is a blinded block.
func (b *BeaconBlock) IsBlinded() bool {
	return b.body.isBlinded
}

// SetBlinded sets the blinded flag of the beacon block.
func (b *BeaconBlock) SetBlinded(blinded bool) {
	b.body.isBlinded = blinded
}

// Version of the underlying protobuf object.
func (b *BeaconBlock) Version() int {
	return b.version
}

// HashTreeRoot returns the ssz root of the block.
func (b *BeaconBlock) HashTreeRoot() ([field_params.RootLength]byte, error) {
	pb, err := b.Proto()
	if err != nil {
		return [field_params.RootLength]byte{}, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.BeaconBlock).HashTreeRoot()
	case version.Altair:
		return pb.(*eth.BeaconBlockAltair).HashTreeRoot()
	case version.Bellatrix:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockBellatrix).HashTreeRoot()
		}
		return pb.(*eth.BeaconBlockBellatrix).HashTreeRoot()
	case version.Capella:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockCapella).HashTreeRoot()
		}
		return pb.(*eth.BeaconBlockCapella).HashTreeRoot()
	case version.EIP4844:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlock4844).HashTreeRoot()
		}
		return pb.(*eth.BeaconBlock4844).HashTreeRoot()
	default:
		return [field_params.RootLength]byte{}, errIncorrectBlockVersion
	}
}

// HashTreeRootWith ssz hashes the BeaconBlock object with a hasher.
func (b *BeaconBlock) HashTreeRootWith(h *ssz.Hasher) error {
	pb, err := b.Proto()
	if err != nil {
		return err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.BeaconBlock).HashTreeRootWith(h)
	case version.Altair:
		return pb.(*eth.BeaconBlockAltair).HashTreeRootWith(h)
	case version.Bellatrix:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockBellatrix).HashTreeRootWith(h)
		}
		return pb.(*eth.BeaconBlockBellatrix).HashTreeRootWith(h)
	case version.Capella:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockCapella).HashTreeRootWith(h)
		}
		return pb.(*eth.BeaconBlockCapella).HashTreeRootWith(h)
	case version.EIP4844:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlock4844).HashTreeRootWith(h)
		}
		return pb.(*eth.BeaconBlock4844).HashTreeRootWith(h)
	default:
		return errIncorrectBlockVersion
	}
}

// MarshalSSZ marshals the block into its respective
// ssz form.
func (b *BeaconBlock) MarshalSSZ() ([]byte, error) {
	pb, err := b.Proto()
	if err != nil {
		return []byte{}, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.BeaconBlock).MarshalSSZ()
	case version.Altair:
		return pb.(*eth.BeaconBlockAltair).MarshalSSZ()
	case version.Bellatrix:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockBellatrix).MarshalSSZ()
		}
		return pb.(*eth.BeaconBlockBellatrix).MarshalSSZ()
	case version.Capella:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockCapella).MarshalSSZ()
		}
		return pb.(*eth.BeaconBlockCapella).MarshalSSZ()
	case version.EIP4844:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlock4844).MarshalSSZ()
		}
		return pb.(*eth.BeaconBlock4844).MarshalSSZ()
	default:
		return []byte{}, errIncorrectBlockVersion
	}
}

// MarshalSSZTo marshals the beacon block's ssz
// form to the provided byte buffer.
func (b *BeaconBlock) MarshalSSZTo(dst []byte) ([]byte, error) {
	pb, err := b.Proto()
	if err != nil {
		return []byte{}, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.BeaconBlock).MarshalSSZTo(dst)
	case version.Altair:
		return pb.(*eth.BeaconBlockAltair).MarshalSSZTo(dst)
	case version.Bellatrix:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockBellatrix).MarshalSSZTo(dst)
		}
		return pb.(*eth.BeaconBlockBellatrix).MarshalSSZTo(dst)
	case version.Capella:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockCapella).MarshalSSZTo(dst)
		}
		return pb.(*eth.BeaconBlockCapella).MarshalSSZTo(dst)
	case version.EIP4844:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlock4844).MarshalSSZTo(dst)
		}
		return pb.(*eth.BeaconBlock4844).MarshalSSZTo(dst)
	default:
		return []byte{}, errIncorrectBlockVersion
	}
}

// SizeSSZ returns the size of the serialized block.
//
// WARNING: This function panics. It is required to change the signature
// of fastssz's SizeSSZ() interface function to avoid panicking.
// Changing the signature causes very problematic issues with wealdtech deps.
// For the time being panicking is preferable.
func (b *BeaconBlock) SizeSSZ() int {
	pb, err := b.Proto()
	if err != nil {
		panic(err)
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.BeaconBlock).SizeSSZ()
	case version.Altair:
		return pb.(*eth.BeaconBlockAltair).SizeSSZ()
	case version.Bellatrix:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockBellatrix).SizeSSZ()
		}
		return pb.(*eth.BeaconBlockBellatrix).SizeSSZ()
	case version.Capella:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlockCapella).SizeSSZ()
		}
		return pb.(*eth.BeaconBlockCapella).SizeSSZ()
	case version.EIP4844:
		if b.IsBlinded() {
			return pb.(*eth.BlindedBeaconBlock4844).SizeSSZ()
		}
		return pb.(*eth.BeaconBlock4844).SizeSSZ()
	default:
		panic(incorrectBodyVersion)
	}
}

// UnmarshalSSZ unmarshals the beacon block from its relevant ssz form.
func (b *BeaconBlock) UnmarshalSSZ(buf []byte) error {
	var newBlock *BeaconBlock
	switch b.version {
	case version.Phase0:
		pb := &eth.BeaconBlock{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newBlock, err = initBlockFromProtoPhase0(pb)
		if err != nil {
			return err
		}
	case version.Altair:
		pb := &eth.BeaconBlockAltair{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newBlock, err = initBlockFromProtoAltair(pb)
		if err != nil {
			return err
		}
	case version.Bellatrix:
		if b.IsBlinded() {
			pb := &eth.BlindedBeaconBlockBellatrix{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initBlindedBlockFromProtoBellatrix(pb)
			if err != nil {
				return err
			}
		} else {
			pb := &eth.BeaconBlockBellatrix{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initBlockFromProtoBellatrix(pb)
			if err != nil {
				return err
			}
		}
	case version.Capella:
		if b.IsBlinded() {
			pb := &eth.BlindedBeaconBlockCapella{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initBlindedBlockFromProtoCapella(pb)
			if err != nil {
				return err
			}
		} else {
			pb := &eth.BeaconBlockCapella{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initBlockFromProtoCapella(pb)
			if err != nil {
				return err
			}
		}
	case version.EIP4844:
		if b.IsBlinded() {
			pb := &eth.BlindedBeaconBlock4844{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initBlindedBlockFromProto4844(pb)
			if err != nil {
				return err
			}
		} else {
			pb := &eth.BeaconBlock4844{}
			if err := pb.UnmarshalSSZ(buf); err != nil {
				return err
			}
			var err error
			newBlock, err = initBlockFromProto4844(pb)
			if err != nil {
				return err
			}
		}
	default:
		return errIncorrectBlockVersion
	}
	*b = *newBlock
	return nil
}

// AsSignRequestObject returns the underlying sign request object.
func (b *BeaconBlock) AsSignRequestObject() (validatorpb.SignRequestObject, error) {
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	switch b.version {
	case version.Phase0:
		return &validatorpb.SignRequest_Block{Block: pb.(*eth.BeaconBlock)}, nil
	case version.Altair:
		return &validatorpb.SignRequest_BlockAltair{BlockAltair: pb.(*eth.BeaconBlockAltair)}, nil
	case version.Bellatrix:
		if b.IsBlinded() {
			return &validatorpb.SignRequest_BlindedBlockBellatrix{BlindedBlockBellatrix: pb.(*eth.BlindedBeaconBlockBellatrix)}, nil
		}
		return &validatorpb.SignRequest_BlockBellatrix{BlockBellatrix: pb.(*eth.BeaconBlockBellatrix)}, nil
	case version.Capella:
		if b.IsBlinded() {
			return &validatorpb.SignRequest_BlindedBlockCapella{BlindedBlockCapella: pb.(*eth.BlindedBeaconBlockCapella)}, nil
		}
		return &validatorpb.SignRequest_BlockCapella{BlockCapella: pb.(*eth.BeaconBlockCapella)}, nil
	case version.EIP4844:
		if b.IsBlinded() {
			return &validatorpb.SignRequest_BlindedBlockEip4844{BlindedBlockEip4844: pb.(*eth.BlindedBeaconBlock4844)}, nil
		}
		return &validatorpb.SignRequest_BlockEip4844{BlockEip4844: pb.(*eth.BeaconBlock4844)}, nil
	default:
		return nil, errIncorrectBlockVersion
	}
}

// IsNil checks if the block body is nil.
func (b *BeaconBlockBody) IsNil() bool {
	return b == nil
}

// RandaoReveal returns the randao reveal from the block body.
func (b *BeaconBlockBody) RandaoReveal() [field_params.BLSSignatureLength]byte {
	return b.randaoReveal
}

// SetRandaoReveal sets the randao reveal in the block body.
func (b *BeaconBlockBody) SetRandaoReveal(r []byte) {
	copy(b.randaoReveal[:], r)
}

// Eth1Data returns the eth1 data in the block.
func (b *BeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return b.eth1Data
}

// SetEth1Data sets the eth1 data in the block.
func (b *BeaconBlockBody) SetEth1Data(e *eth.Eth1Data) {
	b.eth1Data = eth.CopyETH1Data(e)
}

// Graffiti returns the graffiti in the block.
func (b *BeaconBlockBody) Graffiti() [field_params.RootLength]byte {
	return b.graffiti
}

// SetGraffiti sets the graffiti in the block.
func (b *BeaconBlockBody) SetGraffiti(g []byte) {
	copy(b.graffiti[:], g)
}

// ProposerSlashings returns the proposer slashings in the block.
func (b *BeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return b.proposerSlashings
}

// SetProposerSlashings sets the proposer slashings in the block.
func (b *BeaconBlockBody) SetProposerSlashings(p []*eth.ProposerSlashing) {
	b.proposerSlashings = eth.CopyProposerSlashings(p)
}

// AttesterSlashings returns the attester slashings in the block.
func (b *BeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return b.attesterSlashings
}

// SetAttesterSlashings sets the attester slashings in the block.
func (b *BeaconBlockBody) SetAttesterSlashings(a []*eth.AttesterSlashing) {
	b.attesterSlashings = eth.CopyAttesterSlashings(a)
}

// Attestations returns the stored attestations in the block.
func (b *BeaconBlockBody) Attestations() []*eth.Attestation {
	return b.attestations
}

// SetAttestations sets the attestations in the block.
func (b *BeaconBlockBody) SetAttestations(a []*eth.Attestation) {
	b.attestations = eth.CopyAttestations(a)
}

// Deposits returns the stored deposits in the block.
func (b *BeaconBlockBody) Deposits() []*eth.Deposit {
	return b.deposits
}

// SetDeposits sets the deposits in the block.
func (b *BeaconBlockBody) SetDeposits(d []*eth.Deposit) {
	b.deposits = eth.CopyDeposits(d)
}

// VoluntaryExits returns the voluntary exits in the block.
func (b *BeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return b.voluntaryExits
}

// SetVoluntaryExits sets the voluntary exits in the block.
func (b *BeaconBlockBody) SetVoluntaryExits(v []*eth.SignedVoluntaryExit) {
	b.voluntaryExits = eth.CopySignedVoluntaryExits(v)
}

// SyncAggregate returns the sync aggregate in the block.
func (b *BeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	if b.version == version.Phase0 {
		return nil, ErrNotSupported("SyncAggregate", b.version)
	}
	return b.syncAggregate, nil
}

// SetSyncAggregate sets the sync aggregate in the block.
func (b *BeaconBlockBody) SetSyncAggregate(s *eth.SyncAggregate) error {
	if b.version == version.Phase0 {
		return ErrNotSupported("SyncAggregate", b.version)
	}
	b.syncAggregate = eth.CopySyncAggregate(s)
	return nil
}

// Execution returns the execution payload of the block body.
func (b *BeaconBlockBody) Execution() (interfaces.ExecutionData, error) {
	switch b.version {
	case version.Phase0, version.Altair:
		return nil, ErrNotSupported("Execution", b.version)
	case version.Bellatrix:
		if b.isBlinded {
			var ph *enginev1.ExecutionPayloadHeader
			var ok bool
			if b.executionPayloadHeader != nil {
				ph, ok = b.executionPayloadHeader.Proto().(*enginev1.ExecutionPayloadHeader)
				if !ok {
					return nil, errPayloadHeaderWrongType
				}
			}
			return WrappedExecutionPayloadHeader(ph)
		}
		var p *enginev1.ExecutionPayload
		var ok bool
		if b.executionPayload != nil {
			p, ok = b.executionPayload.Proto().(*enginev1.ExecutionPayload)
			if !ok {
				return nil, errPayloadWrongType
			}
		}
		return WrappedExecutionPayload(p)
	case version.Capella:
		if b.isBlinded {
			var ph *enginev1.ExecutionPayloadHeaderCapella
			var ok bool
			if b.executionPayloadHeader != nil {
				ph, ok = b.executionPayloadHeader.Proto().(*enginev1.ExecutionPayloadHeaderCapella)
				if !ok {
					return nil, errPayloadHeaderWrongType
				}
				return WrappedExecutionPayloadHeaderCapella(ph)
			}
		}
		var p *enginev1.ExecutionPayloadCapella
		var ok bool
		if b.executionPayload != nil {
			p, ok = b.executionPayload.Proto().(*enginev1.ExecutionPayloadCapella)
			if !ok {
				return nil, errPayloadWrongType
			}
		}
		return WrappedExecutionPayloadCapella(p)
	case version.EIP4844:
		if b.isBlinded {
			var ph *enginev1.ExecutionPayloadHeader4844
			var ok bool
			if b.executionPayloadHeader != nil {
				ph, ok = b.executionPayloadHeader.Proto().(*enginev1.ExecutionPayloadHeader4844)
				if !ok {
					return nil, errPayloadHeaderWrongType
				}
				return WrappedExecutionPayloadHeaderEIP4844(ph)
			}
		}
		var p *enginev1.ExecutionPayload4844
		var ok bool
		if b.executionPayload != nil {
			p, ok = b.executionPayload.Proto().(*enginev1.ExecutionPayload4844)
			if !ok {
				return nil, errPayloadWrongType
			}
		}
		return WrappedExecutionPayloadEIP4844(p)
	default:
		return nil, errIncorrectBlockVersion
	}
}

// SetExecution sets the execution payload of the block body.
func (b *BeaconBlockBody) SetExecution(e interfaces.ExecutionData) error {
	if b.version == version.Phase0 || b.version == version.Altair {
		return ErrNotSupported("Execution", b.version)
	}
	if b.isBlinded {
		b.executionPayloadHeader = e // TODO: Copy?
		return nil
	}
	b.executionPayload = e
	return nil
}

// BLSToExecutionChanges returns the `BLSToExecutionChanges` objects in the block.
func (b *BeaconBlockBody) BLSToExecutionChanges() ([]*eth.SignedBLSToExecutionChange, error) {
	if b.version < version.Capella {
		return nil, ErrNotSupported("BLSToExecutionChanges", b.version)
	}
	return b.blsToExecutionChanges, nil
}

// BlobKzgCommitments returns the blob kzg commitments in the block.
func (b *BeaconBlockBody) BlobKzgCommitments() ([][]byte, error) {
	switch b.version {
	case version.Phase0, version.Altair, version.Bellatrix, version.Capella:
		return nil, ErrNotSupported("BlobKzgCommitments", b.version)
	case version.EIP4844:
		return b.blobKzgCommitments, nil
	default:
		return nil, errIncorrectBlockVersion
	}
}

// SetBlobKzgCommitments sets the blob kzg commitments in the block.
func (b *BeaconBlockBody) SetBlobKzgCommitments(c [][]byte) error {
	switch b.version {
	case version.Phase0, version.Altair, version.Bellatrix, version.Capella:
		return ErrNotSupported("BlobKzgCommitments", b.version)
	case version.EIP4844:
		b.blobKzgCommitments = c
		return nil
	default:
		return errIncorrectBlockVersion
	}
}

// SetBLSToExecutionChanges sets the BLS to execution changes in the block.
func (b *BeaconBlockBody) SetBLSToExecutionChanges(blsToExecutionChanges []*eth.SignedBLSToExecutionChange) error {
	if b.version < version.Capella {
		return ErrNotSupported("BLSToExecutionChanges", b.version)
	}
	b.blsToExecutionChanges = eth.CopyBLSToExecutionChanges(blsToExecutionChanges)
	return nil
}

// HashTreeRoot returns the ssz root of the block body.
func (b *BeaconBlockBody) HashTreeRoot() ([field_params.RootLength]byte, error) {
	pb, err := b.Proto()
	if err != nil {
		return [field_params.RootLength]byte{}, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.BeaconBlockBody).HashTreeRoot()
	case version.Altair:
		return pb.(*eth.BeaconBlockBodyAltair).HashTreeRoot()
	case version.Bellatrix:
		if b.isBlinded {
			return pb.(*eth.BlindedBeaconBlockBodyBellatrix).HashTreeRoot()
		}
		return pb.(*eth.BeaconBlockBodyBellatrix).HashTreeRoot()
	case version.Capella:
		if b.isBlinded {
			return pb.(*eth.BlindedBeaconBlockBodyCapella).HashTreeRoot()
		}
		return pb.(*eth.BeaconBlockBodyCapella).HashTreeRoot()
	case version.EIP4844:
		if b.isBlinded {
			return pb.(*eth.BlindedBeaconBlockBody4844).HashTreeRoot()
		}
		return pb.(*eth.BeaconBlockBody4844).HashTreeRoot()
	default:
		return [field_params.RootLength]byte{}, errIncorrectBodyVersion
	}
}
