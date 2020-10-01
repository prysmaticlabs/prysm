package beacon_v1

import (
	"context"
	"errors"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

func GetForkSchedule(ctx context.Context, req *ptypes.Empty) (*ethpb.ForkScheduleResponse, error) {
	return nil, errors.New("unimplemented")
}

func GetSpec(ctx context.Context, req *ptypes.Empty) (*ethpb.SpecResponse, error) {
	return nil, errors.New("unimplemented")
}

func GetDepositContract(ctx context.Context, req *ptypes.Empty) (*ethpb.DepositContractResponse, error) {
	return nil, errors.New("unimplemented")
}
