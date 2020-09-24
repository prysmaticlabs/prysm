package beacon

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

func GetForkSchedule(ctx context.Context, req *ptypes.Empty) (*ethpb.ForkScheduleResponse, error) {
	return &ethpb.ForkScheduleResponse{}, nil
}

func GetSpec(ctx context.Context, req *ptypes.Empty) (*ethpb.SpecResponse, error) {
	return &ethpb.SpecResponse{}, nil
}

func GetDepositContract(ctx context.Context, req *ptypes.Empty) (*ethpb.DepositContractResponse, error) {
	return &ethpb.DepositContractResponse{}, nil
}
