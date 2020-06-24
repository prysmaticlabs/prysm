// Package params defines important constants that are essential to eth2 services.
package params

import (
	"testing"
	"time"

	"github.com/mohae/deepcopy"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// BeaconChainConfig contains constant configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	// Constants (non-configurable)
	GenesisSlot              uint64 `yaml:"GENESIS_SLOT"`                // GenesisSlot represents the first canonical slot number of the beacon chain.
	GenesisEpoch             uint64 `yaml:"GENESIS_EPOCH"`               // GenesisEpoch represents the first canonical epoch number of the beacon chain.
	FarFutureEpoch           uint64 `yaml:"FAR_FUTURE_EPOCH"`            // FarFutureEpoch represents a epoch extremely far away in the future used as the default penalization slot for validators.
	BaseRewardsPerEpoch      uint64 `yaml:"BASE_REWARDS_PER_EPOCH"`      // BaseRewardsPerEpoch is used to calculate the per epoch rewards.
	DepositContractTreeDepth uint64 `yaml:"DEPOSIT_CONTRACT_TREE_DEPTH"` // DepositContractTreeDepth depth of the Merkle trie of deposits in the validator deposit contract on the PoW chain.
	JustificationBitsLength  uint64 `yaml:"JUSTIFICATION_BITS_LENGTH"`   // JustificationBitsLength used

	// Misc constants.
	TargetCommitteeSize            uint64 `yaml:"TARGET_COMMITTEE_SIZE"`              // TargetCommitteeSize is the number of validators in a committee when the chain is healthy.
	MaxValidatorsPerCommittee      uint64 `yaml:"MAX_VALIDATORS_PER_COMMITTEE"`       // MaxValidatorsPerCommittee defines the upper bound of the size of a committee.
	MaxCommitteesPerSlot           uint64 `yaml:"MAX_COMMITTEES_PER_SLOT"`            // MaxCommitteesPerSlot defines the max amount of committee in a single slot.
	MinPerEpochChurnLimit          uint64 `yaml:"MIN_PER_EPOCH_CHURN_LIMIT"`          // MinPerEpochChurnLimit is the minimum amount of churn allotted for validator rotations.
	ChurnLimitQuotient             uint64 `yaml:"CHURN_LIMIT_QUOTIENT"`               // ChurnLimitQuotient is used to determine the limit of how many validators can rotate per epoch.
	ShuffleRoundCount              uint64 `yaml:"SHUFFLE_ROUND_COUNT"`                // ShuffleRoundCount is used for retrieving the permuted index.
	MinGenesisActiveValidatorCount uint64 `yaml:"MIN_GENESIS_ACTIVE_VALIDATOR_COUNT"` // MinGenesisActiveValidatorCount defines how many validator deposits needed to kick off beacon chain.
	MinGenesisTime                 uint64 `yaml:"MIN_GENESIS_TIME"`                   // MinGenesisTime is the time that needed to pass before kicking off beacon chain.
	TargetAggregatorsPerCommittee  uint64 // TargetAggregatorsPerCommittee defines the number of aggregators inside one committee.
	HysteresisQuotient             uint64 `yaml:"HYSTERESIS_QUOTIENT"`            // HysteresisQuotient defines the hysteresis quotient for effective balance calculations.
	HysteresisDownwardMultiplier   uint64 `yaml:"HYSTERESIS_DOWNWARD_MULTIPLIER"` // HysteresisDownwardMultiplier defines the hysteresis downward multiplier for effective balance calculations.
	HysteresisUpwardMultiplier     uint64 `yaml:"HYSTERESIS_UPWARD_MULTIPLIER"`   // HysteresisUpwardMultiplier defines the hysteresis upward multiplier for effective balance calculations.

	// Gwei value constants.
	MinDepositAmount          uint64 `yaml:"MIN_DEPOSIT_AMOUNT"`          // MinDepositAmount is the maximal amount of Gwei a validator can send to the deposit contract at once.
	MaxEffectiveBalance       uint64 `yaml:"MAX_EFFECTIVE_BALANCE"`       // MaxEffectiveBalance is the maximal amount of Gwei that is effective for staking.
	EjectionBalance           uint64 `yaml:"EJECTION_BALANCE"`            // EjectionBalance is the minimal GWei a validator needs to have before ejected.
	EffectiveBalanceIncrement uint64 `yaml:"EFFECTIVE_BALANCE_INCREMENT"` // EffectiveBalanceIncrement is used for converting the high balance into the low balance for validators.

	// Initial value constants.
	BLSWithdrawalPrefixByte byte     `yaml:"BLS_WITHDRAWAL_PREFIX"` // BLSWithdrawalPrefixByte is used for BLS withdrawal and it's the first byte.
	ZeroHash                [32]byte // ZeroHash is used to represent a zeroed out 32 byte array.

	// Time parameters constants.
	GenesisDelay                     uint64 `yaml:"GENESIS_DELAY"`                       // Minimum number of seconds to delay starting the ETH2 genesis. Must be at least 1 second.
	MinAttestationInclusionDelay     uint64 `yaml:"MIN_ATTESTATION_INCLUSION_DELAY"`     // MinAttestationInclusionDelay defines how many slots validator has to wait to include attestation for beacon block.
	SecondsPerSlot                   uint64 `yaml:"SECONDS_PER_SLOT"`                    // SecondsPerSlot is how many seconds are in a single slot.
	SlotsPerEpoch                    uint64 `yaml:"SLOTS_PER_EPOCH"`                     // SlotsPerEpoch is the number of slots in an epoch.
	MinSeedLookahead                 uint64 `yaml:"MIN_SEED_LOOKAHEAD"`                  // SeedLookahead is the duration of randao look ahead seed.
	MaxSeedLookahead                 uint64 `yaml:"MAX_SEED_LOOKAHEAD"`                  // MaxSeedLookahead is the duration a validator has to wait for entry and exit in epoch.
	EpochsPerEth1VotingPeriod        uint64 `yaml:"EPOCHS_PER_ETH1_VOTING_PERIOD"`       // EpochsPerEth1VotingPeriod defines how often the merkle root of deposit receipts get updated in beacon node on per epoch basis.
	SlotsPerHistoricalRoot           uint64 `yaml:"SLOTS_PER_HISTORICAL_ROOT"`           // SlotsPerHistoricalRoot defines how often the historical root is saved.
	MinValidatorWithdrawabilityDelay uint64 `yaml:"MIN_VALIDATOR_WITHDRAWABILITY_DELAY"` // MinValidatorWithdrawabilityDelay is the shortest amount of time a validator has to wait to withdraw.
	ShardCommitteePeriod             uint64 `yaml:"SHARD_COMMITTEE_PERIOD"`              // ShardCommitteePeriod is the minimum amount of epochs a validator must participate before exiting.
	MinEpochsToInactivityPenalty     uint64 `yaml:"MIN_EPOCHS_TO_INACTIVITY_PENALTY"`    // MinEpochsToInactivityPenalty defines the minimum amount of epochs since finality to begin penalizing inactivity.
	Eth1FollowDistance               uint64 // Eth1FollowDistance is the number of eth1.0 blocks to wait before considering a new deposit for voting. This only applies after the chain as been started.
	SafeSlotsToUpdateJustified       uint64 // SafeSlotsToUpdateJustified is the minimal slots needed to update justified check point.
	SecondsPerETH1Block              uint64 `yaml:"SECONDS_PER_ETH1_BLOCK"` // SecondsPerETH1Block is the approximate time for a single eth1 block to be produced.
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

	// BLS domain values.
	DomainBeaconProposer    [4]byte `yaml:"DOMAIN_BEACON_PROPOSER"`     // DomainBeaconProposer defines the BLS signature domain for beacon proposal verification.
	DomainRandao            [4]byte `yaml:"DOMAIN_RANDAO"`              // DomainRandao defines the BLS signature domain for randao verification.
	DomainBeaconAttester    [4]byte `yaml:"DOMAIN_BEACON_ATTESTER"`     // DomainBeaconAttester defines the BLS signature domain for attestation verification.
	DomainDeposit           [4]byte `yaml:"DOMAIN_DEPOSIT"`             // DomainDeposit defines the BLS signature domain for deposit verification.
	DomainVoluntaryExit     [4]byte `yaml:"DOMAIN_VOLUNTARY_EXIT"`      // DomainVoluntaryExit defines the BLS signature domain for exit verification.
	DomainSelectionProof    [4]byte `yaml:"DOMAIN_SELECTION_PROOF"`     // DomainSelectionProof defines the BLS signature domain for selection proof.
	DomainAggregateAndProof [4]byte `yaml:"DOMAIN_AGGREGATE_AND_PROOF"` // DomainAggregateAndProof defines the BLS signature domain for aggregate and proof.

	// Prysm constants.
	GweiPerEth                uint64        // GweiPerEth is the amount of gwei corresponding to 1 eth.
	BLSSecretKeyLength        int           // BLSSecretKeyLength defines the expected length of BLS secret keys in bytes.
	BLSPubkeyLength           int           // BLSPubkeyLength defines the expected length of BLS public keys in bytes.
	BLSSignatureLength        int           // BLSSignatureLength defines the expected length of BLS signatures in bytes.
	DefaultBufferSize         int           // DefaultBufferSize for channels across the Prysm repository.
	ValidatorPrivkeyFileName  string        // ValidatorPrivKeyFileName specifies the string name of a validator private key file.
	WithdrawalPrivkeyFileName string        // WithdrawalPrivKeyFileName specifies the string name of a withdrawal private key file.
	RPCSyncCheck              time.Duration // Number of seconds to query the sync service, to find out if the node is synced or not.
	EmptySignature            [96]byte      // EmptySignature is used to represent a zeroed out BLS Signature.
	DefaultPageSize           int           // DefaultPageSize defines the default page size for RPC server request.
	MaxPeersToSync            int           // MaxPeersToSync describes the limit for number of peers in round robin sync.
	SlotsPerArchivedPoint     uint64        // SlotsPerArchivedPoint defines the number of slots per one archived point.
	GenesisCountdownInterval  time.Duration // How often to log the countdown until the genesis time is reached.

	// Slasher constants.
	WeakSubjectivityPeriod    uint64 // WeakSubjectivityPeriod defines the time period expressed in number of epochs were proof of stake network should validate block headers and attestations for slashable events.
	PruneSlasherStoragePeriod uint64 // PruneSlasherStoragePeriod defines the time period expressed in number of epochs were proof of stake network should prune attestation and block header store.

	// Fork-related values.
	GenesisForkVersion  []byte            `yaml:"GENESIS_FORK_VERSION"` // GenesisForkVersion is used to track fork version between state transitions.
	NextForkVersion     []byte            `yaml:"NEXT_FORK_VERSION"`    // NextForkVersion is used to track the upcoming fork version, if any.
	NextForkEpoch       uint64            `yaml:"NEXT_FORK_EPOCH"`      // NextForkEpoch is used to track the epoch of the next fork, if any.
	ForkVersionSchedule map[uint64][]byte // Schedule of fork versions by epoch number.
}

var defaultBeaconConfig = &BeaconChainConfig{
	// Constants (Non-configurable)
	FarFutureEpoch:           1<<64 - 1,
	BaseRewardsPerEpoch:      4,
	DepositContractTreeDepth: 32,
	GenesisDelay:             172800, // 2 days

	// Misc constant.
	TargetCommitteeSize:            128,
	MaxValidatorsPerCommittee:      2048,
	MaxCommitteesPerSlot:           64,
	MinPerEpochChurnLimit:          4,
	ChurnLimitQuotient:             1 << 16,
	ShuffleRoundCount:              90,
	MinGenesisActiveValidatorCount: 16384,
	MinGenesisTime:                 0, // Zero until a proper time is decided.
	TargetAggregatorsPerCommittee:  16,
	HysteresisQuotient:             4,
	HysteresisDownwardMultiplier:   1,
	HysteresisUpwardMultiplier:     5,

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
	SecondsPerSlot:                   12,
	SlotsPerEpoch:                    32,
	MinSeedLookahead:                 1,
	MaxSeedLookahead:                 4,
	EpochsPerEth1VotingPeriod:        32,
	SlotsPerHistoricalRoot:           8192,
	MinValidatorWithdrawabilityDelay: 256,
	ShardCommitteePeriod:             256,
	MinEpochsToInactivityPenalty:     4,
	Eth1FollowDistance:               1024,
	SafeSlotsToUpdateJustified:       8,
	SecondsPerETH1Block:              14,

	// State list length constants.
	EpochsPerHistoricalVector: 65536,
	EpochsPerSlashingsVector:  8192,
	HistoricalRootsLimit:      16777216,
	ValidatorRegistryLimit:    1099511627776,

	// Reward and penalty quotients constants.
	BaseRewardFactor:            64,
	WhistleBlowerRewardQuotient: 512,
	ProposerRewardQuotient:      8,
	InactivityPenaltyQuotient:   1 << 24,
	MinSlashingPenaltyQuotient:  32,

	// Max operations per block constants.
	MaxProposerSlashings: 16,
	MaxAttesterSlashings: 2,
	MaxAttestations:      128,
	MaxDeposits:          16,
	MaxVoluntaryExits:    16,

	// BLS domain values.
	DomainBeaconProposer:    bytesutil.ToBytes4(bytesutil.Bytes4(0)),
	DomainBeaconAttester:    bytesutil.ToBytes4(bytesutil.Bytes4(1)),
	DomainRandao:            bytesutil.ToBytes4(bytesutil.Bytes4(2)),
	DomainDeposit:           bytesutil.ToBytes4(bytesutil.Bytes4(3)),
	DomainVoluntaryExit:     bytesutil.ToBytes4(bytesutil.Bytes4(4)),
	DomainSelectionProof:    bytesutil.ToBytes4(bytesutil.Bytes4(5)),
	DomainAggregateAndProof: bytesutil.ToBytes4(bytesutil.Bytes4(6)),

	// Prysm constants.
	GweiPerEth:                1000000000,
	BLSSecretKeyLength:        32,
	BLSPubkeyLength:           48,
	BLSSignatureLength:        96,
	DefaultBufferSize:         10000,
	WithdrawalPrivkeyFileName: "/shardwithdrawalkey",
	ValidatorPrivkeyFileName:  "/validatorprivatekey",
	RPCSyncCheck:              1,
	EmptySignature:            [96]byte{},
	DefaultPageSize:           250,
	MaxPeersToSync:            15,
	SlotsPerArchivedPoint:     2048,
	GenesisCountdownInterval:  time.Minute,

	// Slasher related values.
	WeakSubjectivityPeriod:    54000,
	PruneSlasherStoragePeriod: 10,

	// Fork related values.
	GenesisForkVersion:  []byte{0, 0, 0, 0},
	NextForkVersion:     []byte{0, 0, 0, 0}, // Set to GenesisForkVersion unless there is a scheduled fork
	NextForkEpoch:       1<<64 - 1,          // Set to FarFutureEpoch unless there is a scheduled fork.
	ForkVersionSchedule: map[uint64][]byte{
		// Any further forks must be specified here by their epoch number.
	},
}

var beaconConfig = defaultBeaconConfig

// BeaconConfig retrieves beacon chain config.
func BeaconConfig() *BeaconChainConfig {
	return beaconConfig
}

// MainnetConfig returns the default config to
// be used in the mainnet.
func MainnetConfig() *BeaconChainConfig {
	return defaultBeaconConfig
}

// MinimalSpecConfig retrieves the minimal config used in spec tests.
func MinimalSpecConfig() *BeaconChainConfig {
	minimalConfig := *defaultBeaconConfig
	// Misc
	minimalConfig.MaxCommitteesPerSlot = 4
	minimalConfig.TargetCommitteeSize = 4
	minimalConfig.MaxValidatorsPerCommittee = 2048
	minimalConfig.MinPerEpochChurnLimit = 4
	minimalConfig.ChurnLimitQuotient = 65536
	minimalConfig.ShuffleRoundCount = 10
	minimalConfig.MinGenesisActiveValidatorCount = 64
	minimalConfig.MinGenesisTime = 0
	minimalConfig.GenesisDelay = 300 // 5 minutes
	minimalConfig.TargetAggregatorsPerCommittee = 3

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
	minimalConfig.MaxSeedLookahead = 4
	minimalConfig.EpochsPerEth1VotingPeriod = 4
	minimalConfig.SlotsPerHistoricalRoot = 64
	minimalConfig.MinValidatorWithdrawabilityDelay = 256
	minimalConfig.ShardCommitteePeriod = 64
	minimalConfig.MinEpochsToInactivityPenalty = 4
	minimalConfig.Eth1FollowDistance = 16
	minimalConfig.SafeSlotsToUpdateJustified = 2
	minimalConfig.SecondsPerETH1Block = 14

	// State vector lengths
	minimalConfig.EpochsPerHistoricalVector = 64
	minimalConfig.EpochsPerSlashingsVector = 64
	minimalConfig.HistoricalRootsLimit = 16777216
	minimalConfig.ValidatorRegistryLimit = 1099511627776

	// Reward and penalty quotients
	minimalConfig.BaseRewardFactor = 64
	minimalConfig.WhistleBlowerRewardQuotient = 512
	minimalConfig.ProposerRewardQuotient = 8
	minimalConfig.InactivityPenaltyQuotient = 1 << 24
	minimalConfig.MinSlashingPenaltyQuotient = 32

	// Max operations per block
	minimalConfig.MaxProposerSlashings = 16
	minimalConfig.MaxAttesterSlashings = 2
	minimalConfig.MaxAttestations = 128
	minimalConfig.MaxDeposits = 16
	minimalConfig.MaxVoluntaryExits = 16

	// Signature domains
	minimalConfig.DomainBeaconProposer = bytesutil.ToBytes4(bytesutil.Bytes4(0))
	minimalConfig.DomainBeaconAttester = bytesutil.ToBytes4(bytesutil.Bytes4(1))
	minimalConfig.DomainRandao = bytesutil.ToBytes4(bytesutil.Bytes4(2))
	minimalConfig.DomainDeposit = bytesutil.ToBytes4(bytesutil.Bytes4(3))
	minimalConfig.DomainVoluntaryExit = bytesutil.ToBytes4(bytesutil.Bytes4(4))
	minimalConfig.GenesisForkVersion = []byte{0, 0, 0, 1}

	minimalConfig.DepositContractTreeDepth = 32
	minimalConfig.FarFutureEpoch = 1<<64 - 1
	return &minimalConfig
}

// E2ETestConfig retrieves the configurations made specifically for E2E testing.
// Warning: This config is only for testing, it is not meant for use outside of E2E.
func E2ETestConfig() *BeaconChainConfig {
	e2eConfig := MinimalSpecConfig()

	// Misc.
	e2eConfig.MinGenesisActiveValidatorCount = 256
	e2eConfig.GenesisDelay = 30 // 30 seconds so E2E has enough time to process deposits and get started.

	// Time parameters.
	e2eConfig.SecondsPerSlot = 8
	e2eConfig.SecondsPerETH1Block = 2
	e2eConfig.Eth1FollowDistance = 4
	e2eConfig.ShardCommitteePeriod = 4
	return e2eConfig
}

// AltonaConfig defines the config for the
// altona testnet.
func AltonaConfig() *BeaconChainConfig {
	altCfg := MainnetConfig()
	altCfg.MinGenesisActiveValidatorCount = 640
	altCfg.MinGenesisTime = 1593086400
	altCfg.GenesisForkVersion = []byte{0x00, 0x00, 0x01, 0x21}
	return altCfg
}

// UseMinimalConfig for beacon chain services.
func UseMinimalConfig() {
	beaconConfig = MinimalSpecConfig()
}

// UseE2EConfig for beacon chain services.
func UseE2EConfig() {
	beaconConfig = E2ETestConfig()
}

// UseAltonaConfig sets the main beacon chain
// config for altona.
func UseAltonaConfig() {
	beaconConfig = AltonaConfig()
}

// UseMainnetConfig for beacon chain services.
func UseMainnetConfig() {
	beaconConfig = MainnetConfig()
}

// OverrideBeaconConfig by replacing the config. The preferred pattern is to
// call BeaconConfig(), change the specific parameters, and then call
// OverrideBeaconConfig(c). Any subsequent calls to params.BeaconConfig() will
// return this new configuration.
func OverrideBeaconConfig(c *BeaconChainConfig) {
	beaconConfig = c
}

// SetupTestConfigCleanup preserves configurations allowing to modify them within tests without any
// restrictions, everything is restored after the test.
func SetupTestConfigCleanup(t *testing.T) {
	prevDefaultBeaconConfig := defaultBeaconConfig.Copy()
	prevBeaconConfig := beaconConfig.Copy()
	prevNetworkCfg := defaultNetworkConfig.Copy()
	t.Cleanup(func() {
		defaultBeaconConfig = prevDefaultBeaconConfig
		beaconConfig = prevBeaconConfig
		defaultNetworkConfig = prevNetworkCfg
	})
}

// Copy returns Copy of the config object.
func (c *BeaconChainConfig) Copy() *BeaconChainConfig {
	config, ok := deepcopy.Copy(*c).(BeaconChainConfig)
	if !ok {
		config = *defaultBeaconConfig
	}
	return &config
}
