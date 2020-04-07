package beacon

import (
	"context"
	"fmt"
	"reflect"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetChainInfo retrieves information about the beacon chain.
// This information is independent of the current state of the beacon node returning the information.
func (bs *Server) GetChainInfo(ctx context.Context, _ *ptypes.Empty) (*ethpb.ChainInfo, error) {
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not retrieve head state")
	}

	genesisTime, err := ptypes.TimestampProto(bs.GenesisTimeFetcher.GenesisTime())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert genesis time to proto: %v", err)
	}

	depositContractAddr, err := bs.BeaconDB.DepositContractAddress(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve contract address from db: %v", err)
	}

	blsWithdrawalPrefix := make([]byte, 1)
	blsWithdrawalPrefix[0] = params.BeaconConfig().BLSWithdrawalPrefixByte

	forkVersionSchedule := make([]*ethpb.ForkDefinition, len(params.BeaconConfig().ForkVersionSchedule))
	for epoch, version := range params.BeaconConfig().ForkVersionSchedule {
		definition := &ethpb.ForkDefinition{
			Version: version,
			Epoch:   epoch,
		}
		forkVersionSchedule[epoch] = definition
	}

	return &ethpb.ChainInfo{
		GenesisTime:                      genesisTime,
		DepositContractAddress:           depositContractAddr,
		SecondsPerSlot:                   params.BeaconConfig().SecondsPerSlot,
		SlotsPerEpoch:                    params.BeaconConfig().SlotsPerEpoch,
		MaxSeedLookahead:                 params.BeaconConfig().MaxSeedLookahead,
		MinValidatorWithdrawabilityDelay: params.BeaconConfig().MinValidatorWithdrawabilityDelay,
		PersistentCommitteePeriod:        params.BeaconConfig().PersistentCommitteePeriod,
		MinEpochsToInactivityPenalty:     params.BeaconConfig().MinEpochsToInactivityPenalty,
		Eth1FollowDistance:               params.BeaconConfig().Eth1FollowDistance,
		FarFutureEpoch:                   params.BeaconConfig().FarFutureEpoch,
		GenesisForkVersion:               params.BeaconConfig().GenesisForkVersion,
		GenesisValidatorsRoot:            headState.GenesisValidatorRoot(),
		MinimumDepositAmount:             params.BeaconConfig().MinDepositAmount,
		MaximumEffectiveBalance:          params.BeaconConfig().MaxEffectiveBalance,
		EffectiveBalanceIncrement:        params.BeaconConfig().EffectiveBalanceIncrement,
		EjectionBalance:                  params.BeaconConfig().EjectionBalance,
		BlsWithdrawalPrefix:              blsWithdrawalPrefix,
		PreviousForkVersion:              headState.Fork().PreviousVersion,
		CurrentForkVersion:               headState.Fork().CurrentVersion,
		CurrentForkEpoch:                 headState.Fork().Epoch,
		NextForkVersion:                  params.BeaconConfig().NextForkVersion,
		NextForkEpoch:                    params.BeaconConfig().NextForkEpoch,
		CurrentEpoch:                     helpers.CurrentEpoch(headState),
		ForkVersionSchedule:              forkVersionSchedule,
	}, nil
}

// GetBeaconConfig retrieves the current configuration parameters of the beacon chain.
func (bs *Server) GetBeaconConfig(ctx context.Context, _ *ptypes.Empty) (*ethpb.BeaconConfig, error) {

	conf := params.BeaconConfig()
	val := reflect.ValueOf(conf).Elem()
	numFields := val.Type().NumField()
	res := make(map[string]string, numFields)
	for i := 0; i < numFields; i++ {
		res[val.Type().Field(i).Name] = fmt.Sprintf("%v", val.Field(i).Interface())
	}
	return &ethpb.BeaconConfig{
		Config: res,
	}, nil
}
