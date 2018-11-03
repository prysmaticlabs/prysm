// Package params defines important constants that are essential to the beacon chain.
package params

import (
	"time"
)

var env = "default"

// ValidatorStatusCode defines which stage a validator is in.
type ValidatorStatusCode int

// SpecialRecordType defines type of special record this message represents.
type SpecialRecordType int

// ValidatorSetDeltaFlags is used for light client to track validator entries.
type ValidatorSetDeltaFlags int

// Config contains configs for node to participate in beacon chain.
type Config struct {
	ShardCount                    uint64    // ShardCount is the fixed number of shards in Ethereum 2.0.
	DepositSize                   uint64    // DepositSize is how much a validator has deposited in Gwei.
	BootstrappedValidatorsCount   uint64    // BootstrappedValidatorsCount is the number of validators we seed the first crystallized state.
	ModuloBias                    uint64    // ModuloBias is the upper bound of validator shuffle function. Can shuffle validator lists up to that size.
	Gwei                          uint64    // Gwei is the denomination of Gwei in Ether.
	CycleLength                   uint64    // CycleLength is one beacon chain cycle length in slots.
	SlotDuration                  uint64    // SlotDuration is how many seconds are in a single slot.
	MinValidatorSetChangeInterval uint64    // MinValidatorSetChangeInterval is the slots needed before validator set changes.
	MinCommiteeSize               uint64    // MinCommiteeSize is the minimal number of validator needs to be in a committee.
	BaseRewardQuotient            uint64    // BaseRewardQuotient is used to calculate validator per-slot interest rate.
	SqrtExpDropTime               uint64    // SqrtEDropTime is a constant to reflect time it takes to cut offline validators’ deposits by 39.4%.
	GenesisTime                   time.Time // GenesisTime used by the protocol.
	LogOutMessage                 string    // This is the message a validator will send in order to log out.
	WithdrawalPeriod              uint64    // WithdrawalPeriod is the number of slots between a validator exit and validator balance being withdrawable.
	MaxValidatorChurnQuotient     uint64    // MaxValidatorChurnQuotient defines the quotient how many validators can change each time.
	MinDeposit                    uint64    // MinDeposit is the minimal amount of Ether a validator needs to participate.
	SimulatedBlockRandao          [32]byte  // SimulatedBlockRandao is a RANDAO seed stubbed for simulated block to advance chain.
	InitialForkVersion 	          uint64    // InitialForkVersion is used to track fork version between station transitions.
}

var defaultConfig = &Config{
	GenesisTime:                   time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC),
	ModuloBias:                    16777216 - 1,
	CycleLength:                   uint64(64),
	ShardCount:                    1024,
	Gwei:                          1e9,
	DepositSize:                   32,
	MinDeposit:                    16,
	SlotDuration:                  uint64(16),
	MinCommiteeSize:               uint64(128),
	BootstrappedValidatorsCount:   16384,
	MinValidatorSetChangeInterval: uint64(256),
	BaseRewardQuotient:            uint64(32768),
	SqrtExpDropTime:               uint64(65536),
	WithdrawalPeriod:              uint64(524288),
	MaxValidatorChurnQuotient:     uint64(32),
	InitialForkVersion: 0,
}

var demoConfig = &Config{
	GenesisTime:                   time.Now(),
	ModuloBias:                    16777216 - 1,
	CycleLength:                   uint64(5),
	ShardCount:                    5,
	Gwei:                          1e9,
	DepositSize:                   32,
	MinDeposit:                    16,
	SlotDuration:                  uint64(2),
	MinCommiteeSize:               uint64(3),
	MinValidatorSetChangeInterval: uint64(256),
	BaseRewardQuotient:            uint64(32768),
	SqrtExpDropTime:               uint64(65536),
	WithdrawalPeriod:              uint64(128),
	MaxValidatorChurnQuotient:     uint64(32),
	InitialForkVersion: 0,
	SimulatedBlockRandao:          [32]byte{'S', 'I', 'M', 'U', 'L', 'A', 'T', 'E', 'R'},
}

const (
	// PendingActivation means a validator is queued and waiting to be active.
	PendingActivation ValidatorStatusCode = iota
	// Active means a validator is participating validator duties.
	Active
	// PendingExit means a validator is waiting to exit.
	PendingExit
	// PendingWithdraw means a validator is waiting to get balance back.
	PendingWithdraw
	// Withdrawn means a validator has successfully withdrawn balance.
	Withdrawn
	// Penalized means a validator did something bad and got slashed.
	Penalized = 128
)

const (
	// Logout means a validator is requesting to exit the validator pool.
	Logout SpecialRecordType = iota
	// CasperSlashing means a validator has committed slashing penalty, other nodes can submit a message.
	// to report and earn rewards.
	CasperSlashing
	// RandaoChange means a validator revealed the pre-image of its randao commitment.
	RandaoChange
)

const (
	// Entry means this is an entry message for light client to track overall validator status.
	Entry ValidatorSetDeltaFlags = iota
	// Exit means this is an exit message for light client to track overall validator status.
	Exit
)

// GetConfig retrieves beacon node config.
func GetConfig() *Config {
	switch env {
	case "default":
		return defaultConfig
	case "demo":
		return demoConfig
	default:
		return defaultConfig
	}
}

// SetEnv sets which config to use.
func SetEnv(e string) {
	env = e
}
