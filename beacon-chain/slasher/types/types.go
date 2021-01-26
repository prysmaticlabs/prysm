package types

// ChunkKind to differentiate what kind of span we are working
// with for slashing detection, either min or max span.
type ChunkKind uint

const (
	MinSpan ChunkKind = iota
	MaxSpan
)

// AttestationRecord encapsulating all the necessary
// information we need to store in the database for slasher
// to properly perform slashing detection.
type AttestationRecord struct {
	Source      uint64
	Target      uint64
	SigningRoot [32]byte
}

// Epoch in eth2 is a fixed period of slots.
type Epoch uint64

// ValidatorIdx in eth2.
type ValidatorIdx uint64

// Span is a difference between two epochs, with the max epoch
// being WEAK_SUBJECTIVITY_PERIOD. This difference cannot grow beyond
// uint16.
type Span uint16
