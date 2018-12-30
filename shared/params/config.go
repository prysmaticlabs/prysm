// Package params defines important constants that are essential to the
// Ethereum 2.0 services.
package params

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func makeEmptySignature() [][]byte {
	signature := make([][]byte, 2)
	signature[0] = make([]byte, 48)
	signature[1] = make([]byte, 48)
	return signature
}

// ValidatorStatusCode defines which stage a validator is in.
type ValidatorStatusCode int

// SpecialRecordType defines type of special record this message represents.
type SpecialRecordType int

// ValidatorSetDeltaFlags is used for light client to track validator entries.
type ValidatorSetDeltaFlags int

// BeaconChainConfig contains configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	ZeroBalanceValidatorTTL                 uint64         // ZeroBalanceValidatorTTL specifies the allowed number of slots a validator with 0 balance can live in the state.
	LatestBlockRootsLength                  uint64         // LatestBlockRootsLength is the number of block roots kept in the beacon state.
	LatestRandaoMixesLength                 uint64         // LatestRandaoMixesLength is the number of randao mixes kept in the beacon state.
	MaxExits                                uint64         // MaxExits determines the maximum number of validator exits in a block.
	MaxAttestations                         uint64         // MaxAttestations defines the maximum allowed attestations in a beacon block.
	MaxProposerSlashings                    uint64         // MaxProposerSlashing defines the maximum number of slashings of proposers possible in a block.
	MaxCasperSlashings                      uint64         // MaxCasperSlashings defines the maximum number of casper FFG slashings possible in a block.
	MaxCasperVotes                          uint64         // MaxCasperVotes defines the maximum number of casper FFG votes possible in a block.
	ShardCount                              uint64         // ShardCount is the fixed number of shards in Ethereum 2.0.
	MaxDeposit                              uint64         // MaxDeposit is the max balance a validator can have at stake.
	MinTopUpSize                            uint64         // MinTopUpSize is the minimal amount of Ether a validator can top up.
	MinOnlineDepositSize                    uint64         // MinOnlineDepositSize is the minimal amount of Ether a validator needs to participate.
	Gwei                                    uint64         // Gwei is the denomination of Gwei in Ether.
	MaxDepositInGwei                        uint64         // MaxDepositInGwei is the max balance in Gwei a validator can have at stake.
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
	IncluderRewardQuotient                  uint64         // IncluderRewardQuotient defines the reward quotient of proposer for including attestations..
	MaxValidatorChurnQuotient               uint64         // MaxValidatorChurnQuotient defines the quotient how many validators can change each time.
	POWContractMerkleTreeDepth              uint64         // POWContractMerkleTreeDepth defines the depth of PoW contract merkle tree.
	InitialForkVersion                      uint64         // InitialForkVersion is used to track fork version between state transitions.
	InitialForkSlot                         uint64         // InitialForkSlot is used to initialize the fork slot in the initial Beacon state.
	InitialSlotNumber                       uint64         // InitialSlotNumber is used to initialize the slot number of the genesis block.
	SimulatedBlockRandao                    [32]byte       // SimulatedBlockRandao is a RANDAO seed stubbed in side simulated block to advance local beacon chain.
	RandBytes                               uint64         // RandBytes is the number of bytes used as entropy to shuffle validators.
	BootstrappedValidatorsCount             uint64         // BootstrappedValidatorsCount is the number of validators we seed to start beacon chain.
	SyncPollingInterval                     int64          // SyncPollingInterval queries network nodes for sync status.
	GenesisTime                             time.Time      // GenesisTime used by the protocol.
	MaxNumLog2Validators                    uint64         // Max number of validators in Log2 can exist given total ETH supply.
	EpochLength                             uint64         // Number of slots that define an Epoch.
	InactivityPenaltyQuotient               uint64         // InactivityPenaltyQuotient defines how much validator leaks out balances for offline.
	EjectionBalance                         uint64         // EjectionBalance is the minimal balance a validator needs before ejected.
	EjectionBalanceInGwei                   uint64         // EjectionBalanceInGwei is the minimal balance in Gwei a validator needs before ejected.
	ZeroHash                                [32]byte       // ZeroHash is used to represent a zeroed out 32 byte array.
	EmptySignature                          [][]byte       // EmptySignature is used to represent a zeroed out BLS Signature.

}

// ShardChainConfig contains configs for node to participate in shard chains.
type ShardChainConfig struct {
	ChunkSize         uint64 // ChunkSize defines the size of each chunk in bytes.
	MaxShardBlockSize uint64 // MaxShardBlockSize defines the max size of each shard block in bytes.
}

var defaultBeaconConfig = &BeaconChainConfig{
	ZeroBalanceValidatorTTL:            4194304,
	LatestRandaoMixesLength:            8192,
	LatestBlockRootsLength:             8192,
	MaxExits:                           16,
	MaxAttestations:                    128,
	MaxProposerSlashings:               16,
	MaxCasperSlashings:                 16,
	MaxCasperVotes:                     1024,
	ShardCount:                         1024,
	MaxDeposit:                         32,
	MinTopUpSize:                       1,
	MinOnlineDepositSize:               16,
	Gwei:                               1e9,
	MaxDepositInGwei:                   32 * 1e9,
	DepositsForChainStart:              16384,
	TargetCommitteeSize:                uint64(256),
	SlotDuration:                       uint64(16),
	CycleLength:                        uint64(64),
	MinValidatorSetChangeInterval:      uint64(256),
	MinAttestationInclusionDelay:       uint64(4),
	SqrtExpDropTime:                    uint64(65536),
	MinWithdrawalPeriod:                uint64(4096),
	WithdrawalsPerCycle:                uint64(4),
	BaseRewardQuotient:                 uint64(1024),
	MaxValidatorChurnQuotient:          uint64(32),
	InitialForkVersion:                 0,
	InitialForkSlot:                    0,
	InitialSlotNumber:                  0,
	RandBytes:                          3,
	BootstrappedValidatorsCount:        16384,
	SyncPollingInterval:                16 * 4, // Query nodes over the network every 4 slots for sync status.
	GenesisTime:                        time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC),
	MaxNumLog2Validators:               24,
	EpochLength:                        64,
	ZeroHash:                           [32]byte{},
	EmptySignature:                     makeEmptySignature(),
	PowReceiptRootVotingPeriod:         1024,
	InactivityPenaltyQuotient:          1 << 24,
	CollectivePenaltyCalculationPeriod: 1 << 20,
	IncluderRewardQuotient:             8,
	EjectionBalance:                    16,
	EjectionBalanceInGwei:              16 * 1e9,
}

var demoBeaconConfig = &BeaconChainConfig{
	ZeroBalanceValidatorTTL:            defaultBeaconConfig.ZeroBalanceValidatorTTL,
	LatestRandaoMixesLength:            defaultBeaconConfig.LatestRandaoMixesLength,
	LatestBlockRootsLength:             defaultBeaconConfig.LatestBlockRootsLength,
	MaxExits:                           defaultBeaconConfig.MaxExits,
	MaxAttestations:                    defaultBeaconConfig.MaxAttestations,
	MaxProposerSlashings:               defaultBeaconConfig.MaxProposerSlashings,
	MaxCasperSlashings:                 defaultBeaconConfig.MaxCasperSlashings,
	ShardCount:                         5,
	MaxDeposit:                         defaultBeaconConfig.MaxDeposit,
	MinTopUpSize:                       defaultBeaconConfig.MinTopUpSize,
	MinOnlineDepositSize:               defaultBeaconConfig.MinOnlineDepositSize,
	Gwei:                               defaultBeaconConfig.Gwei,
	MaxDepositInGwei:                   defaultBeaconConfig.MaxDepositInGwei,
	DepositsForChainStart:              defaultBeaconConfig.DepositsForChainStart,
	TargetCommitteeSize:                uint64(3),
	SlotDuration:                       uint64(2),
	CycleLength:                        uint64(5),
	MinValidatorSetChangeInterval:      uint64(15),
	MinAttestationInclusionDelay:       defaultBeaconConfig.MinAttestationInclusionDelay,
	SqrtExpDropTime:                    defaultBeaconConfig.SqrtExpDropTime,
	MinWithdrawalPeriod:                uint64(20),
	WithdrawalsPerCycle:                uint64(2),
	BaseRewardQuotient:                 defaultBeaconConfig.BaseRewardQuotient,
	MaxValidatorChurnQuotient:          defaultBeaconConfig.MaxValidatorChurnQuotient,
	InitialForkVersion:                 defaultBeaconConfig.InitialForkVersion,
	InitialSlotNumber:                  defaultBeaconConfig.InitialSlotNumber,
	RandBytes:                          defaultBeaconConfig.RandBytes,
	InitialForkSlot:                    defaultBeaconConfig.InitialForkSlot,
	SimulatedBlockRandao:               [32]byte{'S', 'I', 'M', 'U', 'L', 'A', 'T', 'E', 'R'},
	SyncPollingInterval:                2 * 4, // Query nodes over the network every 4 slots for sync status.
	GenesisTime:                        time.Now(),
	MaxNumLog2Validators:               defaultBeaconConfig.MaxNumLog2Validators,
	EpochLength:                        defaultBeaconConfig.EpochLength,
	PowReceiptRootVotingPeriod:         defaultBeaconConfig.PowReceiptRootVotingPeriod,
	InactivityPenaltyQuotient:          defaultBeaconConfig.InactivityPenaltyQuotient,
	ZeroHash:                           defaultBeaconConfig.ZeroHash,
	EmptySignature:                     makeEmptySignature(),
	CollectivePenaltyCalculationPeriod: defaultBeaconConfig.CollectivePenaltyCalculationPeriod,
	IncluderRewardQuotient:             defaultBeaconConfig.IncluderRewardQuotient,
	EjectionBalance:                    defaultBeaconConfig.EjectionBalance,
	EjectionBalanceInGwei:              defaultBeaconConfig.EjectionBalanceInGwei,
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
