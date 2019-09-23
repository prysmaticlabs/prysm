// Package params defines important constants that are essential to the
// Ethereum 2.0 services.
package params

import (
	"math/big"
	"time"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// BeaconChainConfig contains constant configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	// Constants (non-configurable)
	FarFutureEpoch           uint64 `yaml:"FAR_FUTURE_EPOCH"`            // FarFutureEpoch represents a epoch extremely far away in the future used as the default penalization slot for validators.
	BaseRewardsPerEpoch      uint64 `yaml:"BASE_REWARDS_PER_EPOCH"`      // BaseRewardsPerEpoch is used to calculate the per epoch rewards.
	DepositContractTreeDepth uint64 `yaml:"DEPOSIT_CONTRACT_TREE_DEPTH"` // Depth of the Merkle trie of deposits in the validator deposit contract on the PoW chain.
	SecondsPerDay            uint64 `yaml:"SECONDS_PER_DAY"`             // SecondsPerDay number of seconds in day constant.

	// Misc constants.
	ShardCount                     uint64 `yaml:"SHARD_COUNT"`                        // ShardCount is the number of shard chains in Ethereum 2.0.
	TargetCommitteeSize            uint64 `yaml:"TARGET_COMMITTEE_SIZE"`              // TargetCommitteeSize is the number of validators in a committee when the chain is healthy.
	MaxValidatorsPerCommittee      uint64 `yaml:"MAX_VALIDATORS_PER_COMMITTEE"`       // MaxValidatorsPerCommittee defines the upper bound of the size of a committee.
	MinPerEpochChurnLimit          uint64 `yaml:"MIN_PER_EPOCH_CHURN_LIMIT"`          // MinPerEpochChurnLimit is the minimum amount of churn allotted for validator rotations.
	ChurnLimitQuotient             uint64 `yaml:"CHURN_LIMIT_QUOTIENT"`               // ChurnLimitQuotient is used to determine the limit of how many validators can rotate per epoch.
	ShuffleRoundCount              uint64 `yaml:"SHUFFLE_ROUND_COUNT"`                // ShuffleRoundCount is used for retrieving the permuted index.
	MinGenesisActiveValidatorCount uint64 `yaml:"MIN_GENESIS_ACTIVE_VALIDATOR_COUNT"` // MinGenesisActiveValidatorCount defines how many validator deposits needed to kick off beacon chain.
	MinGenesisTime                 uint64 `yaml:"MIN_GENESIS_TIME"`                   // MinGenesisTime is the time that needed to pass before kicking off beacon chain. Currently set to Jan/3/2020.

	// Gwei value constants.
	MinDepositAmount          uint64 `yaml:"MIN_DEPOSIT_AMOUNT"`          // MinDepositAmount is the maximal amount of Gwei a validator can send to the deposit contract at once.
	MaxEffectiveBalance       uint64 `yaml:"MAX_EFFECTIVE_BALANCE"`       // MaxEffectiveBalance is the maximal amount of Gwie that is effective for staking.
	EjectionBalance           uint64 `yaml:"EJECTION_BALANCE"`            // EjectionBalance is the minimal GWei a validator needs to have before ejected.
	EffectiveBalanceIncrement uint64 `yaml:"EFFECTIVE_BALANCE_INCREMENT"` // EffectiveBalanceIncrement is used for converting the high balance into the low balance for validators.

	// Initial value constants.
	BLSWithdrawalPrefixByte byte     `yaml:"BLS_WITHDRAWAL_PREFIX_BYTE"` // BLSWithdrawalPrefixByte is used for BLS withdrawal and it's the first byte.
	ZeroHash                [32]byte // ZeroHash is used to represent a zeroed out 32 byte array.

	// Time parameters constants.
	MinAttestationInclusionDelay     uint64 `yaml:"MIN_ATTESTATION_INCLUSION_DELAY"`     // MinAttestationInclusionDelay defines how many slots validator has to wait to include attestation for beacon block.
	SecondsPerSlot                   uint64 `yaml:"SECONDS_PER_SLOT"`                    // SecondsPerSlot is how many seconds are in a single slot.
	SlotsPerEpoch                    uint64 `yaml:"SLOTS_PER_EPOCH"`                     // SlotsPerEpoch is the number of slots in an epoch.
	MinSeedLookahead                 uint64 `yaml:"MIN_SEED_LOOKAHEAD"`                  // SeedLookahead is the duration of randao look ahead seed.
	ActivationExitDelay              uint64 `yaml:"ACTIVATION_EXIT_DELAY"`               // ActivationExitDelay is the duration a validator has to wait for entry and exit in epoch.
	SlotsPerEth1VotingPeriod         uint64 `yaml:"SLOTS_PER_ETH1_VOTING_PERIOD"`        // SlotsPerEth1VotingPeriod defines how often the merkle root of deposit receipts get updated in beacon node.
	SlotsPerHistoricalRoot           uint64 `yaml:"SLOTS_PER_HISTORICAL_ROOT"`           // SlotsPerHistoricalRoot defines how often the historical root is saved.
	MinValidatorWithdrawabilityDelay uint64 `yaml:"MIN_VALIDATOR_WITHDRAWABILITY_DELAY"` // MinValidatorWithdrawabilityDelay is the shortest amount of time a validator has to wait to withdraw.
	PersistentCommitteePeriod        uint64 `yaml:"PERSISTENT_COMMITTEE_PERIOD"`         // PersistentCommitteePeriod is the minimum amount of epochs a validator must participate before exitting.
	MaxEpochsPerCrosslink            uint64 `yaml:"MAX_EPOCHS_PER_CROSSLINK"`            // MaxEpochsPerCrosslink defines the max epoch from current a crosslink can be formed at.
	MinEpochsToInactivityPenalty     uint64 `yaml:"MIN_EPOCHS_TO_INACTIVITY_PENALTY"`    // MinEpochsToInactivityPenalty defines the minimum amount of epochs since finality to begin penalizing inactivity.
	Eth1FollowDistance               uint64 // Eth1FollowDistance is the number of eth1.0 blocks to wait before considering a new deposit for voting. This only applies after the chain as been started.

	// State list lengths
	EpochsPerHistoricalVector uint64 `yaml:"EPOCHS_PER_HISTORICAL_VECTOR"` // EpochsPerHistoricalVector defines max length in epoch to store old historical stats in beacon state.
	EpochsPerSlashingsVector  uint64 `yaml:"EPOCHS_PER_SLASHINGS_VECTOR"`  // EpochsPerSlashingsVector defines max length in epoch to store old stats to recompute slashing witness.
	HistoricalRootsLimit      uint64 `yaml:"HISTORICAL_ROOTS_LIMIT"`       // HistoricalRootsLimit the define max historical roots can be saved in state before roll over.
	ValidatorRegistryLimit    uint64 `yaml:"VALIDATOR_REGISTRY_LIMIT"`     // ValidatorRegistryLimit defines the upper bound of validators can participate in eth2.

	// Reward and penalty quotients constants.
	BaseRewardFactor            uint64 `yaml:"BASE_REWARD_FACTOR"`            // BaseRewardFactor is used to calculate validator per-slot interest rate.
	WhistleBlowerRewardQuotient uint64 `yaml:"WHISTLEBLOWER_REWARD_QUOTIENT"` // WhistleBlowerRewardQuotient is used to calculate whistler blower reward.
	ProposerRewardQuotient      uint64 `yaml:"PROPOSER_REWARD_QUOTIENT"`      // ProposerRewardQuotient is used to calculate the reward for proposers.
	InactivityPenaltyQuotient   uint64 `yaml:"INACTIVITY_PENALTY_QUOTIENT"`   // InactivityPenaltyQuotient is used to calculate the penalty for a validator that is offline.
	MinSlashingPenaltyQuotient  uint64 `yaml:"MIN_SLASHING_PENALTY_QUOTIENT"` // MinSlashingPenaltyQuotient is used to calculate the minimum penalty to prevent DoS attacks.

	// Max operations per block constants.
	MaxProposerSlashings uint64 `yaml:"MAX_PROPOSER_SLASHINGS"` // MaxProposerSlashings defines the maximum number of slashings of proposers possible in a block.
	MaxAttesterSlashings uint64 `yaml:"MAX_ATTESTER_SLASHINGS"` // MaxAttesterSlashings defines the maximum number of casper FFG slashings possible in a block.
	MaxAttestations      uint64 `yaml:"MAX_ATTESTATIONS"`       // MaxAttestations defines the maximum allowed attestations in a beacon block.
	MaxDeposits          uint64 `yaml:"MAX_DEPOSITS"`           // MaxVoluntaryExits defines the maximum number of validator deposits in a block.
	MaxVoluntaryExits    uint64 `yaml:"MAX_VOLUNTARY_EXITS"`    // MaxVoluntaryExits defines the maximum number of validator exits in a block.
	MaxTransfers         uint64 `yaml:"MAX_TRANSFERS"`          // MaxTransfers defines the maximum number of balance transfers in a block.

	// BLS domain values.
	DomainBeaconProposer []byte `yaml:"DOMAIN_BEACON_PROPOSER"` // DomainBeaconProposer defines the BLS signature domain for beacon proposal verification.
	DomainRandao         []byte `yaml:"DOMAIN_RANDAO"`          // DomainRandao defines the BLS signature domain for randao verification.
	DomainAttestation    []byte `yaml:"DOMAIN_ATTESTATION"`     // DomainAttestation defines the BLS signature domain for attestation verification.
	DomainDeposit        []byte `yaml:"DOMAIN_DEPOSIT"`         // DomainDeposit defines the BLS signature domain for deposit verification.
	DomainVoluntaryExit  []byte `yaml:"DOMAIN_VOLUNTARY_EXIT"`  // DomainVoluntaryExit defines the BLS signature domain for exit verification.
	DomainTransfer       []byte `yaml:"DOMAIN_TRANSFER"`        // DomainTransfer defines the BLS signature domain for transfer verification.

	// Prysm constants.
	GweiPerEth                uint64        // GweiPerEth is the amount of gwei corresponding to 1 eth.
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
	DefaultPageSize           int           // DefaultPageSize defines the default page size for RPC server request.
	MaxPageSize               int           // MaxPageSize defines the max page size for RPC server respond.

	// Slasher constants.
	WeakSubjectivityPeriod    uint64 // WeakSubjectivityPeriod defines the time period expressed in number of epochs were proof of stake network should validate block headers and attestations for slashable events.
	PruneSlasherStoragePeriod uint64 // PruneSlasherStoragePeriod defines the time period expressed in number of epochs were proof of stake network should prune attestation and block header store.
}

// DepositContractConfig contains the deposits for
type DepositContractConfig struct {
	MinGenesisActiveValidatorCount *big.Int // MinGenesisActiveValidatorCount defines how many validator deposits needed to kick off beacon chain.
	MinDepositAmount               *big.Int // MinDepositAmount defines the minimum deposit amount in gwei that is required in the deposit contract.
	MaxEffectiveBalance            *big.Int // MaxEffectiveBalance defines the maximum deposit amount in gwei that is required in the deposit contract.
}

var defaultBeaconConfig = &BeaconChainConfig{
	// Constants (Non-configurable)
	FarFutureEpoch:           1<<64 - 1,
	BaseRewardsPerEpoch:      5,
	DepositContractTreeDepth: 32,
	SecondsPerDay:            86400,

	// Misc constant.
	ShardCount:                     1024,
	TargetCommitteeSize:            128,
	MaxValidatorsPerCommittee:      4096,
	MinPerEpochChurnLimit:          4,
	ChurnLimitQuotient:             1 << 16,
	ShuffleRoundCount:              90,
	MinGenesisActiveValidatorCount: 65536,
	MinGenesisTime:                 1578009600,

	// Gwei value constants.
	MinDepositAmount:          1 * 1e9,
	MaxEffectiveBalance:       32 * 1e9,
	EjectionBalance:           16 * 1e9,
	EffectiveBalanceIncrement: 1 * 1e9,

	// Initial value constants.
	BLSWithdrawalPrefixByte: byte(0),
	ZeroHash:                [32]byte{},

	// Time parameter constants.
	MinAttestationInclusionDelay:     1,
	SecondsPerSlot:                   6,
	SlotsPerEpoch:                    64,
	MinSeedLookahead:                 1,
	ActivationExitDelay:              4,
	SlotsPerEth1VotingPeriod:         1024,
	SlotsPerHistoricalRoot:           8192,
	MinValidatorWithdrawabilityDelay: 256,
	PersistentCommitteePeriod:        2048,
	MaxEpochsPerCrosslink:            64,
	MinEpochsToInactivityPenalty:     4,
	Eth1FollowDistance:               1024,

	// State list length constants.
	EpochsPerHistoricalVector: 65536,
	EpochsPerSlashingsVector:  8192,
	HistoricalRootsLimit:      16777216,
	ValidatorRegistryLimit:    1099511627776,

	// Reward and penalty quotients constants.
	BaseRewardFactor:            64,
	WhistleBlowerRewardQuotient: 512,
	ProposerRewardQuotient:      8,
	InactivityPenaltyQuotient:   1 << 25,
	MinSlashingPenaltyQuotient:  32,

	// Max operations per block constants.
	MaxProposerSlashings: 16,
	MaxAttesterSlashings: 1,
	MaxAttestations:      128,
	MaxDeposits:          16,
	MaxVoluntaryExits:    16,
	MaxTransfers:         0,

	// BLS domain values.
	DomainBeaconProposer: bytesutil.Bytes4(0),
	DomainRandao:         bytesutil.Bytes4(1),
	DomainAttestation:    bytesutil.Bytes4(2),
	DomainDeposit:        bytesutil.Bytes4(3),
	DomainVoluntaryExit:  bytesutil.Bytes4(4),
	DomainTransfer:       bytesutil.Bytes4(5),

	// Prysm constants.
	GweiPerEth:                1000000000,
	LogBlockDelay:             2,
	BLSPubkeyLength:           48,
	DefaultBufferSize:         10000,
	WithdrawalPrivkeyFileName: "/shardwithdrawalkey",
	ValidatorPrivkeyFileName:  "/validatorprivatekey",
	RPCSyncCheck:              1,
	GoerliBlockTime:           14, // 14 seconds on average for a goerli block to be created.
	GenesisForkVersion:        []byte{0, 0, 0, 0},
	EmptySignature:            [96]byte{},
	DefaultPageSize:           250,
	MaxPageSize:               500,

	// Slasher related values.
	WeakSubjectivityPeriod:    54000,
	PruneSlasherStoragePeriod: 10,

	// Testnet misc values.
	TestnetContractEndpoint: "https://beta.prylabs.net/contract", // defines an http endpoint to fetch the testnet contract addr.
}

var defaultDepositContractConfig = &DepositContractConfig{
	MinGenesisActiveValidatorCount: big.NewInt(16384),
	MinDepositAmount:               big.NewInt(1e9),
	MaxEffectiveBalance:            big.NewInt(32e9),
}

var beaconConfig = defaultBeaconConfig
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
// Notable changes from minimal config:
//   - Max effective balance is 3.2 ETH
//   - Ejection threshold is 3.175 ETH
//   - Genesis threshold is disabled (minimum date to start the chain)
func DemoBeaconConfig() *BeaconChainConfig {
	demoConfig := MinimalSpecConfig()
	demoConfig.MinDepositAmount = 100
	demoConfig.MaxEffectiveBalance = 3.2 * 1e9
	demoConfig.EjectionBalance = 1.6 * 1e9
	demoConfig.EffectiveBalanceIncrement = 0.1 * 1e9
	demoConfig.Eth1FollowDistance = 16

	return demoConfig
}

// MinimalSpecConfig retrieves the minimal config used in spec tests.
func MinimalSpecConfig() *BeaconChainConfig {
	minimalConfig := *defaultBeaconConfig
	// Misc
	minimalConfig.ShardCount = 8
	minimalConfig.TargetCommitteeSize = 4
	minimalConfig.MaxValidatorsPerCommittee = 4096
	minimalConfig.MinPerEpochChurnLimit = 4
	minimalConfig.ChurnLimitQuotient = 65536
	minimalConfig.ShuffleRoundCount = 10
	minimalConfig.MinGenesisActiveValidatorCount = 64
	minimalConfig.MinGenesisTime = 0

	// Gwei values
	minimalConfig.MinDepositAmount = 1e9
	minimalConfig.MaxEffectiveBalance = 32e9
	minimalConfig.EjectionBalance = 16e9
	minimalConfig.EffectiveBalanceIncrement = 1e9

	// Initial values
	minimalConfig.BLSWithdrawalPrefixByte = byte(0)

	// Time parameters
	minimalConfig.SecondsPerSlot = 6
	minimalConfig.MinAttestationInclusionDelay = 1
	minimalConfig.SlotsPerEpoch = 8
	minimalConfig.MinSeedLookahead = 1
	minimalConfig.ActivationExitDelay = 4
	minimalConfig.SlotsPerEth1VotingPeriod = 16
	minimalConfig.SlotsPerHistoricalRoot = 64
	minimalConfig.MinValidatorWithdrawabilityDelay = 256
	minimalConfig.PersistentCommitteePeriod = 2048
	minimalConfig.MaxEpochsPerCrosslink = 4
	minimalConfig.MinEpochsToInactivityPenalty = 4

	// State vector lengths
	minimalConfig.EpochsPerHistoricalVector = 64
	minimalConfig.EpochsPerSlashingsVector = 64
	minimalConfig.HistoricalRootsLimit = 16777216
	minimalConfig.ValidatorRegistryLimit = 1099511627776

	// Reward and penalty quotients
	minimalConfig.BaseRewardFactor = 64
	minimalConfig.WhistleBlowerRewardQuotient = 512
	minimalConfig.ProposerRewardQuotient = 8
	minimalConfig.InactivityPenaltyQuotient = 33554432
	minimalConfig.MinSlashingPenaltyQuotient = 32
	minimalConfig.BaseRewardsPerEpoch = 5

	// Max operations per block
	minimalConfig.MaxProposerSlashings = 16
	minimalConfig.MaxAttesterSlashings = 1
	minimalConfig.MaxAttestations = 128
	minimalConfig.MaxDeposits = 16
	minimalConfig.MaxVoluntaryExits = 16
	minimalConfig.MaxTransfers = 0

	// Signature domains
	minimalConfig.DomainBeaconProposer = bytesutil.Bytes4(0)
	minimalConfig.DomainRandao = bytesutil.Bytes4(1)
	minimalConfig.DomainAttestation = bytesutil.Bytes4(2)
	minimalConfig.DomainDeposit = bytesutil.Bytes4(3)
	minimalConfig.DomainVoluntaryExit = bytesutil.Bytes4(4)
	minimalConfig.DomainTransfer = bytesutil.Bytes4(5)

	minimalConfig.DepositContractTreeDepth = 32
	minimalConfig.FarFutureEpoch = 1<<64 - 1
	return &minimalConfig
}

// ContractConfig retrieves the deposit contract config
func ContractConfig() *DepositContractConfig {
	return contractConfig
}

// UseDemoBeaconConfig for beacon chain services.
func UseDemoBeaconConfig() {
	beaconConfig = DemoBeaconConfig()
}

// UseMinimalConfig for beacon chain services.
func UseMinimalConfig() {
	beaconConfig = MinimalSpecConfig()
}

// OverrideBeaconConfig by replacing the config. The preferred pattern is to
// call BeaconConfig(), change the specific parameters, and then call
// OverrideBeaconConfig(c). Any subsequent calls to params.BeaconConfig() will
// return this new configuration.
func OverrideBeaconConfig(c *BeaconChainConfig) {
	beaconConfig = c
}
