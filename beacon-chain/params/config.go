// Package params defines important constants that are essential to the beacon chain.
package params

import (
	"math/big"
)

var (
	// CycleLength is the beacon chain cycle length in slots.
	CycleLength = uint64(64)
	// ShardCount is a fixed number.
	ShardCount = 1024
	// DefaultBalance of a validator in wei.
	DefaultBalance = new(big.Int).Div(big.NewInt(32), big.NewInt(int64(EtherDenomination)))
	// MaxValidators in the protocol.
	MaxValidators = 4194304
	// SlotDuration in seconds.
	SlotDuration = uint64(8)
	// Cofactor is used cutoff algorithm to select slot and shard cutoffs.
	Cofactor = 19
	// MinCommiteeSize is the minimal number of validator needs to be in a committee.
	MinCommiteeSize = uint64(128)
	// DefaultEndDynasty is the upper bound of dynasty. We use it to track queued and exited validators.
	DefaultEndDynasty = uint64(999999999999999999)
	// BootstrappedValidatorsCount is the number of validators we seed the first crystallized
	// state with. This number has yet to be decided by research and is arbitrary for now.
	BootstrappedValidatorsCount = 1000
	// MinDynastyLength is the slots needed before dynasty transition happens.
	MinDynastyLength = uint64(256)
	// EtherDenomination is the denomination of ether in wei.
	EtherDenomination = 1e18
	// BaseRewardQuotient is where 1/BaseRewardQuotient is the per-slot interest rate which will,
	// compound to an annual rate of 3.88% for 10 million eth staked.
	BaseRewardQuotient = 32768
	// SqrtDropTime is a constant set to reflect the amount of time it will take for the quadratic leak to
	// cut nonparticipating validators’ deposits by 39.4%.
	SqrtDropTime = uint64(1048576)
)
