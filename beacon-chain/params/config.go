package params

const (
	// AttesterCount is the number of attesters per committee/
	AttesterCount = 32
	// EpochLength is the beacon chain epoch length in slots.
	EpochLength = 64
	// ShardCount is a fixed number.
	ShardCount = 1024
	// DefaultBalance of a validator.
	DefaultBalance = 32000
	// MaxValidators in the protocol.
	MaxValidators = 4194304
	// SlotDuration in seconds.
	SlotDuration = 8
	// Cofactor is used cutoff algorithm to select height and shard cutoffs.
	Cofactor = 19
	// MinCommiteeSize is the minimal number of validator needs to be in a committee.
	MinCommiteeSize = 128
)
