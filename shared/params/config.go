// Package params defines important constants that are essential to the
// Ethereum 2.0 services.
package params

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ValidatorStatusCode defines which stage a validator is in.
type ValidatorStatusCode int

// SpecialRecordType defines type of special record this message represents.
type SpecialRecordType int

// ValidatorSetDeltaFlags is used for light client to track validator entries.
type ValidatorSetDeltaFlags int

// BeaconChainConfig contains configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	ShardCount                           uint64         // ShardCount is the fixed number of shards in Ethereum 2.0.
	DepositSize                          uint64         // DepositSize is how much a validator has deposited in Eth.
	MinDeposit                           uint64         // MinDeposit is the minimal amount of Ether a validator needs to participate.
	Gwei                                 uint64         // Gwei is the denomination of Gwei in Ether.
	DepositContractAddress               common.Address // DepositContractAddress is the address of validator registration contract in PoW chain.
	TargetCommitteeSize                  uint64         // TargetCommitteeSize is the minimal number of validator needs to be in a committee.
	GenesisTime                          time.Time      // GenesisTime used by the protocol.
	SlotDuration                         uint64         // SlotDuration is how many seconds are in a single slot.
	CycleLength                          uint64         // CycleLength is one beacon chain cycle length in slots.
	MinValidatorSetChangeInterval        uint64         // MinValidatorSetChangeInterval is the slots needed before validator set changes.
	RandaoSlotsPerLayer                  uint64         // RandaoSlotsPerLayer defines how many randao slot a proposer can peel off once.
	SqrtExpDropTime                      uint64         // SqrtEDropTime is a constant to reflect time it takes to cut offline validators’ deposits by 39.4%.
	MinWithdrawalPeriod                  uint64         // MinWithdrawalPeriod defines the slots between a validator exit and validator balance being withdrawable.
	WithdrawalsPerCycle                  uint64         // WithdrawalsPerCycle defines how many withdrawals can go through per cycle.
	CollectivePenaltyCalculationPeriod   uint64         // CollectivePenaltyCalculationPeriod defines the period length for an aggregated penalty amount.
	DeletionPeriod                       uint64         // DeletionPeriod define the period length of when validator is deleted from the pool.
	ShardPersistentCommitteeChangePeriod uint64         // ShardPersistentCommitteeChangePeriod defines how often shard committee gets shuffled.
	BaseRewardQuotient                   uint64         // BaseRewardQuotient is used to calculate validator per-slot interest rate.
	MaxValidatorChurnQuotient            uint64         // MaxValidatorChurnQuotient defines the quotient how many validators can change each time.
	POWHashVotingPeriod                  uint64         // POWHashVotingPeriod defines how often PoW hash gets updated in beacon node.
	POWContractMerkleTreeDepth           uint64         // POWContractMerkleTreeDepth defines the depth of PoW contract merkle tree.
	MaxSpecialsPerBlock                  uint64         // MaxSpecialsPerBlock defines the max number special records permitted per beacon block.
	LogOutMessage                        string         // LogOutMessage is the message a validator submits to log out.
	InitialForkVersion                   uint32         // InitialForkVersion is used to track fork version between state transitions.
	SimulatedBlockRandao                 [32]byte       // SimulatedBlockRandao is a RANDAO seed stubbed in side simulated block to advance local beacon chain.
	ModuloBias                           uint64         // ModuloBias is the upper bound of validator shuffle function. Can shuffle validator lists up to that size.
	BootstrappedValidatorsCount          uint64         // BootstrappedValidatorsCount is the number of validators we seed to start beacon chain.
}

// ShardChainConfig contains configs for node to participate in shard chains.
type ShardChainConfig struct {
	ChunkSize         uint64 // ChunkSize defines the size of each chunk in bytes.
	MaxShardBlockSize uint64 // MaxShardBlockSize defines the max size of each shard block in bytes.
}

var defaultBeaconConfig = &BeaconChainConfig{
	ShardCount:                    1024,
	DepositSize:                   32,
	MinDeposit:                    16,
	Gwei:                          1e9,
	TargetCommitteeSize:           uint64(256),
	GenesisTime:                   time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC),
	SlotDuration:                  uint64(16),
	CycleLength:                   uint64(64),
	MinValidatorSetChangeInterval: uint64(256),
	RandaoSlotsPerLayer:           uint64(4096),
	SqrtExpDropTime:               uint64(65536),
	MinWithdrawalPeriod:           uint64(4096),
	WithdrawalsPerCycle:           uint64(8),
	BaseRewardQuotient:            uint64(32768),
	MaxValidatorChurnQuotient:     uint64(32),
	InitialForkVersion:            0,
	ModuloBias:                    16777216 - 1,
	BootstrappedValidatorsCount:   16384,
}

var demoBeaconConfig = &BeaconChainConfig{
	ShardCount:                    5,
	DepositSize:                   32,
	MinDeposit:                    16,
	Gwei:                          1e9,
	TargetCommitteeSize:           uint64(3),
	GenesisTime:                   time.Now(),
	SlotDuration:                  uint64(2),
	CycleLength:                   uint64(5),
	MinValidatorSetChangeInterval: uint64(15),
	RandaoSlotsPerLayer:           uint64(5),
	SqrtExpDropTime:               uint64(65536),
	MinWithdrawalPeriod:           uint64(20),
	WithdrawalsPerCycle:           uint64(2),
	BaseRewardQuotient:            uint64(32768),
	MaxValidatorChurnQuotient:     uint64(32),
	InitialForkVersion:            0,
	ModuloBias:                    16777216 - 1,
	SimulatedBlockRandao:          [32]byte{'S', 'I', 'M', 'U', 'L', 'A', 'T', 'E', 'R'},
}

var defaultShardConfig = &ShardChainConfig{
	ChunkSize:         uint64(256),
	MaxShardBlockSize: uint64(32768),
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
	Penalized = 127
)

const (
	// Logout means a validator is requesting to exit the validator pool.
	Logout SpecialRecordType = iota
	// CasperSlashing means a attester has committed slashing penalty which a surround vote was made.
	CasperSlashing
	// ProposerSlashing means a proposer has violated a slashing condition which 2 identical blocks were proposed
	// same height.
	ProposerSlashing
	// DepositProof means a validator has submitted a deposit and the data is the proof.
	DepositProof
)

const (
	// Entry means this is an entry message for light client to track overall validator status.
	Entry ValidatorSetDeltaFlags = iota
	// Exit means this is an exit message for light client to track overall validator status.
	Exit
)

var beaconConfig = defaultBeaconConfig
var shardConfig = defaultShardConfig

// BeaconConfig retrieves beacon chain config.
func BeaconConfig() *BeaconChainConfig {
	return beaconConfig
}

// ShardConfig retrieves shard chain config.
func ShardConfig() *ShardChainConfig {
	return shardConfig
}

// UseDemoBeaconConfig for beacon chain services.
func UseDemoBeaconConfig() {
	beaconConfig = demoBeaconConfig
}

// OverrideBeaconConfig by replacing the config. The preferred pattern is to
// call BeaconConfig(), change the specific parameters, and then call
// OverrideBeaconConfig(c). Any subsequent calls to params.BeaconConfig() will
// return this new configuration.
func OverrideBeaconConfig(c *BeaconChainConfig) {
	beaconConfig = c
}
