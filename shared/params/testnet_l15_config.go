package params

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"time"
)

// UseL15NetworkConfig uses the Lukso specific
// network config.
func UseL15NetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 0
	cfg.BootstrapNodes = []string{
		"enr:-Ku4QEL0I7H3EawRwc2ZUevmj-_T0R6JZGMhfp_2KHBlwAt5bwA19c8LSYZzy63EvpsYbifKye6qnE-_vsNimWOz8scBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpCvIkw2g6VTF___________gmlkgnY0gmlwhCPqeliJc2VjcDI1NmsxoQLt36VpP56n0SlTYWcSBwL7aGK_AFwNLGxOGQt91nchMYN1ZHCCEuk",
		"enr:-Ku4QAmYtwrQBZ-WJwTPL4xMpTO6BlZcU6IuXljtd_SgC51nGRs98WvxCX0-ZJBs0G9m9tcFPsktbdSr7EliMhrZnfEBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpCvIkw2g6VTF___________gmlkgnY0gmlwhCKNcPOJc2VjcDI1NmsxoQLt36VpP56n0SlTYWcSBwL7aGK_AFwNLGxOGQt91nchMYN1ZHCCEuk",
		"enr:-Ku4QEXRrSXB7od-xNeoLuq6GicTHpuuCNRPPR9tM48Iai0-FoHL4JsntmpnwUrC-di-lT6gkbxV7Jikg9s6ImsAT1oBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpCvIkw2g6VTF___________gmlkgnY0gmlwhCPGqi6Jc2VjcDI1NmsxoQLt36VpP56n0SlTYWcSBwL7aGK_AFwNLGxOGQt91nchMYN1ZHCCEuk",
		"enr:-Ku4QBuS5wqvF6SHaPpuu4r4ZlRRVC1Ojp1zDOAVC1X0PB3gRujAhWZdk2m0kn3FwoPuHft_Sku0tWHSfBVlHoER160Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpCvIkw2g6VTF___________gmlkgnY0gmlwhCKNLsqJc2VjcDI1NmsxoQLt36VpP56n0SlTYWcSBwL7aGK_AFwNLGxOGQt91nchMYN1ZHCCEuk",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// UseL15Config sets the main beacon chain
// config for Lukso.
func UseL15Config() {
	beaconConfig = L15Config()
}

// L15Config defines the config for the
// Lukso testnet.
func L15Config() *BeaconChainConfig {
	cfg := &BeaconChainConfig{
		// Constants (Non-configurable)
		FarFutureEpoch:           1<<64 - 1,
		FarFutureSlot:            1<<64 - 1,
		BaseRewardsPerEpoch:      4,
		DepositContractTreeDepth: 32,
		GenesisDelay:             0,

		// Misc constant.
		TargetCommitteeSize:            128,
		MaxValidatorsPerCommittee:      2048,
		MaxCommitteesPerSlot:           64,
		MinPerEpochChurnLimit:          4,
		ChurnLimitQuotient:             1 << 16,
		ShuffleRoundCount:              90,
		MinGenesisActiveValidatorCount: 16384,
		MinGenesisTime:                 1627579138,
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
		SecondsPerSlot:                   6,
		SlotsPerEpoch:                    32,
		MinSeedLookahead:                 2,
		MaxSeedLookahead:                 4,
		EpochsPerEth1VotingPeriod:        2,
		SlotsPerHistoricalRoot:           8192,
		MinValidatorWithdrawabilityDelay: 256,
		ShardCommitteePeriod:             256,
		MinEpochsToInactivityPenalty:     4,
		Eth1FollowDistance:               2048,
		SafeSlotsToUpdateJustified:       8,

		// Ethereum PoW parameters.
		DepositChainID:         808081, // Chain ID of eth1 mainnet.
		DepositNetworkID:       808081, // Network ID of eth1 mainnet.
		DepositContractAddress: "0x000000000000000000000000000000000000cafe",

		// Validator params.
		RandomSubnetsPerValidator:         1 << 0,
		EpochsPerRandomSubnetSubscription: 1 << 8,

		// While eth1 mainnet block times are closer to 13s, we must conform with other clients in
		// order to vote on the correct eth1 blocks.
		//
		// Additional context: https://github.com/ethereum/eth2.0-specs/issues/2132
		// Bug prompting this change: https://github.com/prysmaticlabs/prysm/issues/7856
		// Future optimization: https://github.com/prysmaticlabs/prysm/issues/7739
		SecondsPerETH1Block: 6,

		// State list length constants.
		EpochsPerHistoricalVector: 65536,
		EpochsPerSlashingsVector:  8192,
		HistoricalRootsLimit:      16777216,
		ValidatorRegistryLimit:    1099511627776,

		// Reward and penalty quotients constants.
		BaseRewardFactor:               64,
		WhistleBlowerRewardQuotient:    512,
		ProposerRewardQuotient:         8,
		InactivityPenaltyQuotient:      67108864,
		MinSlashingPenaltyQuotient:     128,
		ProportionalSlashingMultiplier: 1,

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
		ConfigName:                ConfigNames[Mainnet],
		BeaconStateFieldCount:     21,

		// Slasher related values.
		WeakSubjectivityPeriod:          54000,
		PruneSlasherStoragePeriod:       10,
		SlashingProtectionPruningEpochs: 512,

		// Weak subjectivity values.
		SafetyDecay: 10,

		// Fork related values.
		GenesisForkVersion: []byte{0x83, 0xa5, 0x53, 0x17},
		NextForkVersion:    []byte{0, 0, 0, 0}, // Set to GenesisForkVersion unless there is a scheduled fork
		NextForkEpoch:      1<<64 - 1,          // Set to FarFutureEpoch unless there is a scheduled fork.
		ForkVersionSchedule: map[types.Epoch][]byte{
			// Any further forks must be specified here by their epoch number.
		},
	}
	return cfg
}
