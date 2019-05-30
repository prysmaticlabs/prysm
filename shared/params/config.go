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
	ShardCount               uint64 `yaml:"SHARD_COUNT"`                 // ShardCount is the number of shard chains in Ethereum 2.0.
	TargetCommitteeSize      uint64 `yaml:"TARGET_COMMITTEE_SIZE"`       // TargetCommitteeSize is the number of validators in a committee when the chain is healthy.
	MaxIndicesPerAttestation uint64 `yaml:"MAX_INDICES_PER_ATTESTATION"` // MaxIndicesPerAttestation is used to determine how many validators participate in an attestation.
	MinPerEpochChurnLimit    uint64 `yaml:"MIN_PER_EPOCH_CHURN_LIMIT"`   // MinPerEpochChurnLimit is the minimum amount of churn allotted for validator rotations.
	ChurnLimitQuotient       uint64 `yaml:"CHURN_LIMIT_QUOTIENT"`        // ChurnLimitQuotient is used to determine the limit of how many validators can rotate per epoch.
	BaseRewardsPerEpoch      uint64 `yaml:"BASE_REWARDS_PER_EPOCH"`      // BaseRewardsPerEpoch is used to calculate the per epoch rewards.
	ShuffleRoundCount        uint64 `yaml:"SHUFFLE_ROUND_COUNT"`         // ShuffleRoundCount is used for retrieving the permuted index.
	// TODO(2307): Remove deprecated fields
	// Deprecated: Do not use.
	MaxBalanceChurnQuotient uint64 // MaxBalanceChurnQuotient is used to determine how many validators can rotate per epoch.
	// Deprecated: Do not use.
	BeaconChainShardNumber uint64 // BeaconChainShardNumber is the shard number of the beacon chain.
	// Deprecated: Do not use.
	MaxIndicesPerSlashableVote uint64 // MaxIndicesPerSlashableVote is used to determine how many validators can be slashed per vote.
	// Deprecated: Do not use.
	MaxExitDequeuesPerEpoch uint64 // MaxWithdrawalsPerEpoch is the max withdrawals can happen for a single epoch.

	// Deposit contract constants.
	DepositContractAddress   []byte `yaml:"DEPOSIT_CONTRACT_ADDRESS"`    // DepositContractAddress is the address of the deposit contract in PoW chain.
	DepositContractTreeDepth uint64 `yaml:"DEPOSIT_CONTRACT_TREE_DEPTH"` // Depth of the Merkle trie of deposits in the validator deposit contract on the PoW chain.

	// Gwei value constants.
	MinDepositAmount          uint64 `yaml:"MIN_DEPOSIT_AMOUNT"` // MinDepositAmount is the maximal amount of Gwei a validator can send to the deposit contract at once.
	MaxDepositAmount          uint64 // MaxDepositAmount is the maximal amount of Gwei a validator can send to the deposit contract at once.
	MaxEffectiveBalance       uint64 `yaml:"MAX_EFFECTIVE_BALANCE"`       // MaxEffectiveBalance is the maximal amount of Gwie that is effective for staking.
	EjectionBalance           uint64 `yaml:"EJECTION_BALANCE"`            // EjectionBalance is the minimal GWei a validator needs to have before ejected.
	EffectiveBalanceIncrement uint64 `yaml:"EFFECTIVE_BALANCE_INCREMENT"` // EffectiveBalanceIncrement is used for converting the high balance into the low balance for validators.
	// TODO(2307): Remove deprecated fields
	//Deprecated: Do not use.
	ForkChoiceBalanceIncrement uint64 // ForkChoiceBalanceIncrement is used to track block score based on balances for fork choice.

	// Initial value constants.
	FarFutureEpoch          uint64   `yaml:"FAR_FUTURE_EPOCH"` // FarFutureEpoch represents a epoch extremely far away in the future used as the default penalization slot for validators.
	ZeroHash                [32]byte // ZeroHash is used to represent a zeroed out 32 byte array.
	BLSWithdrawalPrefixByte byte     `yaml:"BLS_WITHDRAWAL_PREFIX_BYTE"` // BLSWithdrawalPrefixByte is used for BLS withdrawal and it's the first byte.
	// TODO(2307): Remove deprecated fields
	// Deprecated: Do not use.
	GenesisForkVersion []byte `yaml:"GENESIS_FORK_VERSION"` // GenesisForkVersion is used to track fork version between state transitions.
	// Deprecated: Do not use.
	GenesisStartShard uint64 // GenesisStartShard is the first shard to assign validators.
	// Deprecated: Do not use.
	EmptySignature [96]byte // EmptySignature is used to represent a zeroed out BLS Signature.

	// Time parameters constants.
	SecondsPerSlot               uint64 `yaml:"SECONDS_PER_SLOT"`                    // SecondsPerSlot is how many seconds are in a single slot.
	MinAttestationInclusionDelay uint64 `yaml:"MIN_ATTESTATION_INCLUSION_DELAY"`     // MinAttestationInclusionDelay defines how long validator has to wait to include attestation for beacon block.
	SlotsPerEpoch                uint64 `yaml:"SLOTS_PER_EPOCH"`                     // SlotsPerEpoch is the number of slots in an epoch.
	MinSeedLookahead             uint64 `yaml:"MIN_SEED_LOOKAHEAD"`                  // SeedLookahead is the duration of randao look ahead seed.
	ActivationExitDelay          uint64 `yaml:"ACTIVATION_EXIT_DELAY"`               // ActivationExitDelay is the duration a validator has to wait for entry and exit in epoch.
	SlotsPerEth1VotingPeriod     uint64 `yaml:"SLOTS_PER_ETH1_VOTING_PERIOD"`        // SlotsPerEth1VotingPeriod defines how often the merkle root of deposit receipts get updated in beacon node.
	SlotsPerHistoricalRoot       uint64 `yaml:"SLOTS_PER_HISTORICAL_ROOT"`           // SlotsPerHistoricalRoot defines how often the historical root is saved.
	MinValidatorWithdrawalDelay  uint64 `yaml:"MIN_VALIDATOR_WITHDRAWABILITY_DELAY"` // MinValidatorWithdrawalEpochs is the shortest amount of time a validator has to wait to withdraw.
	PersistentCommitteePeriod    uint64 `yaml:"PERSISTENT_COMMITTEE_PERIOD"`         // PersistentCommitteePeriod is the minimum amount of epochs a validator must participate before exitting.
	MaxCrosslinkEpochs           uint64 `yaml:"MAX_EPOCHS_PER_CROSSLINK"`            // MaxCrosslinkEpochs defines the max epoch from current a crosslink can be formed at.
	MinEpochsToInactivityPenalty uint64 `yaml:"MIN_EPOCHS_TO_INACTIVITY_PENALTY"`    // MinEpochsToInactivityPenalty defines the minimum amount of epochs since finality to begin penalizing inactivity.
	Eth1FollowDistance           uint64 // Eth1FollowDistance is the number of eth1.0 blocks to wait before considering a new deposit for voting. This only applies after the chain as been started.
	// TODO(2307): Remove deprecated fields
	// Deprecated: Do not use.
	EpochsPerEth1VotingPeriod uint64 // EpochsPerEth1VotingPeriod defines how often the merkle root of deposit receipts get updated in beacon node.

	// State list lengths
	LatestRandaoMixesLength      uint64 `yaml:"LATEST_RANDAO_MIXES_LENGTH"`       // LatestRandaoMixesLength is the number of randao mixes kept in the beacon state.
	LatestActiveIndexRootsLength uint64 `yaml:"LATEST_ACTIVE_INDEX_ROOTS_LENGTH"` // LatestIndexRootsLength is the number of index roots kept in beacon state, used by light client.
	LatestSlashedExitLength      uint64 `yaml:"LATEST_SLASHED_EXIT_LENGTH"`       // LatestSlashedExitLength is used to track penalized exit balances per time interval.
	// TODO(2307): Remove deprecated fields
	// Deprecated: Do not use.
	LatestBlockRootsLength uint64 // LatestBlockRootsLength is the number of block roots kept in the beacon state.

	// Reward and penalty quotients constants.
	BaseRewardQuotient           uint64 `yaml:"BASE_REWARD_QUOTIENT"`           // BaseRewardQuotient is used to calculate validator per-slot interest rate.
	WhistleBlowingRewardQuotient uint64 `yaml:"WHISTLEBLOWING_REWARD_QUOTIENT"` // WhistleBlowingRewardQuotient is used to calculate whistler blower reward.
	ProposerRewardQuotient       uint64 `yaml:"PROPOSER_REWARD_QUOTIENT"`       // ProposerRewardQuotient is used to calculate the reward for proposers.
	InactivityPenaltyQuotient    uint64 `yaml:"INACTIVITY_PENALTY_QUOTIENT"`    // InactivityPenaltyQuotient is used to calculate the penalty for a validator that is offline.
	MinSlashingPenaltyQuotient   uint64 `yaml:"MIN_SLASHING_PENALTY_QUOTIENT"`  // MinSlashingPenaltyQuotient is used to calculate the minimum penalty to prevent DoS attacks.
	// TODO(2307): Remove deprecated fields
	// Deprecated: Do not use.
	AttestationInclusionRewardQuotient uint64 // AttestationInclusionRewardQuotient defines the reward quotient of proposer for including attestations.

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
	RandBytes                 uint64        // RandBytes is the number of bytes used as entropy to shuffle validators.
	BatchBlockLimit           uint64        // BatchBlockLimit is maximum number of blocks that can be requested for initial sync.
	SyncEpochLimit            uint64        // SyncEpochLimit is the number of epochs the current node can be behind before it requests for the latest state.
	MaxNumLog2Validators      uint64        // MaxNumLog2Validators is the Max number of validators in Log2 exists given total ETH supply.
	SyncPollingInterval       int64         // SyncPollingInterval queries network nodes for sync status.
	LogBlockDelay             int64         // Number of blocks to wait from the current head before processing logs from the deposit contract.
	BLSPubkeyLength           int           // BLSPubkeyLength defines the expected length of BLS public keys in bytes.
	DefaultBufferSize         int           // DefaultBufferSize for channels across the Prysm repository.
	ValidatorPrivkeyFileName  string        // ValidatorPrivKeyFileName specifies the string name of a validator private key file.
	WithdrawalPrivkeyFileName string        // WithdrawalPrivKeyFileName specifies the string name of a withdrawal private key file.
	HashCacheSize             int64         // HashCacheSize defines the size of object hashes that are cached.
	RPCSyncCheck              time.Duration // Number of seconds to query the sync service, to find out if the node is synced or not.
	TestnetContractEndpoint   string        // TestnetContractEndpoint to fetch the contract address of the Prysmatic Labs testnet.
	GoerliBlockTime           uint64        // GoerliBlockTime is the number of seconds on avg a Goerli block is created.
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
	ShardCount:               1024,
	TargetCommitteeSize:      128,
	MaxIndicesPerAttestation: 4096,
	MinPerEpochChurnLimit:    4,
	ChurnLimitQuotient:       1 << 16,
	BaseRewardsPerEpoch:      5,
	ShuffleRoundCount:        90,

	// TODO(2307): Remove deprecated fields
	// Deprecated.
	MaxBalanceChurnQuotient:    32,
	BeaconChainShardNumber:     1<<64 - 1,
	MaxIndicesPerSlashableVote: 4096,
	MaxExitDequeuesPerEpoch:    4,

	// Deposit contract constants.
	DepositContractTreeDepth: 32,

	// Gwei value constants.
	MinDepositAmount:           1 * 1e9,
	MaxDepositAmount:           32 * 1e9,
	MaxEffectiveBalance:        32 * 1e9,
	EjectionBalance:            16 * 1e9,
	EffectiveBalanceIncrement:  1 * 1e9,
	ForkChoiceBalanceIncrement: 1 * 1e9,

	// Initial value constants.
	FarFutureEpoch:          1<<64 - 1,
	ZeroHash:                [32]byte{},
	BLSWithdrawalPrefixByte: byte(0),

	// TODO(2307): Remove deprecated fields
	// Deprecated.
	GenesisForkVersion: []byte{0, 0, 0, 0},
	GenesisStartShard:  0,
	EmptySignature:     [96]byte{},

	// Time parameter constants.
	SecondsPerSlot:               6,
	MinAttestationInclusionDelay: 4,
	SlotsPerEpoch:                64,
	MinSeedLookahead:             1,
	ActivationExitDelay:          4,
	SlotsPerEth1VotingPeriod:     1024,
	SlotsPerHistoricalRoot:       8192,
	MinValidatorWithdrawalDelay:  256,
	PersistentCommitteePeriod:    2048,
	MaxCrosslinkEpochs:           64,
	MinEpochsToInactivityPenalty: 4,
	Eth1FollowDistance:           1024,

	// State list length constants.
	LatestRandaoMixesLength:      8192,
	LatestActiveIndexRootsLength: 8192,
	LatestSlashedExitLength:      8192,

	// TODO(2307): Remove deprecated fields
	// Deprecated.
	EpochsPerEth1VotingPeriod: 16,
	LatestBlockRootsLength:    8192,

	// Reward and penalty quotients constants.
	BaseRewardQuotient:                 32,
	ProposerRewardQuotient:             8,
	WhistleBlowingRewardQuotient:       512,
	AttestationInclusionRewardQuotient: 8,
	InactivityPenaltyQuotient:          1 << 25,
	MinSlashingPenaltyQuotient:         32,

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
	RandBytes:                 3,
	BatchBlockLimit:           64 * 4, // Process blocks in batches of 4 epochs of blocks (threshold before casper penalties).
	MaxNumLog2Validators:      24,
	LogBlockDelay:             2,
	BLSPubkeyLength:           96,
	DefaultBufferSize:         10000,
	HashCacheSize:             100000,
	WithdrawalPrivkeyFileName: "/shardwithdrawalkey",
	ValidatorPrivkeyFileName:  "/validatorprivatekey",
	RPCSyncCheck:              1,
	GoerliBlockTime:           14, // 14 seconds on average for a goerli block to be created.

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

// DemoBeaconConfig retrieves the demo beacon chain config.
func DemoBeaconConfig() *BeaconChainConfig {
	demoConfig := *defaultBeaconConfig
	demoConfig.ShardCount = 1
	demoConfig.MinAttestationInclusionDelay = 1
	demoConfig.TargetCommitteeSize = 1
	demoConfig.DepositsForChainStart = 8
	demoConfig.SlotsPerEpoch = 8
	demoConfig.MinDepositAmount = 100
	demoConfig.MaxDepositAmount = 3.2 * 1e9
	demoConfig.EjectionBalance = 3.175 * 1e9
	demoConfig.SyncPollingInterval = 1 * 10 // Query nodes over the network every slot.
	demoConfig.Eth1FollowDistance = 5
	demoConfig.EpochsPerEth1VotingPeriod = 1
	demoConfig.LatestRandaoMixesLength = 5 * demoConfig.SlotsPerEpoch
	demoConfig.LatestActiveIndexRootsLength = 5 * demoConfig.SlotsPerEpoch
	demoConfig.LatestSlashedExitLength = 5 * demoConfig.SlotsPerEpoch
	demoConfig.LatestBlockRootsLength = 5 * demoConfig.SlotsPerEpoch

	return &demoConfig
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
