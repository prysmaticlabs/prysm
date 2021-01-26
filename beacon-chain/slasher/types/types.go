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
