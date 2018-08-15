// Package params defines important constants that are essential to the beacon chain.
package params

const (
	// AttesterReward determines how much ETH attesters get for performing their duty.
	AttesterReward = 1
	// CycleLength is the beacon chain epoch length in slots.
	CycleLength = 10
	// ShardCount is a fixed number.
	ShardCount = 1024
	// DefaultBalance of a validator in ETH.
	DefaultBalance = 32
	// MaxValidators in the protocol.
	MaxValidators = 4194304
	// SlotDuration in seconds.
	SlotDuration = 8
	// Cofactor is used cutoff algorithm to select height and shard cutoffs.
	Cofactor = 19
	// MinCommiteeSize is the minimal number of validator needs to be in a committee.
	MinCommiteeSize = 128
	// DefaultEndDynasty is the upper bound of dynasty. We use it to track queued and exited validators.
	DefaultEndDynasty = 9999999999999999999
)
