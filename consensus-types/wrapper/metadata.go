package wrapper

import (
	"github.com/prysmaticlabs/go-bitfield"
	pb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/metadata"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"google.golang.org/protobuf/proto"
)

// MetadataV0
// ----------

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

// AttnetsBitfield returns the bitfield stored in the metadata.
func (m MetadataV0) AttnetsBitfield() bitfield.Bitvector64 {
	return m.md.Attnets
}

// SyncnetsBitfield returns the bitfield stored in the metadata.
func (m MetadataV0) SyncnetsBitfield() bitfield.Bitvector4 {
	return bitfield.Bitvector4{0}
}

// CustodySubnetCount returns custody subnet count from the metadata.
func (m MetadataV0) CustodySubnetCount() uint64 {
	return 0
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
func (m MetadataV0) Copy() metadata.Metadata {
	return WrappedMetadataV0(proto.Clone(m.md).(*pb.MetaDataV0))
}

// MarshalSSZ marshals the underlying metadata object
// into its serialized form.
func (m MetadataV0) MarshalSSZ() ([]byte, error) {
	return m.md.MarshalSSZ()
}

// MarshalSSZTo marshals the underlying metadata object
// into its serialized form into the provided byte buffer.
func (m MetadataV0) MarshalSSZTo(dst []byte) ([]byte, error) {
	return m.md.MarshalSSZTo(dst)
}

// SizeSSZ returns the serialized size of the metadata object.
func (m MetadataV0) SizeSSZ() int {
	return m.md.SizeSSZ()
}

// UnmarshalSSZ unmarshals the provided byte buffer into
// the underlying metadata object.
func (m MetadataV0) UnmarshalSSZ(buf []byte) error {
	return m.md.UnmarshalSSZ(buf)
}

// MetadataObjV0 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV0) MetadataObjV0() *pb.MetaDataV0 {
	return m.md
}

// MetadataObjV1 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (MetadataV0) MetadataObjV1() *pb.MetaDataV1 {
	return nil
}

// MetadataObjV2 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (MetadataV0) MetadataObjV2() *pb.MetaDataV2 {
	return nil
}

// Version returns the fork version of the underlying object.
func (MetadataV0) Version() int {
	return version.Phase0
}

// MetadataV1
// ----------

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

// AttnetsBitfield returns the bitfield stored in the metadata.
func (m MetadataV1) AttnetsBitfield() bitfield.Bitvector64 {
	return m.md.Attnets
}

// SyncnetsBitfield returns the bitfield stored in the metadata.
func (m MetadataV1) SyncnetsBitfield() bitfield.Bitvector4 {
	return m.md.Syncnets
}

// CustodySubnetCount returns custody subnet count from the metadata.
func (m MetadataV1) CustodySubnetCount() uint64 {
	return 0
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
func (m MetadataV1) Copy() metadata.Metadata {
	return WrappedMetadataV1(proto.Clone(m.md).(*pb.MetaDataV1))
}

// MarshalSSZ marshals the underlying metadata object
// into its serialized form.
func (m MetadataV1) MarshalSSZ() ([]byte, error) {
	return m.md.MarshalSSZ()
}

// MarshalSSZTo marshals the underlying metadata object
// into its serialized form into the provided byte buffer.
func (m MetadataV1) MarshalSSZTo(dst []byte) ([]byte, error) {
	return m.md.MarshalSSZTo(dst)
}

// SizeSSZ returns the serialized size of the metadata object.
func (m MetadataV1) SizeSSZ() int {
	return m.md.SizeSSZ()
}

// UnmarshalSSZ unmarshals the provided byte buffer into
// the underlying metadata object.
func (m MetadataV1) UnmarshalSSZ(buf []byte) error {
	return m.md.UnmarshalSSZ(buf)
}

// MetadataObjV0 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (MetadataV1) MetadataObjV0() *pb.MetaDataV0 {
	return nil
}

// MetadataObjV1 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV1) MetadataObjV1() *pb.MetaDataV1 {
	return m.md
}

// MetadataObjV2 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV1) MetadataObjV2() *pb.MetaDataV2 {
	return nil
}

// Version returns the fork version of the underlying object.
func (MetadataV1) Version() int {
	return version.Altair
}

// MetadataV2
// ----------

// MetadataV2 is a convenience wrapper around our metadata v3 protobuf object.
type MetadataV2 struct {
	md *pb.MetaDataV2
}

// WrappedMetadataV2 wrappers around the provided protobuf object.
func WrappedMetadataV2(md *pb.MetaDataV2) MetadataV2 {
	return MetadataV2{md: md}
}

// SequenceNumber returns the sequence number from the metadata.
func (m MetadataV2) SequenceNumber() uint64 {
	return m.md.SeqNumber
}

// AttnetsBitfield returns the bitfield stored in the metadata.
func (m MetadataV2) AttnetsBitfield() bitfield.Bitvector64 {
	return m.md.Attnets
}

// SyncnetsBitfield returns the bitfield stored in the metadata.
func (m MetadataV2) SyncnetsBitfield() bitfield.Bitvector4 {
	return m.md.Syncnets
}

// CustodySubnetCount returns custody subnet count from the metadata.
func (m MetadataV2) CustodySubnetCount() uint64 {
	return m.md.CustodySubnetCount
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
func (m MetadataV2) Copy() metadata.Metadata {
	return WrappedMetadataV2(proto.Clone(m.md).(*pb.MetaDataV2))
}

// MarshalSSZ marshals the underlying metadata object
// into its serialized form.
func (m MetadataV2) MarshalSSZ() ([]byte, error) {
	return m.md.MarshalSSZ()
}

// MarshalSSZTo marshals the underlying metadata object
// into its serialized form into the provided byte buffer.
func (m MetadataV2) MarshalSSZTo(dst []byte) ([]byte, error) {
	return m.md.MarshalSSZTo(dst)
}

// SizeSSZ returns the serialized size of the metadata object.
func (m MetadataV2) SizeSSZ() int {
	return m.md.SizeSSZ()
}

// UnmarshalSSZ unmarshals the provided byte buffer into
// the underlying metadata object.
func (m MetadataV2) UnmarshalSSZ(buf []byte) error {
	return m.md.UnmarshalSSZ(buf)
}

// MetadataObjV0 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (MetadataV2) MetadataObjV0() *pb.MetaDataV0 {
	return nil
}

// MetadataObjV1 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV2) MetadataObjV1() *pb.MetaDataV1 {
	return nil
}

// MetadataObjV2 returns the inner metadata object in its type
// specified form. If it doesn't exist then we return nothing.
func (m MetadataV2) MetadataObjV2() *pb.MetaDataV2 {
	return m.md
}

// Version returns the fork version of the underlying object.
func (MetadataV2) Version() int {
	return version.Deneb
}
