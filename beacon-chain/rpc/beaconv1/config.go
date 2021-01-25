package beaconv1

import (
	"context"
	"errors"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// GetForkSchedule retrieve all scheduled upcoming forks this node is aware of.
func (bs *Server) GetForkSchedule(ctx context.Context, req *ptypes.Empty) (*ethpb.ForkScheduleResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetSpec retrieves specification configuration (without Phase 1 params) used on this node. Specification params list
// Values are returned with following format:
// - any value starting with 0x in the spec is returned as a hex string.
// - all other values are returned as number.
func (bs *Server) GetSpec(ctx context.Context, _ *ptypes.Empty) (*ethpb.SpecResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconV1.GetSpec")
	defer span.End()

	data := make(map[string]string)
	data["config_name"] = params.BeaconConfig().NetworkName
	data["max_committees_per_slot"] = strconv.FormatUint(params.BeaconConfig().MaxCommitteesPerSlot, 10)
	data["target_committee_size"] = strconv.FormatUint(params.BeaconConfig().TargetCommitteeSize, 10)
	data["max_validators_per_committee"] = strconv.FormatUint(params.BeaconConfig().MaxValidatorsPerCommittee, 10)
	data["min_per_epoch_churn_limit"] = strconv.FormatUint(params.BeaconConfig().MinPerEpochChurnLimit, 10)
	data["churn_limit_quotient"] = strconv.FormatUint(params.BeaconConfig().ChurnLimitQuotient, 10)
	data["shuffle_round_count"] = strconv.FormatUint(params.BeaconConfig().ShuffleRoundCount, 10)
	data["min_genesis_active_validator_count"] = strconv.FormatUint(params.BeaconConfig().MinGenesisActiveValidatorCount, 10)
	data["min_genesis_time"] = strconv.FormatUint(params.BeaconConfig().MinGenesisTime, 10)
	data["hysteresis_quotient"] = strconv.FormatUint(params.BeaconConfig().HysteresisQuotient, 10)
	data["hysteresis_downward_multiplier"] = strconv.FormatUint(params.BeaconConfig().HysteresisDownwardMultiplier, 10)
	data["hysteresis_upward_multiplier"] = strconv.FormatUint(params.BeaconConfig().HysteresisUpwardMultiplier, 10)
	data["safe_slots_to_update_justified"] = strconv.FormatUint(params.BeaconConfig().SafeSlotsToUpdateJustified, 10)
	data["eth1_follow_distance"] = strconv.FormatUint(params.BeaconConfig().Eth1FollowDistance, 10)
	data["target_aggregators_per_committee"] = strconv.FormatUint(params.BeaconConfig().TargetAggregatorsPerCommittee, 10)
	data["random_subnets_per_validator"] = strconv.FormatUint(params.BeaconConfig().RandomSubnetsPerValidator, 10)
	data["epochs_per_random_subnet_subscription"] = strconv.FormatUint(params.BeaconConfig().EpochsPerRandomSubnetSubscription, 10)
	data["seconds_per_eth1_block"] = strconv.FormatUint(params.BeaconConfig().SecondsPerETH1Block, 10)
	data["deposit_chain_id"] = strconv.FormatUint(params.BeaconConfig().DepositChainID, 10)
	data["deposit_network_id"] = strconv.FormatUint(params.BeaconConfig().DepositNetworkID, 10)
	data["deposit_contract_address"] = params.BeaconConfig().DepositContractAddress
	data["min_deposit_amount"] = strconv.FormatUint(params.BeaconConfig().MinDepositAmount, 10)
	data["max_effective_balance"] = strconv.FormatUint(params.BeaconConfig().MaxEffectiveBalance, 10)
	data["ejection_balance"] = strconv.FormatUint(params.BeaconConfig().EjectionBalance, 10)
	data["effective_balance_increment"] = strconv.FormatUint(params.BeaconConfig().EffectiveBalanceIncrement, 10)
	data["genesis_fork_version"] = hexutil.Encode(params.BeaconConfig().GenesisForkVersion)
	data["bls_withdrawal_prefix"] = hexutil.Encode([]byte{params.BeaconConfig().BLSWithdrawalPrefixByte})
	data["genesis_delay"] = strconv.FormatUint(params.BeaconConfig().GenesisDelay, 10)
	data["seconds_per_slot"] = strconv.FormatUint(params.BeaconConfig().SecondsPerSlot, 10)
	data["min_attestation_inclusion_delay"] = strconv.FormatUint(params.BeaconConfig().MinAttestationInclusionDelay, 10)
	data["slots_per_epoch"] = strconv.FormatUint(params.BeaconConfig().SlotsPerEpoch, 10)
	data["min_seed_lookahead"] = strconv.FormatUint(params.BeaconConfig().MinSeedLookahead, 10)
	data["max_seed_lookahead"] = strconv.FormatUint(params.BeaconConfig().MaxSeedLookahead, 10)
	data["epochs_per_eth1_voting_period"] = strconv.FormatUint(params.BeaconConfig().EpochsPerEth1VotingPeriod, 10)
	data["slots_per_historical_root"] = strconv.FormatUint(params.BeaconConfig().SlotsPerHistoricalRoot, 10)
	data["min_validator_withdrawability_delay"] = strconv.FormatUint(params.BeaconConfig().MinValidatorWithdrawabilityDelay, 10)
	data["shard_committee_period"] = strconv.FormatUint(params.BeaconConfig().ShardCommitteePeriod, 10)
	data["min_epochs_to_inactivity_penalty"] = strconv.FormatUint(params.BeaconConfig().MinEpochsToInactivityPenalty, 10)
	data["epochs_per_historical_vector"] = strconv.FormatUint(params.BeaconConfig().EpochsPerHistoricalVector, 10)
	data["epochs_per_slashings_vector"] = strconv.FormatUint(params.BeaconConfig().EpochsPerSlashingsVector, 10)
	data["historical_roots_limit"] = strconv.FormatUint(params.BeaconConfig().HistoricalRootsLimit, 10)
	data["validator_registry_limit"] = strconv.FormatUint(params.BeaconConfig().ValidatorRegistryLimit, 10)
	data["base_reward_factor"] = strconv.FormatUint(params.BeaconConfig().BaseRewardFactor, 10)
	data["whistleblower_reward_quotient"] = strconv.FormatUint(params.BeaconConfig().WhistleBlowerRewardQuotient, 10)
	data["proposer_reward_quotient"] = strconv.FormatUint(params.BeaconConfig().ProposerRewardQuotient, 10)
	data["inactivity_penalty_quotient"] = strconv.FormatUint(params.BeaconConfig().InactivityPenaltyQuotient, 10)
	data["min_slashing_penalty_quotient"] = strconv.FormatUint(params.BeaconConfig().MinSlashingPenaltyQuotient, 10)
	data["proportional_slashing_multiplier"] = strconv.FormatUint(params.BeaconConfig().ProportionalSlashingMultiplier, 10)
	data["max_proposer_slashings"] = strconv.FormatUint(params.BeaconConfig().MaxProposerSlashings, 10)
	data["max_attester_slashings"] = strconv.FormatUint(params.BeaconConfig().MaxAttesterSlashings, 10)
	data["max_attestations"] = strconv.FormatUint(params.BeaconConfig().MaxAttestations, 10)
	data["max_deposits"] = strconv.FormatUint(params.BeaconConfig().MaxDeposits, 10)
	data["max_voluntary_exits"] = strconv.FormatUint(params.BeaconConfig().MaxVoluntaryExits, 10)
	data["domain_beacon_proposer"] = hexutil.Encode(params.BeaconConfig().DomainBeaconProposer[:])
	data["domain_beacon_attester"] = hexutil.Encode(params.BeaconConfig().DomainBeaconAttester[:])
	data["domain_randao"] = hexutil.Encode(params.BeaconConfig().DomainRandao[:])
	data["domain_deposit"] = hexutil.Encode(params.BeaconConfig().DomainDeposit[:])
	data["domain_voluntary_exit"] = hexutil.Encode(params.BeaconConfig().DomainVoluntaryExit[:])
	data["domain_selection_proof"] = hexutil.Encode(params.BeaconConfig().DomainSelectionProof[:])
	data["domain_aggregate_and_proof"] = hexutil.Encode(params.BeaconConfig().DomainAggregateAndProof[:])

	return &ethpb.SpecResponse{Data: data}, nil
}

// GetDepositContract retrieves deposit contract address and genesis fork version.
func (bs *Server) GetDepositContract(ctx context.Context, req *ptypes.Empty) (*ethpb.DepositContractResponse, error) {
	return nil, errors.New("unimplemented")
}
