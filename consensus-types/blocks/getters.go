package blocks

import (
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

// BeaconBlockIsNil checks if any composite field of input signed beacon block is nil.
// Access to these nil fields will result in run time panic,
// it is recommended to run these checks as first line of defense.
func BeaconBlockIsNil(b interfaces.SignedBeaconBlock) error {
	if b == nil || b.IsNil() {
		return errNilSignedBeaconBlock
	}
	if b.Block().IsNil() {
		return errNilBeaconBlock
	}
	if b.Block().Body().IsNil() {
		return errNilBeaconBlockBody
	}
	return nil
}

// Signature returns the respective block signature.
func (b *SignedBeaconBlock) Signature() []byte {
	return b.signature
}

// Block returns the underlying beacon block object.
func (b *SignedBeaconBlock) Block() *BeaconBlock {
	return b.block
}

// IsNil checks if the underlying beacon block is nil.
func (b *SignedBeaconBlock) IsNil() bool {
	return b == nil || b.block.IsNil()
}

// Copy performs a deep copy of the signed beacon block object.
func (b *SignedBeaconBlock) Copy() (*SignedBeaconBlock, error) {
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
		cp := eth.CopySignedBeaconBlockBellatrix(pb.(*eth.SignedBeaconBlockBellatrix))
		return initSignedBlockFromProtoBellatrix(cp)
	case version.BellatrixBlind:
		cp := eth.CopySignedBlindedBeaconBlockBellatrix(pb.(*eth.SignedBlindedBeaconBlockBellatrix))
		return initBlindedSignedBlockFromProtoBellatrix(cp)
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
		return &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Bellatrix{Bellatrix: pb.(*eth.SignedBeaconBlockBellatrix)},
		}, nil
	case version.BellatrixBlind:
		return &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: pb.(*eth.SignedBlindedBeaconBlockBellatrix)},
		}, nil
	default:
		return nil, errIncorrectBlockVersion
	}

}

// PbPhase0Block returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbPhase0Block() (*eth.SignedBeaconBlock, error) {
	if b.version != version.Phase0 {
		return nil, errNotSupported("PbPhase0Block", b.version)
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
		return nil, errNotSupported("PbAltairBlock", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBeaconBlockAltair), nil
}

// PbBellatrixBlock returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbBellatrixBlock() (*eth.SignedBeaconBlockBellatrix, error) {
	if b.version != version.Bellatrix {
		return nil, errNotSupported("PbBellatrixBlock", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBeaconBlockBellatrix), nil
}

// PbBlindedBellatrixBlock returns the underlying protobuf object.
func (b *SignedBeaconBlock) PbBlindedBellatrixBlock() (*eth.SignedBlindedBeaconBlockBellatrix, error) {
	if b.version != version.BellatrixBlind {
		return nil, errNotSupported("PbBlindedBellatrixBlock", b.version)
	}
	pb, err := b.Proto()
	if err != nil {
		return nil, err
	}
	return pb.(*eth.SignedBlindedBeaconBlockBellatrix), nil
}

// Version of the underlying protobuf object.
func (b *SignedBeaconBlock) Version() int {
	return b.version
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
			ParentRoot:    b.block.parentRoot,
			StateRoot:     b.block.stateRoot,
			BodyRoot:      root[:],
		},
		Signature: b.signature,
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
		return pb.(*eth.SignedBeaconBlockBellatrix).MarshalSSZ()
	case version.BellatrixBlind:
		return pb.(*eth.SignedBlindedBeaconBlockBellatrix).MarshalSSZ()
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
		return pb.(*eth.SignedBeaconBlockBellatrix).MarshalSSZTo(dst)
	case version.BellatrixBlind:
		return pb.(*eth.SignedBlindedBeaconBlockBellatrix).MarshalSSZTo(dst)
	default:
		return []byte{}, errIncorrectBlockVersion
	}
}

// SizeSSZ returns the size of the serialized signed block
func (b *SignedBeaconBlock) SizeSSZ() (int, error) {
	pb, err := b.Proto()
	if err != nil {
		return 0, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.SignedBeaconBlock).SizeSSZ(), nil
	case version.Altair:
		return pb.(*eth.SignedBeaconBlockAltair).SizeSSZ(), nil
	case version.Bellatrix:
		return pb.(*eth.SignedBeaconBlockBellatrix).SizeSSZ(), nil
	case version.BellatrixBlind:
		return pb.(*eth.SignedBlindedBeaconBlockBellatrix).SizeSSZ(), nil
	default:
		return 0, errIncorrectBlockVersion
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
		pb := &eth.SignedBeaconBlockBellatrix{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newBlock, err = initSignedBlockFromProtoBellatrix(pb)
		if err != nil {
			return err
		}
	case version.BellatrixBlind:
		pb := &eth.SignedBlindedBeaconBlockBellatrix{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newBlock, err = initBlindedSignedBlockFromProtoBellatrix(pb)
		if err != nil {
			return err
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

// ProposerIndex returns the proposer index of the beacon block.
func (b *BeaconBlock) ProposerIndex() types.ValidatorIndex {
	return b.proposerIndex
}

// ParentRoot returns the parent root of beacon block.
func (b *BeaconBlock) ParentRoot() []byte {
	return b.parentRoot
}

// StateRoot returns the state root of the beacon block.
func (b *BeaconBlock) StateRoot() []byte {
	return b.stateRoot
}

// Body returns the underlying block body.
func (b *BeaconBlock) Body() *BeaconBlockBody {
	return b.body
}

// IsNil checks if the beacon block is nil.
func (b *BeaconBlock) IsNil() bool {
	return b == nil || b.Body().IsNil()
}

// IsBlinded checks if the beacon block is a blinded block.
func (b *BeaconBlock) IsBlinded() bool {
	switch b.version {
	case version.Phase0, version.Altair, version.Bellatrix:
		return false
	case version.BellatrixBlind:
		return true
	default:
		return false
	}
}

// Version of the underlying protobuf object.
func (b *BeaconBlock) Version() int {
	return b.version
}

// HashTreeRoot returns the ssz root of the block.
func (b *BeaconBlock) HashTreeRoot() ([32]byte, error) {
	pb, err := b.Proto()
	if err != nil {
		return [32]byte{}, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.BeaconBlock).HashTreeRoot()
	case version.Altair:
		return pb.(*eth.BeaconBlockAltair).HashTreeRoot()
	case version.Bellatrix:
		return pb.(*eth.BeaconBlockBellatrix).HashTreeRoot()
	case version.BellatrixBlind:
		return pb.(*eth.BlindedBeaconBlockBellatrix).HashTreeRoot()
	default:
		return [32]byte{}, errIncorrectBlockVersion
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
		return pb.(*eth.BeaconBlockBellatrix).HashTreeRootWith(h)
	case version.BellatrixBlind:
		return pb.(*eth.BlindedBeaconBlockBellatrix).HashTreeRootWith(h)
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
		return pb.(*eth.BeaconBlockBellatrix).MarshalSSZ()
	case version.BellatrixBlind:
		return pb.(*eth.BlindedBeaconBlockBellatrix).MarshalSSZ()
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
		return pb.(*eth.BeaconBlockBellatrix).MarshalSSZTo(dst)
	case version.BellatrixBlind:
		return pb.(*eth.BlindedBeaconBlockBellatrix).MarshalSSZTo(dst)
	default:
		return []byte{}, errIncorrectBlockVersion
	}
}

// SizeSSZ returns the size of the serialized block.
func (b *BeaconBlock) SizeSSZ() (int, error) {
	pb, err := b.Proto()
	if err != nil {
		return 0, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.BeaconBlock).SizeSSZ(), nil
	case version.Altair:
		return pb.(*eth.BeaconBlockAltair).SizeSSZ(), nil
	case version.Bellatrix:
		return pb.(*eth.BeaconBlockBellatrix).SizeSSZ(), nil
	case version.BellatrixBlind:
		return pb.(*eth.BlindedBeaconBlockBellatrix).SizeSSZ(), nil
	default:
		return 0, errIncorrectBlockVersion
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
		pb := &eth.BeaconBlockBellatrix{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newBlock, err = initBlockFromProtoBellatrix(pb)
		if err != nil {
			return err
		}
	case version.BellatrixBlind:
		pb := &eth.BlindedBeaconBlockBellatrix{}
		if err := pb.UnmarshalSSZ(buf); err != nil {
			return err
		}
		var err error
		newBlock, err = initBlindedBlockFromProtoBellatrix(pb)
		if err != nil {
			return err
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
		return &validatorpb.SignRequest_BlockV2{BlockV2: pb.(*eth.BeaconBlockAltair)}, nil
	case version.Bellatrix:
		return &validatorpb.SignRequest_BlockV3{BlockV3: pb.(*eth.BeaconBlockBellatrix)}, nil
	case version.BellatrixBlind:
		return &validatorpb.SignRequest_BlindedBlockV3{BlindedBlockV3: pb.(*eth.BlindedBeaconBlockBellatrix)}, nil
	default:
		return nil, errIncorrectBlockVersion
	}
}

// IsNil checks if the block body is nil.
func (b *BeaconBlockBody) IsNil() bool {
	return b == nil
}

// RandaoReveal returns the randao reveal from the block body.
func (b *BeaconBlockBody) RandaoReveal() []byte {
	return b.randaoReveal
}

// Eth1Data returns the eth1 data in the block.
func (b *BeaconBlockBody) Eth1Data() *eth.Eth1Data {
	return b.eth1Data
}

// Graffiti returns the graffiti in the block.
func (b *BeaconBlockBody) Graffiti() []byte {
	return b.graffiti
}

// ProposerSlashings returns the proposer slashings in the block.
func (b *BeaconBlockBody) ProposerSlashings() []*eth.ProposerSlashing {
	return b.proposerSlashings
}

// AttesterSlashings returns the attester slashings in the block.
func (b *BeaconBlockBody) AttesterSlashings() []*eth.AttesterSlashing {
	return b.attesterSlashings
}

// Attestations returns the stored attestations in the block.
func (b *BeaconBlockBody) Attestations() []*eth.Attestation {
	return b.attestations
}

// Deposits returns the stored deposits in the block.
func (b *BeaconBlockBody) Deposits() []*eth.Deposit {
	return b.deposits
}

// VoluntaryExits returns the voluntary exits in the block.
func (b *BeaconBlockBody) VoluntaryExits() []*eth.SignedVoluntaryExit {
	return b.voluntaryExits
}

// SyncAggregate returns the sync aggregate in the block.
func (b *BeaconBlockBody) SyncAggregate() (*eth.SyncAggregate, error) {
	if b.version == version.Phase0 {
		return nil, errNotSupported("SyncAggregate", b.version)
	}
	return b.syncAggregate, nil
}

// ExecutionPayload returns the execution payload of the block body.
func (b *BeaconBlockBody) ExecutionPayload() (*enginev1.ExecutionPayload, error) {
	if b.version != version.Bellatrix {
		return nil, errNotSupported("ExecutionPayload", b.version)
	}
	return b.executionPayload, nil
}

// ExecutionPayloadHeader returns the execution payload header of the block body.
func (b *BeaconBlockBody) ExecutionPayloadHeader() (*enginev1.ExecutionPayloadHeader, error) {
	if b.version != version.BellatrixBlind {
		return nil, errNotSupported("ExecutionPayloadHeader", b.version)
	}
	return b.executionPayloadHeader, nil
}

// HashTreeRoot returns the ssz root of the block body.
func (b *BeaconBlockBody) HashTreeRoot() ([32]byte, error) {
	pb, err := b.Proto()
	if err != nil {
		return [32]byte{}, err
	}
	switch b.version {
	case version.Phase0:
		return pb.(*eth.BeaconBlockBody).HashTreeRoot()
	case version.Altair:
		return pb.(*eth.BeaconBlockBodyAltair).HashTreeRoot()
	case version.Bellatrix:
		return pb.(*eth.BeaconBlockBodyBellatrix).HashTreeRoot()
	case version.BellatrixBlind:
		return pb.(*eth.BlindedBeaconBlockBodyBellatrix).HashTreeRoot()
	default:
		return [32]byte{}, errIncorrectBodyVersion
	}
}
