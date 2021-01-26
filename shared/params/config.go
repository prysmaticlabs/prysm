// Package params defines important constants that are essential to eth2 services.
package params

import (
	"time"
)

// BeaconChainConfig contains constant configs for node to participate in beacon chain.
type BeaconChainConfig struct {
	// Constants (non-configurable)
	GenesisSlot              uint64 `yaml:"GENESIS_SLOT"`                // GenesisSlot represents the first canonical slot number of the beacon chain.
	GenesisEpoch             uint64 `yaml:"GENESIS_EPOCH"`               // GenesisEpoch represents the first canonical epoch number of the beacon chain.
	FarFutureEpoch           uint64 `yaml:"FAR_FUTURE_EPOCH"`            // FarFutureEpoch represents a epoch extremely far away in the future used as the default penalization slot for validators.
	BaseRewardsPerEpoch      uint64 `yaml:"BASE_REWARDS_PER_EPOCH"`      // BaseRewardsPerEpoch is used to calculate the per epoch rewards.
	DepositContractTreeDepth uint64 `yaml:"DEPOSIT_CONTRACT_TREE_DEPTH"` // DepositContractTreeDepth depth of the Merkle trie of deposits in the validator deposit contract on the PoW chain.
	JustificationBitsLength  uint64 `yaml:"JUSTIFICATION_BITS_LENGTH"`   // JustificationBitsLength defines number of epochs to track when implementing k-finality in Casper FFG.

	// Misc constants.
	TargetCommitteeSize            uint64 `yaml:"TARGET_COMMITTEE_SIZE" api:"target_committee_size"`                           // TargetCommitteeSize is the number of validators in a committee when the chain is healthy.
	MaxValidatorsPerCommittee      uint64 `yaml:"MAX_VALIDATORS_PER_COMMITTEE" api:"max_validators_per_committee"`             // MaxValidatorsPerCommittee defines the upper bound of the size of a committee.
	MaxCommitteesPerSlot           uint64 `yaml:"MAX_COMMITTEES_PER_SLOT" api:"max_committees_per_slot"`                       // MaxCommitteesPerSlot defines the max amount of committee in a single slot.
	MinPerEpochChurnLimit          uint64 `yaml:"MIN_PER_EPOCH_CHURN_LIMIT" api:"min_per_epoch_churn_limit"`                   // MinPerEpochChurnLimit is the minimum amount of churn allotted for validator rotations.
	ChurnLimitQuotient             uint64 `yaml:"CHURN_LIMIT_QUOTIENT" api:"churn_limit_quotient"`                             // ChurnLimitQuotient is used to determine the limit of how many validators can rotate per epoch.
	ShuffleRoundCount              uint64 `yaml:"SHUFFLE_ROUND_COUNT" api:"shuffle_round_count"`                               // ShuffleRoundCount is used for retrieving the permuted index.
	MinGenesisActiveValidatorCount uint64 `yaml:"MIN_GENESIS_ACTIVE_VALIDATOR_COUNT" api:"min_genesis_active_validator_count"` // MinGenesisActiveValidatorCount defines how many validator deposits needed to kick off beacon chain.
	MinGenesisTime                 uint64 `yaml:"MIN_GENESIS_TIME" api:"min_genesis_time"`                                     // MinGenesisTime is the time that needed to pass before kicking off beacon chain.
	TargetAggregatorsPerCommittee  uint64 `yaml:"TARGET_AGGREGATORS_PER_COMMITTEE" api:"target_aggregators_per_committee"`     // TargetAggregatorsPerCommittee defines the number of aggregators inside one committee.
	HysteresisQuotient             uint64 `yaml:"HYSTERESIS_QUOTIENT" api:"hysteresis_quotient"`                               // HysteresisQuotient defines the hysteresis quotient for effective balance calculations.
	HysteresisDownwardMultiplier   uint64 `yaml:"HYSTERESIS_DOWNWARD_MULTIPLIER" api:"hysteresis_downward_multiplier"`         // HysteresisDownwardMultiplier defines the hysteresis downward multiplier for effective balance calculations.
	HysteresisUpwardMultiplier     uint64 `yaml:"HYSTERESIS_UPWARD_MULTIPLIER" api:"hysteresis_upward_multiplier"`             // HysteresisUpwardMultiplier defines the hysteresis upward multiplier for effective balance calculations.

	// Gwei value constants.
	MinDepositAmount          uint64 `yaml:"MIN_DEPOSIT_AMOUNT" api:"min_deposit_amount"`                   // MinDepositAmount is the minimum amount of Gwei a validator can send to the deposit contract at once (lower amounts will be reverted).
	MaxEffectiveBalance       uint64 `yaml:"MAX_EFFECTIVE_BALANCE" api:"max_effective_balance"`             // MaxEffectiveBalance is the maximal amount of Gwei that is effective for staking.
	EjectionBalance           uint64 `yaml:"EJECTION_BALANCE" api:"ejection_balance"`                       // EjectionBalance is the minimal GWei a validator needs to have before ejected.
	EffectiveBalanceIncrement uint64 `yaml:"EFFECTIVE_BALANCE_INCREMENT" api:"effective_balance_increment"` // EffectiveBalanceIncrement is used for converting the high balance into the low balance for validators.

	// Initial value constants.
	BLSWithdrawalPrefixByte byte     `yaml:"BLS_WITHDRAWAL_PREFIX" api:"bls_withdrawal_prefix"` // BLSWithdrawalPrefixByte is used for BLS withdrawal and it's the first byte.
	ZeroHash                [32]byte // ZeroHash is used to represent a zeroed out 32 byte array.

	// Time parameters constants.
	GenesisDelay                     uint64 `yaml:"GENESIS_DELAY" api:"genesis_delay"`                                             // GenesisDelay is the minimum number of seconds to delay starting the ETH2 genesis. Must be at least 1 second.
	MinAttestationInclusionDelay     uint64 `yaml:"MIN_ATTESTATION_INCLUSION_DELAY" api:"min_attestation_inclusion_delay"`         // MinAttestationInclusionDelay defines how many slots validator has to wait to include attestation for beacon block.
	SecondsPerSlot                   uint64 `yaml:"SECONDS_PER_SLOT" api:"seconds_per_slot"`                                       // SecondsPerSlot is how many seconds are in a single slot.
	SlotsPerEpoch                    uint64 `yaml:"SLOTS_PER_EPOCH" api:"slots_per_epoch"`                                         // SlotsPerEpoch is the number of slots in an epoch.
	MinSeedLookahead                 uint64 `yaml:"MIN_SEED_LOOKAHEAD" api:"min_seed_lookahead"`                                   // MinSeedLookahead is the duration of randao look ahead seed.
	MaxSeedLookahead                 uint64 `yaml:"MAX_SEED_LOOKAHEAD" api:"max_seed_lookahead"`                                   // MaxSeedLookahead is the duration a validator has to wait for entry and exit in epoch.
	EpochsPerEth1VotingPeriod        uint64 `yaml:"EPOCHS_PER_ETH1_VOTING_PERIOD" api:"epochs_per_eth1_voting_period"`             // EpochsPerEth1VotingPeriod defines how often the merkle root of deposit receipts get updated in beacon node on per epoch basis.
	SlotsPerHistoricalRoot           uint64 `yaml:"SLOTS_PER_HISTORICAL_ROOT" api:"slots_per_historical_root"`                     // SlotsPerHistoricalRoot defines how often the historical root is saved.
	MinValidatorWithdrawabilityDelay uint64 `yaml:"MIN_VALIDATOR_WITHDRAWABILITY_DELAY" api:"min_validator_withdrawability_delay"` // MinValidatorWithdrawabilityDelay is the shortest amount of time a validator has to wait to withdraw.
	ShardCommitteePeriod             uint64 `yaml:"SHARD_COMMITTEE_PERIOD" api:"shard_committee_period"`                           // ShardCommitteePeriod is the minimum amount of epochs a validator must participate before exiting.
	MinEpochsToInactivityPenalty     uint64 `yaml:"MIN_EPOCHS_TO_INACTIVITY_PENALTY" api:"min_epochs_to_inactivity_penalty"`       // MinEpochsToInactivityPenalty defines the minimum amount of epochs since finality to begin penalizing inactivity.
	Eth1FollowDistance               uint64 `yaml:"ETH1_FOLLOW_DISTANCE" api:"eth1_follow_distance"`                               // Eth1FollowDistance is the number of eth1.0 blocks to wait before considering a new deposit for voting. This only applies after the chain as been started.
	SafeSlotsToUpdateJustified       uint64 `yaml:"SAFE_SLOTS_TO_UPDATE_JUSTIFIED" api:"safe_slots_to_update_justified"`           // SafeSlotsToUpdateJustified is the minimal slots needed to update justified check point.
	SecondsPerETH1Block              uint64 `yaml:"SECONDS_PER_ETH1_BLOCK" api:"seconds_per_eth1_block"`                           // SecondsPerETH1Block is the approximate time for a single eth1 block to be produced.

	// Ethereum PoW parameters.
	DepositChainID         uint64 `yaml:"DEPOSIT_CHAIN_ID" api:"deposit_chain_id"`                 // DepositChainID of the eth1 network. This used for replay protection.
	DepositNetworkID       uint64 `yaml:"DEPOSIT_NETWORK_ID" api:"deposit_network_id"`             // DepositNetworkID of the eth1 network. This used for replay protection.
	DepositContractAddress string `yaml:"DEPOSIT_CONTRACT_ADDRESS" api:"deposit_contract_address"` // DepositContractAddress is the address of the deposit contract.

	// Validator parameters.
	RandomSubnetsPerValidator         uint64 `yaml:"RANDOM_SUBNETS_PER_VALIDATOR" api:"random_subnets_per_validator"`                   // RandomSubnetsPerValidator specifies the amount of subnets a validator has to be subscribed to at one time.
	EpochsPerRandomSubnetSubscription uint64 `yaml:"EPOCHS_PER_RANDOM_SUBNET_SUBSCRIPTION" api:"epochs_per_random_subnet_subscription"` // EpochsPerRandomSubnetSubscription specifies the minimum duration a validator is connected to their subnet.

	// State list lengths
	EpochsPerHistoricalVector uint64 `yaml:"EPOCHS_PER_HISTORICAL_VECTOR" api:"epochs_per_historical_vector"` // EpochsPerHistoricalVector defines max length in epoch to store old historical stats in beacon state.
	EpochsPerSlashingsVector  uint64 `yaml:"EPOCHS_PER_SLASHINGS_VECTOR" api:"epochs_per_slashings_vector"`   // EpochsPerSlashingsVector defines max length in epoch to store old stats to recompute slashing witness.
	HistoricalRootsLimit      uint64 `yaml:"HISTORICAL_ROOTS_LIMIT" api:"historical_roots_limit"`             // HistoricalRootsLimit defines max historical roots that can be saved in state before roll over.
	ValidatorRegistryLimit    uint64 `yaml:"VALIDATOR_REGISTRY_LIMIT" api:"validator_registry_limit"`         // ValidatorRegistryLimit defines the upper bound of validators can participate in eth2.

	// Reward and penalty quotients constants.
	BaseRewardFactor               uint64 `yaml:"BASE_REWARD_FACTOR" api:"base_reward_factor"`                             // BaseRewardFactor is used to calculate validator per-slot interest rate.
	WhistleBlowerRewardQuotient    uint64 `yaml:"WHISTLEBLOWER_REWARD_QUOTIENT" api:"whistleblower_reward_quotient"`       // WhistleBlowerRewardQuotient is used to calculate whistle blower reward.
	ProposerRewardQuotient         uint64 `yaml:"PROPOSER_REWARD_QUOTIENT" api:"proposer_reward_quotient"`                 // ProposerRewardQuotient is used to calculate the reward for proposers.
	InactivityPenaltyQuotient      uint64 `yaml:"INACTIVITY_PENALTY_QUOTIENT" api:"inactivity_penalty_quotient"`           // InactivityPenaltyQuotient is used to calculate the penalty for a validator that is offline.
	MinSlashingPenaltyQuotient     uint64 `yaml:"MIN_SLASHING_PENALTY_QUOTIENT" api:"min_slashing_penalty_quotient"`       // MinSlashingPenaltyQuotient is used to calculate the minimum penalty to prevent DoS attacks.
	ProportionalSlashingMultiplier uint64 `yaml:"PROPORTIONAL_SLASHING_MULTIPLIER" api:"proportional_slashing_multiplier"` // ProportionalSlashingMultiplier is used as a multiplier on slashed penalties.

	// Max operations per block constants.
	MaxProposerSlashings uint64 `yaml:"MAX_PROPOSER_SLASHINGS" api:"max_proposer_slashings"` // MaxProposerSlashings defines the maximum number of slashings of proposers possible in a block.
	MaxAttesterSlashings uint64 `yaml:"MAX_ATTESTER_SLASHINGS" api:"max_attester_slashings"` // MaxAttesterSlashings defines the maximum number of casper FFG slashings possible in a block.
	MaxAttestations      uint64 `yaml:"MAX_ATTESTATIONS" api:"max_attestations"`             // MaxAttestations defines the maximum allowed attestations in a beacon block.
	MaxDeposits          uint64 `yaml:"MAX_DEPOSITS" api:"max_deposits"`                     // MaxDeposits defines the maximum number of validator deposits in a block.
	MaxVoluntaryExits    uint64 `yaml:"MAX_VOLUNTARY_EXITS" api:"max_voluntary_exits"`       // MaxVoluntaryExits defines the maximum number of validator exits in a block.

	// BLS domain values.
	DomainBeaconProposer    [4]byte `yaml:"DOMAIN_BEACON_PROPOSER" api:"domain_beacon_proposer"`         // DomainBeaconProposer defines the BLS signature domain for beacon proposal verification.
	DomainRandao            [4]byte `yaml:"DOMAIN_RANDAO" api:"domain_randao"`                           // DomainRandao defines the BLS signature domain for randao verification.
	DomainBeaconAttester    [4]byte `yaml:"DOMAIN_BEACON_ATTESTER" api:"domain_beacon_attester"`         // DomainBeaconAttester defines the BLS signature domain for attestation verification.
	DomainDeposit           [4]byte `yaml:"DOMAIN_DEPOSIT" api:"domain_deposit"`                         // DomainDeposit defines the BLS signature domain for deposit verification.
	DomainVoluntaryExit     [4]byte `yaml:"DOMAIN_VOLUNTARY_EXIT" api:"domain_voluntary_exit"`           // DomainVoluntaryExit defines the BLS signature domain for exit verification.
	DomainSelectionProof    [4]byte `yaml:"DOMAIN_SELECTION_PROOF" api:"domain_selection_proof"`         // DomainSelectionProof defines the BLS signature domain for selection proof.
	DomainAggregateAndProof [4]byte `yaml:"DOMAIN_AGGREGATE_AND_PROOF" api:"domain_aggregate_and_proof"` // DomainAggregateAndProof defines the BLS signature domain for aggregate and proof.

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
	NetworkName               string        `api:"config_name"` // NetworkName for allowing an easy human-readable way of knowing what chain is being used.
	BeaconStateFieldCount     int           // BeaconStateFieldCount defines how many fields are in beacon state.

	// Slasher constants.
	WeakSubjectivityPeriod    uint64 // WeakSubjectivityPeriod defines the time period expressed in number of epochs were proof of stake network should validate block headers and attestations for slashable events.
	PruneSlasherStoragePeriod uint64 // PruneSlasherStoragePeriod defines the time period expressed in number of epochs were proof of stake network should prune attestation and block header store.

	// Fork-related values.
	GenesisForkVersion  []byte            `yaml:"GENESIS_FORK_VERSION" api:"genesis_fork_version"` // GenesisForkVersion is used to track fork version between state transitions.
	NextForkVersion     []byte            `yaml:"NEXT_FORK_VERSION"`                               // NextForkVersion is used to track the upcoming fork version, if any.
	NextForkEpoch       uint64            `yaml:"NEXT_FORK_EPOCH"`                                 // NextForkEpoch is used to track the epoch of the next fork, if any.
	ForkVersionSchedule map[uint64][]byte // Schedule of fork versions by epoch number.

	// Weak subjectivity values.
	SafetyDecay uint64 // SafetyDecay is defined as the loss in the 1/3 consensus safety margin of the casper FFG mechanism.
}
