package testing

import (
	"github.com/prysmaticlabs/prysm/shared/interfaces"
)

// MockMetadataProvider is a fake implementation of the MetadataProvider interface.
type MockMetadataProvider struct {
	Data interfaces.Metadata
}

// Metadata --
func (m *MockMetadataProvider) Metadata() interfaces.Metadata {
	return m.Data
}

// MetadataSeq --
func (m *MockMetadataProvider) MetadataSeq() uint64 {
	return m.Data.SequenceNumber()
}
