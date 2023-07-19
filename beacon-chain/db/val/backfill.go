package val

// BackfillStatus is used to persist data needed to manage the backfill process.
type BackfillStatus struct {
	OriginSlot uint64
	LowSlot    uint64
	HighSlot   uint64
	OriginRoot [32]byte
	LowRoot    [32]byte
	HighRoot   [32]byte
}
