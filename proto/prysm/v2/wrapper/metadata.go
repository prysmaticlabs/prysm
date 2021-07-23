package wrapper

import (
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/proto/beacon/p2p"
	pb "github.com/prysmaticlabs/prysm/proto/proto/prysm/v2"
	"google.golang.org/protobuf/proto"
)

// MetadataV0 is a convenience wrapper around our metadata protobuf object.
type MetadataV0 struct {
	md *pb.MetaDataV0
}

// WrappedMetadataV0 wrappers around the provided protobuf object.
func WrappedMetadataV0(md *pb.MetaDataV0) MetadataV0 {
	return MetadataV0{md: md}
}

// SequenceNumber returns the sequence number from the metadata.
func (m MetadataV0) SequenceNumber() uint64 {
	return m.md.SeqNumber
}

// AttnetsBitfield retruns the bitfield stored in the metadata.
func (m MetadataV0) AttnetsBitfield() bitfield.Bitvector64 {
	return m.md.Attnets
}

// InnerObject returns the underlying metadata protobuf structure.
func (m MetadataV0) InnerObject() interface{} {
	return m.md
}

// IsNil checks for the nilness of the underlying object.
func (m MetadataV0) IsNil() bool {
	return m.md == nil
}

// Copy performs a full copy of the underlying metadata object.
func (m MetadataV0) Copy() p2p.Metadata {
	return WrappedMetadataV0(proto.Clone(m.md).(*pb.MetaDataV0))
}

// MetadataObjV0 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV0) MetadataObjV0() *pb.MetaDataV0 {
	return m.md
}

// MetadataObjV1 returns the inner metatdata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV0) MetadataObjV1() *pb.MetaDataV1 {
	return nil
}

// MetadataV1 is a convenience wrapper around our metadata v2 protobuf object.
type MetadataV1 struct {
	md *pb.MetaDataV1
}

// WrappedMetadataV1 wrappers around the provided protobuf object.
func WrappedMetadataV1(md *pb.MetaDataV1) MetadataV1 {
	return MetadataV1{md: md}
}

// SequenceNumber returns the sequence number from the metadata.
func (m MetadataV1) SequenceNumber() uint64 {
	return m.md.SeqNumber
}

// AttnetsBitfield retruns the bitfield stored in the metadata.
func (m MetadataV1) AttnetsBitfield() bitfield.Bitvector64 {
	return m.md.Attnets
}

// InnerObject returns the underlying metadata protobuf structure.
func (m MetadataV1) InnerObject() interface{} {
	return m.md
}

// IsNil checks for the nilness of the underlying object.
func (m MetadataV1) IsNil() bool {
	return m.md == nil
}

// Copy performs a full copy of the underlying metadata object.
func (m MetadataV1) Copy() p2p.Metadata {
	return WrappedMetadataV1(proto.Clone(m.md).(*pb.MetaDataV1))
}

// MetadataObjV0 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV1) MetadataObjV0() *pb.MetaDataV0 {
	return nil
}

// MetadataObjV1 returns the inner metatdata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV1) MetadataObjV1() *pb.MetaDataV1 {
	return m.md
}
