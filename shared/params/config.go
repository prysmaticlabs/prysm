// Package params defines important constants that are essential to the
// Ethereum 2.0 services.
package params

import (
	"math/big"
	"time"
)

// BeaconChainConfig contains constant configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	// Misc constants.
	ShardCount                uint64 `yaml:"SHARD_COUNT"`                  // ShardCount is the number of shard chains in Ethereum 2.0.
	TargetCommitteeSize       uint64 `yaml:"TARGET_COMMITTEE_SIZE"`        // TargetCommitteeSize is the number of validators in a committee when the chain is healthy.
	MaxIndicesPerAttestation  uint64 `yaml:"MAX_INDICES_PER_ATTESTATION"`  // MaxIndicesPerAttestation is used to determine how many validators participate in an attestation.
	MinPerEpochChurnLimit     uint64 `yaml:"MIN_PER_EPOCH_CHURN_LIMIT"`    // MinPerEpochChurnLimit is the minimum amount of churn allotted for validator rotations.
	ChurnLimitQuotient        uint64 `yaml:"CHURN_LIMIT_QUOTIENT"`         // ChurnLimitQuotient is used to determine the limit of how many validators can rotate per epoch.
	BaseRewardsPerEpoch       uint64 `yaml:"BASE_REWARDS_PER_EPOCH"`       // BaseRewardsPerEpoch is used to calculate the per epoch rewards.
	ShuffleRoundCount         uint64 `yaml:"SHUFFLE_ROUND_COUNT"`          // ShuffleRoundCount is used for retrieving the permuted index.
	SecondsPerDay             uint64 `yaml:"SECONDS_PER_DAY"`              // SecondsPerDay defines how many seconds are in a given day.
	MaxValidatorsPerCommittee uint64 `yaml:"MAX_VALIDATORS_PER_COMMITTEE"` // MaxValidatorsPerCommittee defines the upper bound of the size of a committee.
	MinGenesisActiveValidatorCount uint64 `yaml:MIN_GENESIS_ACTIVE_VALIDATOR_COUNT` // MinGenesisActiveValidatorCount defines how many validator deposits needed to kick off beacon chain.
	MinGenesisTime            uint64 `yaml:"MIN_GENESIS_TIME"`             // MinGenesisTime defines the lower bound of the genesis time.
	JustificationBitsLength   uint64 `yaml:"JUSTIFICATION_BITS_LENGTH"`    // JustificationBitsLength defines the length in bytes of the justification bits.

	// Deposit contract constants.
	DepositContractTreeDepth uint64 `yaml:"DEPOSIT_CONTRACT_TREE_DEPTH"` // Depth of the Merkle trie of deposits in the validator deposit contract on the PoW chain.

	// Gwei value constants.
	MinDepositAmount          uint64 `yaml:"MIN_DEPOSIT_AMOUNT"`          // MinDepositAmount is the maximal amount of Gwei a validator can send to the deposit contract at once.
	MaxEffectiveBalance       uint64 `yaml:"MAX_EFFECTIVE_BALANCE"`       // MaxEffectiveBalance is the maximal amount of Gwie that is effective for staking.
	EjectionBalance           uint64 `yaml:"EJECTION_BALANCE"`            // EjectionBalance is the minimal GWei a validator needs to have before ejected.
	EffectiveBalanceIncrement uint64 `yaml:"EFFECTIVE_BALANCE_INCREMENT"` // EffectiveBalanceIncrement is used for converting the high balance into the low balance for validators.

	// Initial value constants.
	FarFutureEpoch          uint64 `yaml:"FAR_FUTURE_EPOCH"`           // FarFutureEpoch represents a epoch extremely far away in the future used as the default penalization slot for validators.
	BLSWithdrawalPrefixByte byte   `yaml:"BLS_WITHDRAWAL_PREFIX_BYTE"` // BLSWithdrawalPrefixByte is used for BLS withdrawal and it's the first byte.

	// Time parameters constants.
	SecondsPerSlot               uint64 `yaml:"SECONDS_PER_SLOT"`                    // SecondsPerSlot is how many seconds are in a single slot.
	MinAttestationInclusionDelay uint64 `yaml:"MIN_ATTESTATION_INCLUSION_DELAY"`     // MinAttestationInclusionDelay defines how long validator has to wait to include attestation for beacon block.
	SlotsPerEpoch                uint64 `yaml:"SLOTS_PER_EPOCH"`                     // SlotsPerEpoch is the number of slots in an epoch.
	MinSeedLookahead             uint64 `yaml:"MIN_SEED_LOOKAHEAD"`                  // SeedLookahead is the duration of randao look ahead seed.
	ActivationExitDelay          uint64 `yaml:"ACTIVATION_EXIT_DELAY"`               // ActivationExitDelay is the duration a validator has to wait for entry and exit in epoch.
	SlotsPerEth1VotingPeriod     uint64 `yaml:"SLOTS_PER_ETH1_VOTING_PERIOD"`        // SlotsPerEth1VotingPeriod defines how often the merkle root of deposit receipts get updated in beacon node.
	MinValidatorWithdrawalDelay  uint64 `yaml:"MIN_VALIDATOR_WITHDRAWABILITY_DELAY"` // MinValidatorWithdrawalEpochs is the shortest amount of time a validator has to wait to withdraw.
	PersistentCommitteePeriod    uint64 `yaml:"PERSISTENT_COMMITTEE_PERIOD"`         // PersistentCommitteePeriod is the minimum amount of epochs a validator must participate before exitting.
	MaxEpochsPerCrosslink        uint64 `yaml:"MAX_EPOCHS_PER_CROSSLINK"`            // MaxEpochsPerCrosslink defines the max epoch from current a crosslink can be formed at.
	MinEpochsToInactivityPenalty uint64 `yaml:"MIN_EPOCHS_TO_INACTIVITY_PENALTY"`    // MinEpochsToInactivityPenalty defines the minimum amount of epochs since finality to begin penalizing inactivity.
	Eth1FollowDistance           uint64 // Eth1FollowDistance is the number of eth1.0 blocks to wait before considering a new deposit for voting. This only applies after the chain as been started.

	// State list lengths
	EpochsPerHistoricalVector      uint64 `yaml:"EPOCHS_PER_HISTORICAL_VECTOR"`       // EpochsPerHistoricalVector defines max length in epoch to store old historical stats in beacon state.
	EpochsPerSlashedBalancesVector uint64 `yaml:"EPOCHS_PER_SLASHED_BALANCES_VECTOR"` // EpochsPerSlashedBalancesVector defines max length in epoch to store old stats to recompute slashing witness.
	HistoricalRootsLimit           uint64 `yaml:"HISTORICAL_ROOTS_LIMIT"`             // HistoricalRootsLimit the define max historical roots can be saved in state before roll over.
	ValidatorRegistryLimit         uint64 `yaml:"VALIDATOR_REGISTRY_LIMIT"`           // ValidatorRegistryLimit defines the upper bound of validators can participate in eth2.

	// Reward and penalty quotients constants.
	BaseRewardFactor             uint64 `yaml:"BASE_REWARD_FACTOR"`             // BaseRewardFactor is used to calculate validator per-slot interest rate.
	WhistleBlowingRewardQuotient uint64 `yaml:"WHISTLEBLOWING_REWARD_QUOTIENT"` // WhistleBlowingRewardQuotient is used to calculate whistler blower reward.
	ProposerRewardQuotient       uint64 `yaml:"PROPOSER_REWARD_QUOTIENT"`       // ProposerRewardQuotient is used to calculate the reward for proposers.
	InactivityPenaltyQuotient    uint64 `yaml:"INACTIVITY_PENALTY_QUOTIENT"`    // InactivityPenaltyQuotient is used to calculate the penalty for a validator that is offline.
	MinSlashingPenaltyQuotient   uint64 `yaml:"MIN_SLASHING_PENALTY_QUOTIENT"`  // MinSlashingPenaltyQuotient is used to calculate the minimum penalty to prevent DoS attacks.

	// Max operations per block constants.
	MaxProposerSlashings uint64 `yaml:"MAX_PROPOSER_SLASHINGS"` // MaxProposerSlashings defines the maximum number of slashings of proposers possible in a block.
	MaxAttesterSlashings uint64 `yaml:"MAX_ATTESTER_SLASHINGS"` // MaxAttesterSlashings defines the maximum number of casper FFG slashings possible in a block.
	MaxAttestations      uint64 `yaml:"MAX_ATTESTATIONS"`       // MaxAttestations defines the maximum allowed attestations in a beacon block.
	MaxDeposits          uint64 `yaml:"MAX_DEPOSITS"`           // MaxVoluntaryExits defines the maximum number of validator deposits in a block.
	MaxVoluntaryExits    uint64 `yaml:"MAX_VOLUNTARY_EXITS"`    // MaxVoluntaryExits defines the maximum number of validator exits in a block.
	MaxTransfers         uint64 `yaml:"MAX_TRANSFERS"`          // MaxTransfers defines the maximum number of balance transfers in a block.

	// BLS domain values.
	DomainBeaconProposer uint64 `yaml:"DOMAIN_BEACON_PROPOSER"` // DomainBeaconProposer defines the BLS signature domain for beacon proposal verification.
	DomainRandao         uint64 `yaml:"DOMAIN_RANDAO"`          // DomainRandao defines the BLS signature domain for randao verification.
	DomainAttestation    uint64 `yaml:"DOMAIN_ATTESTATION"`     // DomainAttestation defines the BLS signature domain for attestation verification.
	DomainDeposit        uint64 `yaml:"DOMAIN_DEPOSIT"`         // DomainDeposit defines the BLS signature domain for deposit verification.
	DomainVoluntaryExit  uint64 `yaml:"DOMAIN_VOLUNTARY_EXIT"`  // DomainVoluntaryExit defines the BLS signature domain for exit verification.
	DomainTransfer       uint64 `yaml:"DOMAIN_TRANSFER"`        // DomainTransfer defines the BLS signature domain for transfer verification.

	// Prysm constants.
	GweiPerEth                uint64        // GweiPerEth is the amount of gwei corresponding to 1 eth.
	DepositsForChainStart     uint64        // DepositsForChainStart defines how many validator deposits needed to kick off beacon chain.
	SyncPollingInterval       int64         // SyncPollingInterval queries network nodes for sync status.
	LogBlockDelay             int64         // Number of blocks to wait from the current head before processing logs from the deposit contract.
	BLSPubkeyLength           int           // BLSPubkeyLength defines the expected length of BLS public keys in bytes.
	DefaultBufferSize         int           // DefaultBufferSize for channels across the Prysm repository.
	ValidatorPrivkeyFileName  string        // ValidatorPrivKeyFileName specifies the string name of a validator private key file.
	WithdrawalPrivkeyFileName string        // WithdrawalPrivKeyFileName specifies the string name of a withdrawal private key file.
	RPCSyncCheck              time.Duration // Number of seconds to query the sync service, to find out if the node is synced or not.
	TestnetContractEndpoint   string        // TestnetContractEndpoint to fetch the contract address of the Prysmatic Labs testnet.
	GoerliBlockTime           uint64        // GoerliBlockTime is the number of seconds on avg a Goerli block is created.
	GenesisForkVersion        []byte        `yaml:"GENESIS_FORK_VERSION"` // GenesisForkVersion is used to track fork version between state transitions.
	EmptySignature            [96]byte      // EmptySignature is used to represent a zeroed out BLS Signature.
}

// DepositContractConfig contains the deposits for
type DepositContractConfig struct {
	DepositsForChainStart *big.Int // DepositsForChainStart defines how many validator deposits needed to kick off beacon chain.
	MinDepositAmount      *big.Int // MinDepositAmount defines the minimum deposit amount in gwei that is required in the deposit contract.
	MaxDepositAmount      *big.Int // MaxDepositAmount defines the maximum deposit amount in gwei that is required in the deposit contract.
}

// ShardChainConfig contains configs for node to participate in shard chains.
type ShardChainConfig struct {
	ChunkSize         uint64 // ChunkSize defines the size of each chunk in bytes.
	MaxShardBlockSize uint64 // MaxShardBlockSize defines the max size of each shard block in bytes.
}

var defaultBeaconConfig = &BeaconChainConfig{
	// Misc constant.
	ShardCount:                1024,
	TargetCommitteeSize:       128,
	MaxIndicesPerAttestation:  4096,
	MinPerEpochChurnLimit:     4,
	ChurnLimitQuotient:        1 << 16,
	BaseRewardsPerEpoch:       5,
	ShuffleRoundCount:         90,
	SecondsPerDay:             86400,
	MaxValidatorsPerCommittee: 4096,
	MinGenesisActiveValidatorCount: 65536,
	MinGenesisTime:            1578009600,
	JustificationBitsLength:   4,

	// Deposit contract constants.
	DepositContractTreeDepth: 32,

	// Gwei value constants.
	MinDepositAmount:          1 * 1e9,
	MaxEffectiveBalance:       32 * 1e9,
	EjectionBalance:           16 * 1e9,
	EffectiveBalanceIncrement: 1 * 1e9,

	// Initial value constants.
	FarFutureEpoch:          1<<64 - 1,
	BLSWithdrawalPrefixByte: byte(0),

	// Time parameter constants.
	SecondsPerSlot:               6,
	MinAttestationInclusionDelay: 1,
	SlotsPerEpoch:                64,
	MinSeedLookahead:             1,
	ActivationExitDelay:          4,
	SlotsPerEth1VotingPeriod:     1024,
	MinValidatorWithdrawalDelay:  256,
	PersistentCommitteePeriod:    2048,
	MaxEpochsPerCrosslink:        64,
	MinEpochsToInactivityPenalty: 4,
	Eth1FollowDistance:           1024,

	// State list length constants.
	EpochsPerHistoricalVector:      65536,
	EpochsPerSlashedBalancesVector: 8192,
	HistoricalRootsLimit:           8192,
	ValidatorRegistryLimit:         1099511627776,

	// Reward and penalty quotients constants.
	BaseRewardFactor:             32,
	ProposerRewardQuotient:       8,
	WhistleBlowingRewardQuotient: 512,
	InactivityPenaltyQuotient:    1 << 25,
	MinSlashingPenaltyQuotient:   32,

	// Max operations per block constants.
	MaxProposerSlashings: 16,
	MaxAttesterSlashings: 1,
	MaxAttestations:      128,
	MaxDeposits:          16,
	MaxVoluntaryExits:    16,
	MaxTransfers:         0,

	// BLS domain values.
	DomainBeaconProposer: 0,
	DomainRandao:         1,
	DomainAttestation:    2,
	DomainDeposit:        3,
	DomainVoluntaryExit:  4,
	DomainTransfer:       5,

	// Prysm constants.
	GweiPerEth:                1000000000,
	DepositsForChainStart:     16384,
	LogBlockDelay:             2,
	BLSPubkeyLength:           96,
	DefaultBufferSize:         10000,
	WithdrawalPrivkeyFileName: "/shardwithdrawalkey",
	ValidatorPrivkeyFileName:  "/validatorprivatekey",
	RPCSyncCheck:              1,
	GoerliBlockTime:           14, // 14 seconds on average for a goerli block to be created.
	GenesisForkVersion:        []byte{0, 0, 0, 0},
	EmptySignature:            [96]byte{},

	// Testnet misc values.
	TestnetContractEndpoint: "https://beta.prylabs.net/contract", // defines an http endpoint to fetch the testnet contract addr.
}

var defaultShardConfig = &ShardChainConfig{
	ChunkSize:         uint64(256),
	MaxShardBlockSize: uint64(32768),
}

var defaultDepositContractConfig = &DepositContractConfig{
	DepositsForChainStart: big.NewInt(16384),
	MinDepositAmount:      big.NewInt(1e9),
	MaxDepositAmount:      big.NewInt(32e9),
}

var beaconConfig = defaultBeaconConfig
var shardConfig = defaultShardConfig
var contractConfig = defaultDepositContractConfig

// BeaconConfig retrieves beacon chain config.
func BeaconConfig() *BeaconChainConfig {
	return beaconConfig
}

// MainnetConfig returns the default config to
// be used in the mainnet.
func MainnetConfig() *BeaconChainConfig {
	return defaultBeaconConfig
}

// DemoBeaconConfig retrieves the demo beacon chain config.
func DemoBeaconConfig() *BeaconChainConfig {
	demoConfig := *defaultBeaconConfig
	demoConfig.ShardCount = 1
	demoConfig.MinAttestationInclusionDelay = 1
	demoConfig.TargetCommitteeSize = 1
	demoConfig.DepositsForChainStart = 8
	demoConfig.SlotsPerEpoch = 8
	demoConfig.MinDepositAmount = 100
	demoConfig.MaxEffectiveBalance = 3.2 * 1e9
	demoConfig.EjectionBalance = 3.175 * 1e9
	demoConfig.SyncPollingInterval = 1 * 10 // Query nodes over the network every slot.
	demoConfig.Eth1FollowDistance = 5
	demoConfig.SlotsPerEth1VotingPeriod = 1
	demoConfig.EpochsPerHistoricalVector = 5 * demoConfig.SlotsPerEpoch
	demoConfig.EpochsPerSlashedBalancesVector = 5 * demoConfig.SlotsPerEpoch
	demoConfig.HistoricalRootsLimit = 5 * demoConfig.SlotsPerEpoch

	return &demoConfig
}

// MinimalSpecConfig retrieves the minimal config used in spec tests.
func MinimalSpecConfig() *BeaconChainConfig {

	minimalConfig := *defaultBeaconConfig
	minimalConfig.ShardCount = 8
	minimalConfig.TargetCommitteeSize = 4
	minimalConfig.MaxIndicesPerAttestation = 4096
	minimalConfig.MinPerEpochChurnLimit = 4
	minimalConfig.ChurnLimitQuotient = 65536
	minimalConfig.BaseRewardsPerEpoch = 5
	minimalConfig.ShuffleRoundCount = 10
	minimalConfig.MinGenesisActiveValidatorCount = 64
	minimalConfig.DepositContractTreeDepth = 32
	minimalConfig.MinDepositAmount = 1e9
	minimalConfig.MaxEffectiveBalance = 32e9
	minimalConfig.EjectionBalance = 16e9
	minimalConfig.EffectiveBalanceIncrement = 1e9
	minimalConfig.FarFutureEpoch = 1<<64 - 1
	minimalConfig.BLSWithdrawalPrefixByte = byte(0)
	minimalConfig.SecondsPerSlot = 6
	minimalConfig.MinAttestationInclusionDelay = 2
	minimalConfig.SlotsPerEpoch = 8
	minimalConfig.MinSeedLookahead = 1
	minimalConfig.ActivationExitDelay = 4
	minimalConfig.SlotsPerEth1VotingPeriod = 16
	minimalConfig.HistoricalRootsLimit = 64
	minimalConfig.MinValidatorWithdrawalDelay = 256
	minimalConfig.PersistentCommitteePeriod = 2048
	minimalConfig.MaxEpochsPerCrosslink = 4
	minimalConfig.MinEpochsToInactivityPenalty = 4
	minimalConfig.EpochsPerHistoricalVector = 64
	minimalConfig.EpochsPerSlashedBalancesVector = 64
	minimalConfig.HistoricalRootsLimit = 16777216
	minimalConfig.ValidatorRegistryLimit = 1099511627776
	minimalConfig.BaseRewardFactor = 32
	minimalConfig.WhistleBlowingRewardQuotient = 512
	minimalConfig.ProposerRewardQuotient = 8
	minimalConfig.InactivityPenaltyQuotient = 33554432
	minimalConfig.MinSlashingPenaltyQuotient = 32
	minimalConfig.MaxProposerSlashings = 16
	minimalConfig.MaxAttesterSlashings = 1
	minimalConfig.MaxAttestations = 128
	minimalConfig.MaxDeposits = 16
	minimalConfig.MaxVoluntaryExits = 16
	minimalConfig.MaxTransfers = 0
	minimalConfig.DomainBeaconProposer = 0
	minimalConfig.DomainRandao = 1
	minimalConfig.DomainAttestation = 2
	minimalConfig.DomainDeposit = 3
	minimalConfig.DomainVoluntaryExit = 4
	minimalConfig.DomainTransfer = 5

	return &minimalConfig
}

// ShardConfig retrieves shard chain config.
func ShardConfig() *ShardChainConfig {
	return shardConfig
}

// ContractConfig retrieves the deposit contract config
func ContractConfig() *DepositContractConfig {
	return contractConfig
}

// DemoContractConfig uses the argument provided to initialize a fresh config.
func DemoContractConfig(depositsReq *big.Int, minDeposit *big.Int, maxDeposit *big.Int) *DepositContractConfig {
	return &DepositContractConfig{
		DepositsForChainStart: depositsReq,
		MinDepositAmount:      minDeposit,
		MaxDepositAmount:      maxDeposit,
	}
}

// UseDemoBeaconConfig for beacon chain services.
func UseDemoBeaconConfig() {
	beaconConfig = DemoBeaconConfig()
}

// OverrideBeaconConfig by replacing the config. The preferred pattern is to
// call BeaconConfig(), change the specific parameters, and then call
// OverrideBeaconConfig(c). Any subsequent calls to params.BeaconConfig() will
// return this new configuration.
func OverrideBeaconConfig(c *BeaconChainConfig) {
	beaconConfig = c
}
