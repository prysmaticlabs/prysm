// Package params defines important constants that are essential to the
// Ethereum 2.0 services.
package params

import (
	"time"
)

func makeEmptySignature() [][]byte {
	signature := make([][]byte, 2)
	signature[0] = make([]byte, 48)
	signature[1] = make([]byte, 48)
	return signature
}

// BeaconChainConfig contains constant configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	// Misc constants.
	ShardCount                uint64 // ShardCount is the number of shard chains in Ethereum 2.0.
	TargetCommitteeSize       uint64 // TargetCommitteeSize is the number of validators in a committee when the chain is healthy.
	EjectionBalance           uint64 // EjectionBalance is the minimal ETH a validator needs to have before ejected.
	EjectionBalanceInGwei     uint64 // EjectionBalance is the minimal GWei a validator needs to have before ejected.
	MaxBalanceChurnQuotient   uint64 // MaxBalanceChurnQuotient is used to determine how many validators can rotate per epoch.
	Gwei                      uint64 // Gwei is the denomination of Gwei in Ether.
	BeaconChainShardNumber    uint64 // BeaconChainShardNumber is the shard number of the beacon chain.
	MaxCasperVotes            uint64 // MaxCasperVotes is used to verify slashable Casper vote data.
	LatestBlockRootsLength    uint64 // LatestBlockRootsLength is the number of block roots kept in the beacon state.
	LatestRandaoMixesLength   uint64 // LatestRandaoMixesLength is the number of randao mixes kept in the beacon state.
	LatestPenalizedExitLength uint64 // LatestPenalizedExitLength is used to track penalized exit balances per time interval.
	MaxWithdrawalsPerEpoch    uint64 // MaxWithdrawalsPerEpoch is the max withdrawals can happen for a single epoch.

	// Deposit contract constants.
	DepositContractAddress   []byte // DepositContractAddress is the address of the deposit contract in PoW chain.
	DepositContractTreeDepth uint64 // Depth of the Merkle trie of deposits in the validator deposit contract on the PoW chain.
	MaxDeposit               uint64 // MaxDeposit is the maximal amount of ETH a validator can send to the deposit contract at once.
	MaxDepositInGwei         uint64 // MaxDepositInGwei is the maximal amount of Gwei a validator can send to the deposit contract at once.
	MinDeposit               uint64 // MinDeposit is the minimal amount of ETH a validator can send to the deposit contract at once.
	MinDepositinGwei         uint64 // MinDepositinGwei is the maximal amount of Gwei a validator can send to the deposit contract at once.

	// Initial value constants.
	GenesisForkVersion      uint64   // GenesisForkVersion is used to track fork version between state transitions.
	GenesisSlot             uint64   // GenesisSlot is used to initialize the genesis state fields..
	ZeroHash                [32]byte // ZeroHash is used to represent a zeroed out 32 byte array.
	EmptySignature          [][]byte // EmptySignature is used to represent a zeroed out BLS Signature.
	BLSWithdrawalPrefixByte byte     // BLSWithdrawalPrefixByte is used for BLS withdrawal and it's the first byte.

	// Time parameters constants.
	SlotDuration                 uint64 // SlotDuration is how many seconds are in a single slot.
	MinAttestationInclusionDelay uint64 // MinAttestationInclusionDelay defines how long validator has to wait to include attestation for beacon block.
	EpochLength                  uint64 // EpochLength is the number of slots in an epoch.
	SeedLookahead                uint64 // SeedLookahead is the duration of randao look ahead seed.
	EntryExitDelay               uint64 // EntryExitDelay is the duration a validator has to wait for entry and exit.
	DepositRootVotingPeriod      uint64 // DepositRootVotingPeriod defines how often the merkle root of deposit receipts get updated in beacon node.
	MinValidatorWithdrawalTime   uint64 // MinValidatorWithdrawalTime is the shortest amount of time a validator can get the deposit out.
	FarFutureSlot                uint64 // FarFutureSlot represents a slot extremely far away in the future used as the default penalization slot for validators.

	// Reward and penalty quotients constants.
	BaseRewardQuotient           uint64 // BaseRewardQuotient is used to calculate validator per-slot interest rate.
	WhistlerBlowerRewardQuotient uint64 // WhistlerBlowerRewardQuotient is used to calculate whistler blower reward.
	IncluderRewardQuotient       uint64 // IncluderRewardQuotient defines the reward quotient of proposer for including attestations..
	InactivityPenaltyQuotient    uint64 // InactivityPenaltyQuotient defines how much validator leaks out balances for offline.

	// Max operations per block constants.
	MaxExits             uint64 // MaxExits determines the maximum number of validator exits in a block.
	MaxDeposits          uint64 // MaxExits determines the maximum number of validator deposits in a block.
	MaxAttestations      uint64 // MaxAttestations defines the maximum allowed attestations in a beacon block.
	MaxProposerSlashings uint64 // MaxProposerSlashings defines the maximum number of slashings of proposers possible in a block.
	MaxCasperSlashings   uint64 // MaxCasperSlashings defines the maximum number of casper FFG slashings possible in a block.

	// Prysm constants.
	DepositsForChainStart uint64    // DepositsForChainStart defines how many validator deposits needed to kick off beacon chain.
	SimulatedBlockRandao  [32]byte  // SimulatedBlockRandao is a RANDAO seed stubbed in simulated block for advancing local beacon chain.
	RandBytes             uint64    // RandBytes is the number of bytes used as entropy to shuffle validators.
	SyncPollingInterval   int64     // SyncPollingInterval queries network nodes for sync status.
	GenesisTime           time.Time // GenesisTime used by the protocol.
	MaxNumLog2Validators  uint64    // MaxNumLog2Validators is the Max number of validators in Log2 exists given total ETH supply.
}

// ShardChainConfig contains configs for node to participate in shard chains.
type ShardChainConfig struct {
	ChunkSize         uint64 // ChunkSize defines the size of each chunk in bytes.
	MaxShardBlockSize uint64 // MaxShardBlockSize defines the max size of each shard block in bytes.
}

var defaultBeaconConfig = &BeaconChainConfig{
	// Misc constant.
	ShardCount:                1024,
	TargetCommitteeSize:       256,
	EjectionBalance:           16,
	EjectionBalanceInGwei:     16 * 1e9,
	MaxBalanceChurnQuotient:   32,
	Gwei:                      1e9,
	BeaconChainShardNumber:    1<<64 - 1,
	MaxCasperVotes:            1024,
	LatestBlockRootsLength:    8192,
	LatestRandaoMixesLength:   8192,
	LatestPenalizedExitLength: 8192,
	MaxWithdrawalsPerEpoch:    4,

	// Deposit contract constants.
	DepositContractTreeDepth: 32,
	MaxDeposit:               32,
	MaxDepositInGwei:         32 * 1e9,
	MinDeposit:               1,
	MinDepositinGwei:         1 * 1e9,

	// Initial value constants.
	GenesisForkVersion: 0,
	GenesisSlot:        0,
	FarFutureSlot:      1<<64 - 1,
	ZeroHash:           [32]byte{},
	EmptySignature:     makeEmptySignature(),

	// Time parameter constants.
	SlotDuration:                 16,
	MinAttestationInclusionDelay: 4,
	EpochLength:                  64,
	SeedLookahead:                64,
	EntryExitDelay:               256,
	DepositRootVotingPeriod:      1024,

	// Reward and penalty quotients constants.
	BaseRewardQuotient:           1024,
	WhistlerBlowerRewardQuotient: 512,
	IncluderRewardQuotient:       8,
	InactivityPenaltyQuotient:    1 << 24,

	// Max operations per block constants.
	MaxExits:             16,
	MaxDeposits:          16,
	MaxAttestations:      128,
	MaxProposerSlashings: 16,
	MaxCasperSlashings:   16,

	// Prysm constants.
	DepositsForChainStart: 16384,
	RandBytes:             3,
	SyncPollingInterval:   16 * 4, // Query nodes over the network every 4 slots for sync status.
	GenesisTime:           time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC),
	MaxNumLog2Validators:  24,
}

var demoBeaconConfig = &BeaconChainConfig{
	// Misc constant.
	ShardCount:                5,
	TargetCommitteeSize:       3,
	EjectionBalance:           defaultBeaconConfig.EjectionBalance,
	EjectionBalanceInGwei:     defaultBeaconConfig.EjectionBalanceInGwei,
	MaxBalanceChurnQuotient:   defaultBeaconConfig.MaxBalanceChurnQuotient,
	Gwei:                      defaultBeaconConfig.Gwei,
	BeaconChainShardNumber:    defaultBeaconConfig.BeaconChainShardNumber,
	MaxCasperVotes:            defaultBeaconConfig.MaxCasperVotes,
	LatestBlockRootsLength:    defaultBeaconConfig.LatestBlockRootsLength,
	LatestRandaoMixesLength:   defaultBeaconConfig.LatestRandaoMixesLength,
	LatestPenalizedExitLength: defaultBeaconConfig.LatestPenalizedExitLength,
	MaxWithdrawalsPerEpoch:    defaultBeaconConfig.MaxWithdrawalsPerEpoch,

	// Deposit contract constants.
	DepositContractTreeDepth: defaultBeaconConfig.DepositContractTreeDepth,
	MaxDeposit:               defaultBeaconConfig.MaxDeposit,
	MaxDepositInGwei:         defaultBeaconConfig.MaxDepositInGwei,
	MinDeposit:               defaultBeaconConfig.MinDeposit,

	// Initial value constants.
	GenesisForkVersion: defaultBeaconConfig.GenesisForkVersion,
	GenesisSlot:        defaultBeaconConfig.GenesisSlot,
	FarFutureSlot:      defaultBeaconConfig.FarFutureSlot,
	ZeroHash:           defaultBeaconConfig.ZeroHash,
	EmptySignature:     defaultBeaconConfig.EmptySignature,

	// Time parameter constants.
	SlotDuration:                 2,
	MinAttestationInclusionDelay: defaultBeaconConfig.MinAttestationInclusionDelay,
	EpochLength:                  defaultBeaconConfig.EpochLength,
	SeedLookahead:                defaultBeaconConfig.SeedLookahead,
	EntryExitDelay:               defaultBeaconConfig.EntryExitDelay,
	DepositRootVotingPeriod:      defaultBeaconConfig.DepositRootVotingPeriod,

	// Reward and penalty quotients constants.
	BaseRewardQuotient:           defaultBeaconConfig.BaseRewardQuotient,
	WhistlerBlowerRewardQuotient: defaultBeaconConfig.WhistlerBlowerRewardQuotient,
	IncluderRewardQuotient:       defaultBeaconConfig.IncluderRewardQuotient,
	InactivityPenaltyQuotient:    defaultBeaconConfig.InactivityPenaltyQuotient,

	// Max operations per block constants.
	MaxExits:             defaultBeaconConfig.MaxExits,
	MaxDeposits:          defaultBeaconConfig.MaxDeposit,
	MaxAttestations:      defaultBeaconConfig.MaxAttestations,
	MaxProposerSlashings: defaultBeaconConfig.MaxProposerSlashings,
	MaxCasperSlashings:   defaultBeaconConfig.MaxCasperSlashings,

	// Prysm constants.
	DepositsForChainStart: defaultBeaconConfig.DepositsForChainStart,
	RandBytes:             defaultBeaconConfig.RandBytes,
	SyncPollingInterval:   2 * 4, // Query nodes over the network every 4 slots for sync status.
	GenesisTime:           time.Now(),
	MaxNumLog2Validators:  defaultBeaconConfig.MaxNumLog2Validators,
	SimulatedBlockRandao:  [32]byte{'S', 'I', 'M', 'U', 'L', 'A', 'T', 'O', 'R'},
}

var defaultShardConfig = &ShardChainConfig{
	ChunkSize:         uint64(256),
	MaxShardBlockSize: uint64(32768),
}

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
