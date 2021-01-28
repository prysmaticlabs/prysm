package beaconv1

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetSpec(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()

	config.ConfigName = "ConfigName"
	config.MaxCommitteesPerSlot = 1
	config.TargetCommitteeSize = 2
	config.MaxValidatorsPerCommittee = 3
	config.MinPerEpochChurnLimit = 4
	config.ChurnLimitQuotient = 5
	config.ShuffleRoundCount = 6
	config.MinGenesisActiveValidatorCount = 7
	config.MinGenesisTime = 8
	config.HysteresisQuotient = 9
	config.HysteresisDownwardMultiplier = 10
	config.HysteresisUpwardMultiplier = 11
	config.SafeSlotsToUpdateJustified = 12
	config.Eth1FollowDistance = 13
	config.TargetAggregatorsPerCommittee = 14
	config.RandomSubnetsPerValidator = 15
	config.EpochsPerRandomSubnetSubscription = 16
	config.SecondsPerETH1Block = 17
	config.DepositChainID = 18
	config.DepositNetworkID = 19
	config.DepositContractAddress = "DepositContractAddress"
	config.MinDepositAmount = 20
	config.MaxEffectiveBalance = 21
	config.EjectionBalance = 22
	config.EffectiveBalanceIncrement = 23
	config.GenesisForkVersion = []byte("GenesisForkVersion")
	config.BLSWithdrawalPrefixByte = byte('b')
	config.GenesisDelay = 24
	config.SecondsPerSlot = 25
	config.MinAttestationInclusionDelay = 26
	config.SlotsPerEpoch = 27
	config.MinSeedLookahead = 28
	config.MaxSeedLookahead = 29
	config.EpochsPerEth1VotingPeriod = 30
	config.SlotsPerHistoricalRoot = 31
	config.MinValidatorWithdrawabilityDelay = 32
	config.ShardCommitteePeriod = 33
	config.MinEpochsToInactivityPenalty = 34
	config.EpochsPerHistoricalVector = 35
	config.EpochsPerSlashingsVector = 36
	config.HistoricalRootsLimit = 37
	config.ValidatorRegistryLimit = 38
	config.BaseRewardFactor = 39
	config.WhistleBlowerRewardQuotient = 40
	config.ProposerRewardQuotient = 41
	config.InactivityPenaltyQuotient = 42
	config.MinSlashingPenaltyQuotient = 43
	config.ProportionalSlashingMultiplier = 44
	config.MaxProposerSlashings = 45
	config.MaxAttesterSlashings = 46
	config.MaxAttestations = 47
	config.MaxDeposits = 48
	config.MaxVoluntaryExits = 49

	var dbp [4]byte
	copy(dbp[:], []byte{'0', '0', '0', '1'})
	config.DomainBeaconProposer = dbp
	var dba [4]byte
	copy(dba[:], []byte{'0', '0', '0', '2'})
	config.DomainBeaconAttester = dba
	var dr [4]byte
	copy(dr[:], []byte{'0', '0', '0', '3'})
	config.DomainRandao = dr
	var dd [4]byte
	copy(dd[:], []byte{'0', '0', '0', '4'})
	config.DomainDeposit = dd
	var dve [4]byte
	copy(dve[:], []byte{'0', '0', '0', '5'})
	config.DomainVoluntaryExit = dve
	var dsp [4]byte
	copy(dsp[:], []byte{'0', '0', '0', '6'})
	config.DomainSelectionProof = dsp
	var daap [4]byte
	copy(daap[:], []byte{'0', '0', '0', '7'})
	config.DomainAggregateAndProof = daap

	params.OverrideBeaconConfig(config)

	server := &Server{}
	resp, err := server.GetSpec(context.Background(), &types.Empty{})
	require.NoError(t, err)

	assert.Equal(t, 60, len(resp.Data))
	for k, v := range resp.Data {
		switch k {
		case "config_name":
			assert.Equal(t, "ConfigName", v)
		case "max_committees_per_slot":
			assert.Equal(t, "1", v)
		case "target_committee_size":
			assert.Equal(t, "2", v)
		case "max_validators_per_committee":
			assert.Equal(t, "3", v)
		case "min_per_epoch_churn_limit":
			assert.Equal(t, "4", v)
		case "churn_limit_quotient":
			assert.Equal(t, "5", v)
		case "shuffle_round_count":
			assert.Equal(t, "6", v)
		case "min_genesis_active_validator_count":
			assert.Equal(t, "7", v)
		case "min_genesis_time":
			assert.Equal(t, "8", v)
		case "hysteresis_quotient":
			assert.Equal(t, "9", v)
		case "hysteresis_downward_multiplier":
			assert.Equal(t, "10", v)
		case "hysteresis_upward_multiplier":
			assert.Equal(t, "11", v)
		case "safe_slots_to_update_justified":
			assert.Equal(t, "12", v)
		case "eth1_follow_distance":
			assert.Equal(t, "13", v)
		case "target_aggregators_per_committee":
			assert.Equal(t, "14", v)
		case "random_subnets_per_validator":
			assert.Equal(t, "15", v)
		case "epochs_per_random_subnet_subscription":
			assert.Equal(t, "16", v)
		case "seconds_per_eth1_block":
			assert.Equal(t, "17", v)
		case "deposit_chain_id":
			assert.Equal(t, "18", v)
		case "deposit_network_id":
			assert.Equal(t, "19", v)
		case "deposit_contract_address":
			assert.Equal(t, "DepositContractAddress", v)
		case "min_deposit_amount":
			assert.Equal(t, "20", v)
		case "max_effective_balance":
			assert.Equal(t, "21", v)
		case "ejection_balance":
			assert.Equal(t, "22", v)
		case "effective_balance_increment":
			assert.Equal(t, "23", v)
		case "genesis_fork_version":
			assert.Equal(t, "0x47656e65736973466f726b56657273696f6e", v)
		case "bls_withdrawal_prefix":
			assert.Equal(t, "0x62", v)
		case "genesis_delay":
			assert.Equal(t, "24", v)
		case "seconds_per_slot":
			assert.Equal(t, "25", v)
		case "min_attestation_inclusion_delay":
			assert.Equal(t, "26", v)
		case "slots_per_epoch":
			assert.Equal(t, "27", v)
		case "min_seed_lookahead":
			assert.Equal(t, "28", v)
		case "max_seed_lookahead":
			assert.Equal(t, "29", v)
		case "epochs_per_eth1_voting_period":
			assert.Equal(t, "30", v)
		case "slots_per_historical_root":
			assert.Equal(t, "31", v)
		case "min_validator_withdrawability_delay":
			assert.Equal(t, "32", v)
		case "shard_committee_period":
			assert.Equal(t, "33", v)
		case "min_epochs_to_inactivity_penalty":
			assert.Equal(t, "34", v)
		case "epochs_per_historical_vector":
			assert.Equal(t, "35", v)
		case "epochs_per_slashings_vector":
			assert.Equal(t, "36", v)
		case "historical_roots_limit":
			assert.Equal(t, "37", v)
		case "validator_registry_limit":
			assert.Equal(t, "38", v)
		case "base_reward_factor":
			assert.Equal(t, "39", v)
		case "whistleblower_reward_quotient":
			assert.Equal(t, "40", v)
		case "proposer_reward_quotient":
			assert.Equal(t, "41", v)
		case "inactivity_penalty_quotient":
			assert.Equal(t, "42", v)
		case "min_slashing_penalty_quotient":
			assert.Equal(t, "43", v)
		case "proportional_slashing_multiplier":
			assert.Equal(t, "44", v)
		case "max_proposer_slashings":
			assert.Equal(t, "45", v)
		case "max_attester_slashings":
			assert.Equal(t, "46", v)
		case "max_attestations":
			assert.Equal(t, "47", v)
		case "max_deposits":
			assert.Equal(t, "48", v)
		case "max_voluntary_exits":
			assert.Equal(t, "49", v)
		case "domain_beacon_proposer":
			assert.Equal(t, "0x30303031", v)
		case "domain_beacon_attester":
			assert.Equal(t, "0x30303032", v)
		case "domain_randao":
			assert.Equal(t, "0x30303033", v)
		case "domain_deposit":
			assert.Equal(t, "0x30303034", v)
		case "domain_voluntary_exit":
			assert.Equal(t, "0x30303035", v)
		case "domain_selection_proof":
			assert.Equal(t, "0x30303036", v)
		case "domain_aggregate_and_proof":
			assert.Equal(t, "0x30303037", v)
		default:
			t.Errorf("Incorrect key: %s", k)
		}
	}
}

func TestGetDepositContract(t *testing.T) {
	const chainId = 99
	const address = "0x0000000000000000000000000000000000000009"
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.DepositChainID = chainId
	config.DepositContractAddress = address
	params.OverrideBeaconConfig(config)

	s := Server{}
	resp, err := s.GetDepositContract(context.Background(), &types.Empty{})
	require.NoError(t, err)
	assert.Equal(t, uint64(chainId), resp.Data.ChainId)
	assert.Equal(t, address, resp.Data.Address)
}

func TestForkSchedule_Ok(t *testing.T) {
	genesisForkVersion := []byte("Genesis")
	firstForkVersion, firstForkEpoch := []byte("First"), uint64(100)
	secondForkVersion, secondForkEpoch := []byte("Second"), uint64(200)
	thirdForkVersion, thirdForkEpoch := []byte("Third"), uint64(300)

	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.GenesisForkVersion = genesisForkVersion
	// Create fork schedule adding keys in non-sorted order.
	schedule := make(map[uint64][]byte, 3)
	schedule[secondForkEpoch] = secondForkVersion
	schedule[firstForkEpoch] = firstForkVersion
	schedule[thirdForkEpoch] = thirdForkVersion
	config.ForkVersionSchedule = schedule
	params.OverrideBeaconConfig(config)

	s := &Server{}
	resp, err := s.GetForkSchedule(context.Background(), &types.Empty{})
	require.NoError(t, err)
	require.Equal(t, 3, len(resp.Data))
	fork := resp.Data[0]
	assert.DeepEqual(t, genesisForkVersion, fork.PreviousVersion)
	assert.DeepEqual(t, firstForkVersion, fork.CurrentVersion)
	assert.Equal(t, firstForkEpoch, fork.Epoch)
	fork = resp.Data[1]
	assert.DeepEqual(t, firstForkVersion, fork.PreviousVersion)
	assert.DeepEqual(t, secondForkVersion, fork.CurrentVersion)
	assert.Equal(t, secondForkEpoch, fork.Epoch)
	fork = resp.Data[2]
	assert.DeepEqual(t, secondForkVersion, fork.PreviousVersion)
	assert.DeepEqual(t, thirdForkVersion, fork.CurrentVersion)
	assert.Equal(t, thirdForkEpoch, fork.Epoch)
}

func TestForkSchedule_NoForks(t *testing.T) {
	s := &Server{}
	resp, err := s.GetForkSchedule(context.Background(), &types.Empty{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(resp.Data))
}
