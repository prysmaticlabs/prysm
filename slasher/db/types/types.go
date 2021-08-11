// Package types includes important database-related types for
// slasher-specific logic.
package types

// SlashingStatus enum like structure.
type SlashingStatus uint8

//noinspection GoUnusedConst
const (
	// Unknown default status in case it is not set
	Unknown = iota
	// Active slashing proof hasn't been included yet.
	Active
	// Included slashing proof that has been included in a block.
	Included
	// Reverted slashing proof that has been reverted and therefore is relevant again.
	Reverted //relevant again
)

const (
	// UseCache is used to mark when calling a DB function, to save it to the cache.
	UseCache = true
	// UseDB is used to mark when calling a DB function, to save it to the DB.
	UseDB         = false
	UnknownStatus = "Unknown"
)

func (s SlashingStatus) String() string {
	names := [...]string{
		"Unknown",
		"Active",
		"Included",
		"Reverted"}

	if s < Active || s > Reverted {
		return UnknownStatus
	}
	// return the name of a SlashingStatus
	// constant from the names array
	// above.
	return names[s]
}

// SlashingType enum like type of slashing proof.
type SlashingType uint8

const (
	// Proposal enum value.
	Proposal = iota
	// Attestation enum value.
	Attestation
)

// String returns the string representation of the status SlashingType.
func (t SlashingType) String() string {
	names := [...]string{
		"Proposal",
		"Attestation",
	}

	if t < Active || t > Reverted {
		return UnknownStatus
	}
	return names[t]
}
