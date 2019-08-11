package filters

// QueryFilter defines a generic interface for type-asserting
// specific filters to use in querying DB objects.
type QueryFilter struct {
	// Root filter criteria.
	Root       []byte
	ParentRoot []byte
	// Slot filter criteria.
	StartSlot uint64
	EndSlot   uint64
	// Epoch filter criteria.
	StartEpoch uint64
	EndEpoch   uint64
	// Optional criteria to retrieve a genesis value.
	IsGenesis bool
}
