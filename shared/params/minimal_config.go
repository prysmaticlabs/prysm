package params

import "github.com/prysmaticlabs/prysm/shared/bytesutil"

// UseMinimalConfig for beacon chain services.
func UseMinimalConfig() {
	beaconConfig = MinimalSpecConfig()
}

// MinimalSpecConfig retrieves the minimal config used in spec tests.
func MinimalSpecConfig() *BeaconChainConfig {
	minimalConfig := mainnetBeaconConfig.Copy()
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

	return minimalConfig
}
