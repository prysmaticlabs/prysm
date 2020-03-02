package types

// DetectionKind defines an enum type that
// gives us information on the type of slashable offense
// found when analyzing validator min-max spans.
type DetectionKind uint8

const (
	// DoubleVote denotes a slashable offense in which
	// a validator cast two conflicting attestations within
	// the same target epoch.
	DoubleVote DetectionKind = iota
	// SurroundVote denotes a slashable offense in which
	// a validator surrounded or was surrounded by a previous
	// attestation created by the same validator.
	SurroundVote
)

// DetectionResult tells us the kind of slashable
// offense found from detecting on min-max spans +
// the slashable epoch for the offense.
// Also includes the signature bytes for assistance in
// finding the attestation for the slashing proof.
type DetectionResult struct {
	SlashableEpoch uint64
	Kind           DetectionKind
	SigBytes       [2]byte
}

// Span defines the structure used for detecting surround and double votes.
type Span struct {
	MinSpan     uint16
	MaxSpan     uint16
	SigBytes    [2]byte
	HasAttested bool
}
