package testing

import (
	metadata2 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/metadata"
)

// MockMetadataProvider is a fake implementation of the MetadataProvider interface.
type MockMetadataProvider struct {
	Data metadata2.Metadata
}

// Metadata --
func (m *MockMetadataProvider) Metadata() metadata2.Metadata {
	return m.Data
}

// MetadataSeq --
func (m *MockMetadataProvider) MetadataSeq() uint64 {
	return m.Data.SequenceNumber()
}
