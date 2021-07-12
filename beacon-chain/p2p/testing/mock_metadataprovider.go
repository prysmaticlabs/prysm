package testing

import (
	"github.com/prysmaticlabs/prysm/proto/beacon/p2p"
)

// MockMetadataProvider is a fake implementation of the MetadataProvider interface.
type MockMetadataProvider struct {
	Data p2p.Metadata
}

// Metadata --
func (m *MockMetadataProvider) Metadata() p2p.Metadata {
	return m.Data
}

// MetadataSeq --
func (m *MockMetadataProvider) MetadataSeq() uint64 {
	return m.Data.SequenceNumber()
}
