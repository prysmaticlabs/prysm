package testing

import pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"

// MockMetadataProvider is a fake implementation of the MetadataProvider interface.
type MockMetadataProvider struct {
	Data *pb.MetaData
}

// Metadata --
func (m *MockMetadataProvider) Metadata() *pb.MetaData {
	return m.Data
}

// MetadataSeq --
func (m *MockMetadataProvider) MetadataSeq() uint64 {
	return m.Data.SeqNumber
}
