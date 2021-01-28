package beaconv1

import (
	"context"
	"errors"
	"sort"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// GetForkSchedule retrieve all scheduled upcoming forks this node is aware of.
func (bs *Server) GetForkSchedule(ctx context.Context, _ *ptypes.Empty) (*ethpb.ForkScheduleResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetForkSchedule")
	defer span.End()

	schedule := params.BeaconConfig().ForkVersionSchedule
	if len(schedule) == 0 {
		return &ethpb.ForkScheduleResponse{
			Data: make([]*ethpb.Fork, 0),
		}, nil
	}

	epochs := sortedEpochs(schedule)
	forks := make([]*ethpb.Fork, len(schedule))
	var previous, current []byte
	for i, e := range epochs {
		if i == 0 {
			previous = params.BeaconConfig().GenesisForkVersion
		} else {
			previous = current
		}
		current = schedule[e]
		forks[i] = &ethpb.Fork{
			PreviousVersion: previous,
			CurrentVersion:  current,
			Epoch:           e,
		}
	}

	return &ethpb.ForkScheduleResponse{
		Data: forks,
	}, nil
}

// GetSpec retrieves specification configuration (without Phase 1 params) used on this node. Specification params list
// Values are returned with following format:
// - any value starting with 0x in the spec is returned as a hex string.
// - all other values are returned as number.
func (bs *Server) GetSpec(ctx context.Context, req *ptypes.Empty) (*ethpb.SpecResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetDepositContract retrieves deposit contract address and genesis fork version.
func (bs *Server) GetDepositContract(ctx context.Context, _ *ptypes.Empty) (*ethpb.DepositContractResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetDepositContract")
	defer span.End()

	return &ethpb.DepositContractResponse{
		Data: &ethpb.DepositContract{
			ChainId: params.BeaconConfig().DepositChainID,
			Address: params.BeaconConfig().DepositContractAddress,
		},
	}, nil
}

func sortedEpochs(forkSchedule map[uint64][]byte) []uint64 {
	sortedEpochs := make([]uint64, len(forkSchedule))
	i := 0
	for k := range forkSchedule {
		sortedEpochs[i] = k
		i++
	}
	sort.Slice(sortedEpochs, func(a, b int) bool { return sortedEpochs[a] < sortedEpochs[b] })
	return sortedEpochs
}
