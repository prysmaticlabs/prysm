package interfaces

import (
	"github.com/prysmaticlabs/go-bitfield"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"google.golang.org/protobuf/proto"
)

// MetadataV1 is a convenience wrapper around our metadata protobuf object.
type MetadataV1 struct {
	md *pb.MetaDataV0
}

// WrappedMetadataV1 wrappers around the provided protobuf object.
func WrappedMetadataV1(md *pb.MetaDataV0) MetadataV1 {
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
func (m MetadataV1) Copy() Metadata {
	return WrappedMetadataV1(proto.Clone(m.md).(*pb.MetaDataV0))
}

// MetadataObj returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV1) MetadataObj() *pb.MetaDataV0 {
	return m.md
}

// MetadataObjV2 returns the inner metatdata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV1) MetadataObjV2() *pb.MetaDataV1 {
	return nil
}

// MetadataV2 is a convenience wrapper around our metadata v2 protobuf object.
type MetadataV2 struct {
	md *pb.MetaDataV1
}

// WrappedMetadataV2 wrappers around the provided protobuf object.
func WrappedMetadataV2(md *pb.MetaDataV1) MetadataV2 {
	return MetadataV2{md: md}
}

// SequenceNumber returns the sequence number from the metadata.
func (m MetadataV2) SequenceNumber() uint64 {
	return m.md.SeqNumber
}

// AttnetsBitfield retruns the bitfield stored in the metadata.
func (m MetadataV2) AttnetsBitfield() bitfield.Bitvector64 {
	return m.md.Attnets
}

// InnerObject returns the underlying metadata protobuf structure.
func (m MetadataV2) InnerObject() interface{} {
	return m.md
}

// IsNil checks for the nilness of the underlying object.
func (m MetadataV2) IsNil() bool {
	return m.md == nil
}

// Copy performs a full copy of the underlying metadata object.
func (m MetadataV2) Copy() Metadata {
	return WrappedMetadataV2(proto.Clone(m.md).(*pb.MetaDataV1))
}

// MetadataObj returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV2) MetadataObj() *pb.MetaDataV0 {
	return nil
}

// MetadataObjV2 returns the inner metatdata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV2) MetadataObjV2() *pb.MetaDataV1 {
	return m.md
}
