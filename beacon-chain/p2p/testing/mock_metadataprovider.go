package testing

import (
	p2pInterfaces "github.com/prysmaticlabs/prysm/proto/beacon/p2p/interfaces"
)

// MockMetadataProvider is a fake implementation of the MetadataProvider interface.
type MockMetadataProvider struct {
	Data p2pInterfaces.Metadata
}

// Metadata --
func (m *MockMetadataProvider) Metadata() p2pInterfaces.Metadata {
	return m.Data
}

// MetadataSeq --
func (m *MockMetadataProvider) MetadataSeq() uint64 {
	return m.Data.SequenceNumber()
}
