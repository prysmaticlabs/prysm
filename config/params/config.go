// Package params defines important constants that are essential to Prysm services.
package params

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

// BeaconChainConfig contains constant configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	ForkVersionNames                          map[[fieldparams.VersionLength]byte]string
	ForkVersionSchedule                       map[[fieldparams.VersionLength]byte]types.Epoch
	WithdrawalPrivkeyFileName                 string
	EthBurnAddressHex                         string
	TerminalTotalDifficulty                   string `yaml:"TERMINAL_TOTAL_DIFFICULTY" spec:"true"`
	ValidatorPrivkeyFileName                  string
	DepositContractAddress                    string      `yaml:"DEPOSIT_CONTRACT_ADDRESS" spec:"true"`
	PresetBase                                string      `yaml:"PRESET_BASE" spec:"true"`
	ConfigName                                string      `yaml:"CONFIG_NAME" spec:"true"`
	ShardingForkVersion                       []byte      `yaml:"SHARDING_FORK_VERSION" spec:"true"`
	AltairForkVersion                         []byte      `yaml:"ALTAIR_FORK_VERSION" spec:"true"`
	GenesisForkVersion                        []byte      `yaml:"GENESIS_FORK_VERSION" spec:"true"`
	BellatrixForkVersion                      []byte      `yaml:"BELLATRIX_FORK_VERSION" spec:"true"`
	CapellaForkVersion                        []byte      `yaml:"CAPELLA_FORK_VERSION" spec:"true"`
	ShardingForkEpoch                         types.Epoch `yaml:"SHARDING_FORK_EPOCH" spec:"true"`
	EpochsPerEth1VotingPeriod                 types.Epoch `yaml:"EPOCHS_PER_ETH1_VOTING_PERIOD" spec:"true"`
	MinGenesisTime                            uint64      `yaml:"MIN_GENESIS_TIME" spec:"true"`
	TargetAggregatorsPerCommittee             uint64      `yaml:"TARGET_AGGREGATORS_PER_COMMITTEE" spec:"true"`
	HysteresisQuotient                        uint64      `yaml:"HYSTERESIS_QUOTIENT" spec:"true"`
	HysteresisDownwardMultiplier              uint64      `yaml:"HYSTERESIS_DOWNWARD_MULTIPLIER" spec:"true"`
	HysteresisUpwardMultiplier                uint64      `yaml:"HYSTERESIS_UPWARD_MULTIPLIER" spec:"true"`
	MinDepositAmount                          uint64      `yaml:"MIN_DEPOSIT_AMOUNT" spec:"true"`
	MaxEffectiveBalance                       uint64      `yaml:"MAX_EFFECTIVE_BALANCE" spec:"true"`
	EjectionBalance                           uint64      `yaml:"EJECTION_BALANCE" spec:"true"`
	EffectiveBalanceIncrement                 uint64      `yaml:"EFFECTIVE_BALANCE_INCREMENT" spec:"true"`
	ExecutionEngineTimeoutValue               uint64
	MaxBuilderEpochMissedSlots                types.Slot
	MaxBuilderConsecutiveMissedSlots          types.Slot
	GenesisDelay                              uint64     `yaml:"GENESIS_DELAY" spec:"true"`
	MinAttestationInclusionDelay              types.Slot `yaml:"MIN_ATTESTATION_INCLUSION_DELAY" spec:"true"`
	SecondsPerSlot                            uint64     `yaml:"SECONDS_PER_SLOT" spec:"true"`
	SlotsPerEpoch                             types.Slot `yaml:"SLOTS_PER_EPOCH" spec:"true"`
	SqrRootSlotsPerEpoch                      types.Slot
	MinSeedLookahead                          types.Epoch `yaml:"MIN_SEED_LOOKAHEAD" spec:"true"`
	MaxSeedLookahead                          types.Epoch `yaml:"MAX_SEED_LOOKAHEAD" spec:"true"`
	DefaultBuilderGasLimit                    uint64
	SlotsPerHistoricalRoot                    types.Slot  `yaml:"SLOTS_PER_HISTORICAL_ROOT" spec:"true"`
	MinValidatorWithdrawabilityDelay          types.Epoch `yaml:"MIN_VALIDATOR_WITHDRAWABILITY_DELAY" spec:"true"`
	ShardCommitteePeriod                      types.Epoch `yaml:"SHARD_COMMITTEE_PERIOD" spec:"true"`
	MinEpochsToInactivityPenalty              types.Epoch `yaml:"MIN_EPOCHS_TO_INACTIVITY_PENALTY" spec:"true"`
	Eth1FollowDistance                        uint64      `yaml:"ETH1_FOLLOW_DISTANCE" spec:"true"`
	SafeSlotsToUpdateJustified                types.Slot  `yaml:"SAFE_SLOTS_TO_UPDATE_JUSTIFIED" spec:"true"`
	DeprecatedSafeSlotsToImportOptimistically types.Slot  `yaml:"SAFE_SLOTS_TO_IMPORT_OPTIMISTICALLY" spec:"true"`
	SecondsPerETH1Block                       uint64      `yaml:"SECONDS_PER_ETH1_BLOCK" spec:"true"`
	ProposerScoreBoost                        uint64      `yaml:"PROPOSER_SCORE_BOOST" spec:"true"`
	IntervalsPerSlot                          uint64      `yaml:"INTERVALS_PER_SLOT" spec:"true"`
	DepositChainID                            uint64      `yaml:"DEPOSIT_CHAIN_ID" spec:"true"`
	DepositNetworkID                          uint64      `yaml:"DEPOSIT_NETWORK_ID" spec:"true"`
	ShuffleRoundCount                         uint64      `yaml:"SHUFFLE_ROUND_COUNT" spec:"true"`
	RandomSubnetsPerValidator                 uint64      `yaml:"RANDOM_SUBNETS_PER_VALIDATOR" spec:"true"`
	EpochsPerRandomSubnetSubscription         uint64      `yaml:"EPOCHS_PER_RANDOM_SUBNET_SUBSCRIPTION" spec:"true"`
	EpochsPerHistoricalVector                 types.Epoch `yaml:"EPOCHS_PER_HISTORICAL_VECTOR" spec:"true"`
	EpochsPerSlashingsVector                  types.Epoch `yaml:"EPOCHS_PER_SLASHINGS_VECTOR" spec:"true"`
	HistoricalRootsLimit                      uint64      `yaml:"HISTORICAL_ROOTS_LIMIT" spec:"true"`
	ValidatorRegistryLimit                    uint64      `yaml:"VALIDATOR_REGISTRY_LIMIT" spec:"true"`
	BaseRewardFactor                          uint64      `yaml:"BASE_REWARD_FACTOR" spec:"true"`
	WhistleBlowerRewardQuotient               uint64      `yaml:"WHISTLEBLOWER_REWARD_QUOTIENT" spec:"true"`
	ProposerRewardQuotient                    uint64      `yaml:"PROPOSER_REWARD_QUOTIENT" spec:"true"`
	InactivityPenaltyQuotient                 uint64      `yaml:"INACTIVITY_PENALTY_QUOTIENT" spec:"true"`
	MinSlashingPenaltyQuotient                uint64      `yaml:"MIN_SLASHING_PENALTY_QUOTIENT" spec:"true"`
	ProportionalSlashingMultiplier            uint64      `yaml:"PROPORTIONAL_SLASHING_MULTIPLIER" spec:"true"`
	MaxProposerSlashings                      uint64      `yaml:"MAX_PROPOSER_SLASHINGS" spec:"true"`
	MaxAttesterSlashings                      uint64      `yaml:"MAX_ATTESTER_SLASHINGS" spec:"true"`
	MaxAttestations                           uint64      `yaml:"MAX_ATTESTATIONS" spec:"true"`
	MaxDeposits                               uint64      `yaml:"MAX_DEPOSITS" spec:"true"`
	MaxVoluntaryExits                         uint64      `yaml:"MAX_VOLUNTARY_EXITS" spec:"true"`
	GenesisEpoch                              types.Epoch `yaml:"GENESIS_EPOCH"`
	FarFutureEpoch                            types.Epoch `yaml:"FAR_FUTURE_EPOCH"`
	TerminalBlockHashActivationEpoch          types.Epoch `yaml:"TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH" spec:"true"`
	MinSyncCommitteeParticipants              uint64      `yaml:"MIN_SYNC_COMMITTEE_PARTICIPANTS" spec:"true"`
	InactivityPenaltyQuotientBellatrix        uint64      `yaml:"INACTIVITY_PENALTY_QUOTIENT_BELLATRIX" spec:"true"`
	ProportionalSlashingMultiplierBellatrix   uint64      `yaml:"PROPORTIONAL_SLASHING_MULTIPLIER_BELLATRIX" spec:"true"`
	MinSlashingPenaltyQuotientBellatrix       uint64      `yaml:"MIN_SLASHING_PENALTY_QUOTIENT_BELLATRIX" spec:"true"`
	ProportionalSlashingMultiplierAltair      uint64      `yaml:"PROPORTIONAL_SLASHING_MULTIPLIER_ALTAIR" spec:"true"`
	MinSlashingPenaltyQuotientAltair          uint64      `yaml:"MIN_SLASHING_PENALTY_QUOTIENT_ALTAIR" spec:"true"`
	InactivityPenaltyQuotientAltair           uint64      `yaml:"INACTIVITY_PENALTY_QUOTIENT_ALTAIR" spec:"true"`
	GenesisSlot                               types.Slot  `yaml:"GENESIS_SLOT"`
	EpochsPerSyncCommitteePeriod              types.Epoch `yaml:"EPOCHS_PER_SYNC_COMMITTEE_PERIOD" spec:"true"`
	MinGenesisActiveValidatorCount            uint64      `yaml:"MIN_GENESIS_ACTIVE_VALIDATOR_COUNT" spec:"true"`
	GweiPerEth                                uint64
	BLSSecretKeyLength                        int
	BLSPubkeyLength                           int
	DefaultBufferSize                         int
	ChurnLimitQuotient                        uint64 `yaml:"CHURN_LIMIT_QUOTIENT" spec:"true"`
	MinPerEpochChurnLimit                     uint64 `yaml:"MIN_PER_EPOCH_CHURN_LIMIT" spec:"true"`
	RPCSyncCheck                              time.Duration
	InactivityScoreRecoveryRate               uint64 `yaml:"INACTIVITY_SCORE_RECOVERY_RATE" spec:"true"`
	DefaultPageSize                           int
	MaxPeersToSync                            int
	SlotsPerArchivedPoint                     types.Slot
	GenesisCountdownInterval                  time.Duration
	BeaconStateFieldCount                     int
	BeaconStateAltairFieldCount               int
	BeaconStateBellatrixFieldCount            int
	BeaconStateCapellaFieldCount              int
	WeakSubjectivityPeriod                    types.Epoch
	PruneSlasherStoragePeriod                 types.Epoch
	SlashingProtectionPruningEpochs           types.Epoch
	MaxCommitteesPerSlot                      uint64      `yaml:"MAX_COMMITTEES_PER_SLOT" spec:"true"`
	MaxValidatorsPerCommittee                 uint64      `yaml:"MAX_VALIDATORS_PER_COMMITTEE" spec:"true"`
	AltairForkEpoch                           types.Epoch `yaml:"ALTAIR_FORK_EPOCH" spec:"true"`
	TargetCommitteeSize                       uint64      `yaml:"TARGET_COMMITTEE_SIZE" spec:"true"`
	BellatrixForkEpoch                        types.Epoch `yaml:"BELLATRIX_FORK_EPOCH" spec:"true"`
	JustificationBitsLength                   uint64      `yaml:"JUSTIFICATION_BITS_LENGTH"`
	InactivityScoreBias                       uint64      `yaml:"INACTIVITY_SCORE_BIAS" spec:"true"`
	DepositContractTreeDepth                  uint64      `yaml:"DEPOSIT_CONTRACT_TREE_DEPTH"`
	CapellaForkEpoch                          types.Epoch `yaml:"CAPELLA_FORK_EPOCH" spec:"true"`
	BaseRewardsPerEpoch                       uint64      `yaml:"BASE_REWARDS_PER_EPOCH"`
	FarFutureSlot                             types.Slot  `yaml:"FAR_FUTURE_SLOT"`
	SafetyDecay                               uint64
	SyncCommitteeSize                         uint64 `yaml:"SYNC_COMMITTEE_SIZE" spec:"true"`
	SyncCommitteeSubnetCount                  uint64 `yaml:"SYNC_COMMITTEE_SUBNET_COUNT" spec:"true"`
	TargetAggregatorsPerSyncSubcommittee      uint64 `yaml:"TARGET_AGGREGATORS_PER_SYNC_SUBCOMMITTEE" spec:"true"`
	TimelySourceWeight                        uint64 `yaml:"TIMELY_SOURCE_WEIGHT" spec:"true"`
	TimelyTargetWeight                        uint64 `yaml:"TIMELY_TARGET_WEIGHT" spec:"true"`
	TimelyHeadWeight                          uint64 `yaml:"TIMELY_HEAD_WEIGHT" spec:"true"`
	SyncRewardWeight                          uint64 `yaml:"SYNC_REWARD_WEIGHT" spec:"true"`
	WeightDenominator                         uint64 `yaml:"WEIGHT_DENOMINATOR" spec:"true"`
	ProposerWeight                            uint64 `yaml:"PROPOSER_WEIGHT" spec:"true"`
	EmptySignature                            [96]byte
	ZeroHash                                  [32]byte
	TerminalBlockHash                         common.Hash `yaml:"TERMINAL_BLOCK_HASH" spec:"true"`
	DefaultFeeRecipient                       common.Address
	DomainRandao                              [4]byte `yaml:"DOMAIN_RANDAO" spec:"true"`
	DomainBeaconProposer                      [4]byte `yaml:"DOMAIN_BEACON_PROPOSER" spec:"true"`
	DomainContributionAndProof                [4]byte `yaml:"DOMAIN_CONTRIBUTION_AND_PROOF" spec:"true"`
	DomainSyncCommitteeSelectionProof         [4]byte `yaml:"DOMAIN_SYNC_COMMITTEE_SELECTION_PROOF" spec:"true"`
	DomainApplicationBuilder                  [4]byte
	DomainSelectionProof                      [4]byte `yaml:"DOMAIN_SELECTION_PROOF" spec:"true"`
	DomainAggregateAndProof                   [4]byte `yaml:"DOMAIN_AGGREGATE_AND_PROOF" spec:"true"`
	DomainSyncCommittee                       [4]byte `yaml:"DOMAIN_SYNC_COMMITTEE" spec:"true"`
	DomainDeposit                             [4]byte `yaml:"DOMAIN_DEPOSIT" spec:"true"`
	DomainApplicationMask                     [4]byte `yaml:"DOMAIN_APPLICATION_MASK" spec:"true"`
	DomainBeaconAttester                      [4]byte `yaml:"DOMAIN_BEACON_ATTESTER" spec:"true"`
	DomainBLSToExecutionChange                [4]byte
	DomainVoluntaryExit                       [4]byte `yaml:"DOMAIN_VOLUNTARY_EXIT" spec:"true"`
	TimelySourceFlagIndex                     uint8   `yaml:"TIMELY_SOURCE_FLAG_INDEX" spec:"true"`
	TimelyHeadFlagIndex                       uint8   `yaml:"TIMELY_HEAD_FLAG_INDEX" spec:"true"`
	BLSWithdrawalPrefixByte                   byte    `yaml:"BLS_WITHDRAWAL_PREFIX" spec:"true"`
	ETH1AddressWithdrawalPrefixByte           byte    `yaml:"ETH1_ADDRESS_WITHDRAWAL_PREFIX" spec:"true"`
	TimelyTargetFlagIndex                     uint8   `yaml:"TIMELY_TARGET_FLAG_INDEX" spec:"true"`
}

// InitializeForkSchedule initializes the schedules forks baked into the config.
func (b *BeaconChainConfig) InitializeForkSchedule() {
	// Reset Fork Version Schedule.
	b.ForkVersionSchedule = configForkSchedule(b)
	b.ForkVersionNames = configForkNames(b)
}

func configForkSchedule(b *BeaconChainConfig) map[[fieldparams.VersionLength]byte]types.Epoch {
	fvs := map[[fieldparams.VersionLength]byte]types.Epoch{}
	// Set Genesis fork data.
	fvs[bytesutil.ToBytes4(b.GenesisForkVersion)] = b.GenesisEpoch
	// Set Altair fork data.
	fvs[bytesutil.ToBytes4(b.AltairForkVersion)] = b.AltairForkEpoch
	// Set Bellatrix fork data.
	fvs[bytesutil.ToBytes4(b.BellatrixForkVersion)] = b.BellatrixForkEpoch
	return fvs
}

func configForkNames(b *BeaconChainConfig) map[[fieldparams.VersionLength]byte]string {
	fvn := map[[fieldparams.VersionLength]byte]string{}
	// Set Genesis fork data.
	fvn[bytesutil.ToBytes4(b.GenesisForkVersion)] = "phase0"
	// Set Altair fork data.
	fvn[bytesutil.ToBytes4(b.AltairForkVersion)] = "altair"
	// Set Bellatrix fork data.
	fvn[bytesutil.ToBytes4(b.BellatrixForkVersion)] = "bellatrix"
	return fvn
}
