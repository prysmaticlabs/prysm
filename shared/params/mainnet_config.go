package params

import (
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// MainnetConfig returns the configuration to be used in the main network.
func MainnetConfig() *BeaconChainConfig {
	return mainnetBeaconConfig
}

// UseMainnetConfig for beacon chain services.
func UseMainnetConfig() {
	beaconConfig = MainnetConfig()
}

var mainnetNetworkConfig = &NetworkConfig{
	GossipMaxSize:                   1 << 20, // 1 MiB
	MaxChunkSize:                    1 << 20, // 1 MiB
	AttestationSubnetCount:          64,
	AttestationPropagationSlotRange: 32,
	MaxRequestBlocks:                1 << 10, // 1024
	TtfbTimeout:                     5 * time.Second,
	RespTimeout:                     10 * time.Second,
	MaximumGossipClockDisparity:     500 * time.Millisecond,
	MessageDomainInvalidSnappy:      [4]byte{00, 00, 00, 00},
	MessageDomainValidSnappy:        [4]byte{01, 00, 00, 00},
	ETH2Key:                         "eth2",
	AttSubnetKey:                    "attnets",
	MinimumPeersInSubnet:            4,
	MinimumPeersInSubnetSearch:      20,
	ContractDeploymentBlock:         11184524, // Note: contract was deployed in block 11052984 but no transactions were sent until 11184524.
	BootstrapNodes: []string{
		"enr:-KG4QAot7X6xLDriqiOpDx9SsoGyDa_IOEaMyibfGxSnSwGNSRU49YJDkluFuw-bqO3EpYiLCBJkfp5JTEcoh6XA6UQDhGV0aDKQ6PqIZwAABwL__________4JpZIJ2NIJpcIQS3eH5iXNlY3AyNTZrMaEDzOz7QvfIbJC2iYzO1E0H_EkoA7Rr3cXK88CHRFl2bTODdGNwgiMog3VkcIIjKA",
		"enr:-KG4QHfAwMzERDnjq59K5YPzWpRXrkCKKF-vwcGbE-Y5-WbVVbjzjYGP1mFxmcwpLmqPZLmlZOJNkZeLA3Le9dImCocDhGV0aDKQ6PqIZwAABwL__________4JpZIJ2NIJpcIQSv2NjiXNlY3AyNTZrMaECEuQJA4lxZX7zoqTXq5Lj2LhqzY865AHIn-apmr_Q4oyDdGNwgiMog3VkcIIjKA",
		"enr:-KG4QFddZVEYZl40l7lfrDLwWD2zxyvdvscVwh68dNYSJMzOTqdlPFvsyymK5NE-e3dEiTwPENAyY-FpyoYRqnPoqu0DhGV0aDKQ6PqIZwAABwL__________4JpZIJ2NIJpcIQNOxpTiXNlY3AyNTZrMaECBRIx5tIZXH6Zcyb4uIf7nJx_o1jDHoxR5UIxBZBokS2DdGNwgiMog3VkcIIjKA",
		"enr:-LK4QM7m9MQqU7N2cDt2TqxrY6sDG1jnm4hk0MigeQGcUxTURvEZc1gk2nveZiZfBN7mumqIwQ9lWdSwegaEbnUNphoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDo-ohnAAAHAv__________gmlkgnY0gmlwhBLex-SJc2VjcDI1NmsxoQPHMYyjD5XaR3797lG_Wj-JoqPcFOa3dwHDlAbIx6loBYN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QHso1e97FeoDFl0FcSGaQNlq6REBUxxt7vwTvUKC8WhjMVUYDLQ44lhAwVt5wUKRCKeX63KF_eQSbbYKCzluaWEBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDo-ohnAAAHAv__________gmlkgnY0gmlwhBLYe_qJc2VjcDI1NmsxoQPkN-M3x7SKIqlebY40swTVbBT3-XILub8PI5sPIz_LBYN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QMl5I25EREe2En8EKM-9lLLm9WcjHu5znHAHga8uCQv9M67nZ9qusrypHsdTfu7MMKzmJYS4XKYcZFTYiw4AOicBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDo-ohnAAAHAv__________gmlkgnY0gmlwhAOJvQyJc2VjcDI1NmsxoQMKcofI7xFJkHKBvZn34PonEtwLl6_Wf4n6MlMcYzzV04N0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QO47q-FNnfDcQk88xW3gOXhqN1qomLSd5OgUNJ1pSg4eJXEA65Eq99K6KaUHQ6lxfY1adMGWeN0An7FXNuKlaHcBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDo-ohnAAAHAv__________gmlkgnY0gmlwhAOFWluJc2VjcDI1NmsxoQIWLK0r-BMnD1zGl3cSI1lO7ph07NZGJuNSgxLUlB336oN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QI1brZ3_T28UxYaU9k-2B324nb7_IfEsT_3pjS_Kjx5eYZ96JM-YujYO2UX28TKzvhn3awrREs_aGBCKFGdddHcBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDo-ohnAAAHAv__________gmlkgnY0gmlwhBLcIVCJc2VjcDI1NmsxoQOX5hzJZXAOnZ2JmcMy25uOcWz713bcf01HRHopA8EwnoN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QNstPLesGCGI4g4t-79M6b8C6Fdxms4vmKtWietRgfYmP2Hkk_z_mHcV4eQbkqB6P7rh1QI_lJ4siWKBmQWk7wcBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDo-ohnAAAHAv__________gmlkgnY0gmlwhBLg_OOJc2VjcDI1NmsxoQJhwfIDOttCK4baG6D_LvT8oANUmEkbGPMx7ra0hmEHeoN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QOH7_twj22lYG7Ua3V_2bO1FKu3PUtH11VFVKvNBsN6WE2KCPNA4ZeznCUtaV9t1ifNE2dv4j2cLI1io5E9WihkBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDo-ohnAAAHAv__________gmlkgnY0gmlwhAONxf-Jc2VjcDI1NmsxoQKYfHmHELRhqvZyuMr-K6v-Ii7C3ngmoE72UpVM6P8gpIN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QPLOWBnsm00pHaNa5cCfmbO9afg0WZ1r4jZlcwtcQWoQTOKwIumDEEUnCgrJOpd5h61qgvvJI0-YccAaE8zhyq0Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDo-ohnAAAHAv__________gmlkgnY0gmlwhBJ1CQ6Jc2VjcDI1NmsxoQJO4yHzl09NWLY-5_el7stRQyWy7ZA2kISbZE93ITqyToN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QO_bDdCQDgCZPgbSiGOY_6tMUZszQI2xUNehnWC9bPmsEJNhgUsQ33pHPzSKbfh9czhTP4NkhcQ2qSBBFKKavHEBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDo-ohnAAAHAv__________gmlkgnY0gmlwhBLbRUiJc2VjcDI1NmsxoQJPx5SiPDV6VX0lLUpB-ikjZ7oEZUIImDnCmc1Z7atU5oN0Y3CCIyiDdWRwgiMo",
		"enr:-Ku4QEDWyEs1Va0YSxSHMbcSaFuasQ2F-kGZ_Mn-doSxbuvpBCrKmTwmvHJK2jcELj7vUgKKj3ORACvqTwtH1y8F9skBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpB5-A5dAAAHAf__________gmlkgnY0gmlwhAOAq0KJc2VjcDI1NmsxoQI-C81GsaRAfP8-HH3FM_rcBPTBfzSFIWCkNCkKj9TmTIN1ZHCCIyg",
	},
}

var mainnetBeaconConfig = &BeaconChainConfig{
	// Constants (Non-configurable)
	FarFutureEpoch:           1<<64 - 1,
	FarFutureSlot:            1<<64 - 1,
	BaseRewardsPerEpoch:      4,
	DepositContractTreeDepth: 32,
	GenesisDelay:             604800, // 1 week.

	// Misc constant.
	TargetCommitteeSize:            128,
	MaxValidatorsPerCommittee:      2048,
	MaxCommitteesPerSlot:           64,
	MinPerEpochChurnLimit:          4,
	ChurnLimitQuotient:             1 << 16,
	ShuffleRoundCount:              90,
	MinGenesisActiveValidatorCount: 16384,
	MinGenesisTime:                 1606824000, // Dec 1, 2020, 12pm UTC.
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
	EpochsPerEth1VotingPeriod:        64,
	SlotsPerHistoricalRoot:           8192,
	MinValidatorWithdrawabilityDelay: 256,
	ShardCommitteePeriod:             256,
	MinEpochsToInactivityPenalty:     4,
	Eth1FollowDistance:               2048,
	SafeSlotsToUpdateJustified:       8,

	// Ethereum PoW parameters.
	DepositChainID:         1, // Chain ID of eth1 mainnet.
	DepositNetworkID:       1, // Network ID of eth1 mainnet.
	DepositContractAddress: "0x00000000219ab540356cBB839Cbe05303d7705Fa",

	// Validator params.
	RandomSubnetsPerValidator:         1 << 0,
	EpochsPerRandomSubnetSubscription: 1 << 8,

	// While eth1 mainnet block times are closer to 13s, we must conform with other clients in
	// order to vote on the correct eth1 blocks.
	//
	// Additional context: https://github.com/ethereum/eth2.0-specs/issues/2132
	// Bug prompting this change: https://github.com/prysmaticlabs/prysm/issues/7856
	// Future optimization: https://github.com/prysmaticlabs/prysm/issues/7739
	SecondsPerETH1Block: 14,

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
	MaxProposerSlashings:         16,
	MaxAttesterSlashings:         2,
	MaxAttestations:              128,
	MaxDeposits:                  16,
	MaxVoluntaryExits:            16,
	MaxExecutionTransactions:     16384,
	MaxBytesPerOpaqueTransaction: 1048576,

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
	BeaconStateFieldCount:     22,

	// Slasher related values.
	WeakSubjectivityPeriod:          54000,
	PruneSlasherStoragePeriod:       10,
	SlashingProtectionPruningEpochs: 512,

	// Weak subjectivity values.
	SafetyDecay: 10,

	// Fork related values.
	GenesisForkVersion:  []byte{0, 0, 0, 0},
	NextForkVersion:     []byte{0, 0, 0, 0}, // Set to GenesisForkVersion unless there is a scheduled fork
	NextForkEpoch:       1<<64 - 1,          // Set to FarFutureEpoch unless there is a scheduled fork.
	ForkVersionSchedule: map[types.Epoch][]byte{
		// Any further forks must be specified here by their epoch number.
	},
}
