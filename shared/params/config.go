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
	MaxProposerSlashings                    uint64         // MaxProposerSlashing defines the maximum number of slashings of proposers possible in a block.
	ShardCount                              uint64         // ShardCount is the fixed number of shards in Ethereum 2.0.
	DepositSize                             uint64         // DepositSize is how much a validator has deposited in Eth.
	MinTopUpSize                            uint64         // MinTopUpSize is the minimal amount of Ether a validator can top up.
	MinOnlineDepositSize                    uint64         // MinOnlineDepositSize is the minimal amount of Ether a validator needs to participate.
	Gwei                                    uint64         // Gwei is the denomination of Gwei in Ether.
	DepositContractAddress                  common.Address // DepositContractAddress is the address of validator registration contract in PoW chain.
	DepositsForChainStart                   uint64         // DepositsForChainStart defines how many deposits needed to start off beacon chain.
	TargetCommitteeSize                     uint64         // TargetCommitteeSize is the minimal number of validator needs to be in a committee.
	SlotDuration                            uint64         // SlotDuration is how many seconds are in a single slot.
	CycleLength                             uint64         // CycleLength is one beacon chain cycle length in slots.
	MinValidatorSetChangeInterval           uint64         // MinValidatorSetChangeInterval is the slots needed before validator set changes.
	ShardPersistentCommitteeChangePeriod    uint64         // ShardPersistentCommitteeChangePeriod defines how often shard committee gets shuffled.
	MinAttestationInclusionDelay            uint64         // MinAttestationInclusionDelay defines how long validator has to wait to include attestation for beacon block.
	SqrtExpDropTime                         uint64         // SqrtEDropTime is a constant to reflect time it takes to cut offline validators’ deposits by 39.4%.
	WithdrawalsPerCycle                     uint64         // WithdrawalsPerCycle defines how many withdrawals can go through per cycle.
	MinWithdrawalPeriod                     uint64         // MinWithdrawalPeriod defines the slots between a validator exit and validator balance being withdrawable.
	DeletionPeriod                          uint64         // DeletionPeriod define the period length of when validator is deleted from the pool.
	CollectivePenaltyCalculationPeriod      uint64         // CollectivePenaltyCalculationPeriod defines the period length for an aggregated penalty amount.
	PowReceiptRootVotingPeriod              uint64         // PowReceiptRootVotingPeriod defines how often PoW hash gets updated in beacon node.
	SlashingWhistlerBlowerRewardDenominator uint64         // SlashingWhistlerBlowerRewardDenominator defines how the reward denominator of whistler blower.
	BaseRewardQuotient                      uint64         // BaseRewardQuotient is used to calculate validator per-slot interest rate.
	IncluderRewardShareQuotient             uint64         // IncluderRewardShareQuotient defines the reward quotient for proposer.
	MaxValidatorChurnQuotient               uint64         // MaxValidatorChurnQuotient defines the quotient how many validators can change each time.
	POWContractMerkleTreeDepth              uint64         // POWContractMerkleTreeDepth defines the depth of PoW contract merkle tree.
	InitialForkVersion                      uint64         // InitialForkVersion is used to track fork version between state transitions.
	InitialForkSlot                         uint64         // InitialForkSlot is used to initialize the fork slot in the initial Beacon state.
	SimulatedBlockRandao                    [32]byte       // SimulatedBlockRandao is a RANDAO seed stubbed in side simulated block to advance local beacon chain.
	RandBytes                               uint64         // RandBytes is the number of bytes used as entropy to shuffle validators.
	BootstrappedValidatorsCount             uint64         // BootstrappedValidatorsCount is the number of validators we seed to start beacon chain.
	SyncPollingInterval                     int64          // SyncPollingInterval queries network nodes for sync status.
	GenesisTime                             time.Time      // GenesisTime used by the protocol.
	MaxNumLog2Validators                    uint64         // Max number of validators in Log2 can exist given total ETH supply.
	EpochLength                             uint64         // Number of slots that define an Epoch.
}

// ShardChainConfig contains configs for node to participate in shard chains.
type ShardChainConfig struct {
	ChunkSize         uint64 // ChunkSize defines the size of each chunk in bytes.
	MaxShardBlockSize uint64 // MaxShardBlockSize defines the max size of each shard block in bytes.
}

var defaultBeaconConfig = &BeaconChainConfig{
	MaxProposerSlashings:          16,
	ShardCount:                    1024,
	DepositSize:                   32,
	MinTopUpSize:                  1,
	MinOnlineDepositSize:          16,
	Gwei:                          1e9,
	DepositsForChainStart:         16384,
	TargetCommitteeSize:           uint64(256),
	SlotDuration:                  uint64(16),
	CycleLength:                   uint64(64),
	MinValidatorSetChangeInterval: uint64(256),
	MinAttestationInclusionDelay:  uint64(4),
	SqrtExpDropTime:               uint64(65536),
	MinWithdrawalPeriod:           uint64(4096),
	WithdrawalsPerCycle:           uint64(4),
	BaseRewardQuotient:            uint64(32768),
	MaxValidatorChurnQuotient:     uint64(32),
	InitialForkVersion:            0,
	InitialForkSlot:               0,
	RandBytes:                     3,
	BootstrappedValidatorsCount:   16384,
	SyncPollingInterval:           16 * 4, // Query nodes over the network every 4 slots for sync status.
	GenesisTime:                   time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC),
	MaxNumLog2Validators:          24,
	EpochLength:                   64,
}

var demoBeaconConfig = &BeaconChainConfig{
	MaxProposerSlashings:          16,
	ShardCount:                    5,
	DepositSize:                   32,
	MinTopUpSize:                  1,
	MinOnlineDepositSize:          16,
	Gwei:                          1e9,
	DepositsForChainStart:         16384,
	TargetCommitteeSize:           uint64(3),
	SlotDuration:                  uint64(2),
	CycleLength:                   uint64(5),
	MinValidatorSetChangeInterval: uint64(15),
	MinAttestationInclusionDelay:  uint64(4),
	SqrtExpDropTime:               uint64(65536),
	MinWithdrawalPeriod:           uint64(20),
	WithdrawalsPerCycle:           uint64(2),
	BaseRewardQuotient:            uint64(32768),
	MaxValidatorChurnQuotient:     uint64(32),
	InitialForkVersion:            0,
	RandBytes:                     3,
	InitialForkSlot:               defaultBeaconConfig.InitialForkSlot,
	SimulatedBlockRandao:          [32]byte{'S', 'I', 'M', 'U', 'L', 'A', 'T', 'E', 'R'},
	SyncPollingInterval:           2 * 4, // Query nodes over the network every 4 slots for sync status.
	GenesisTime:                   time.Now(),
	MaxNumLog2Validators:          24,
	EpochLength:                   defaultBeaconConfig.EpochLength,
}

var defaultShardConfig = &ShardChainConfig{
	ChunkSize:         uint64(256),
	MaxShardBlockSize: uint64(32768),
}

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
