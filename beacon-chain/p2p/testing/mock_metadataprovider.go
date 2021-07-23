package testing

import (
	"github.com/prysmaticlabs/prysm/proto/prysm"
)

// MockMetadataProvider is a fake implementation of the MetadataProvider interface.
type MockMetadataProvider struct {
	Data prysm.Metadata
}

// Metadata --
func (m *MockMetadataProvider) Metadata() prysm.Metadata {
	return m.Data
}

// MetadataSeq --
func (m *MockMetadataProvider) MetadataSeq() uint64 {
	return m.Data.SequenceNumber()
}
