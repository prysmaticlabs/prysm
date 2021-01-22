package beaconv1

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/status-im/keycard-go/hexutils"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetForkSchedule retrieve all scheduled upcoming forks this node is aware of.
func (bs *Server) GetForkSchedule(ctx context.Context, req *ptypes.Empty) (*ethpb.ForkScheduleResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetSpec retrieves specification configuration (without Phase 1 params) used on this node. Specification params list
// Values are returned with following format:
// - any value starting with 0x in the spec is returned as a hex string.
// - all other values are returned as number.
func (bs *Server) GetSpec(ctx context.Context, req *ptypes.Empty) (*ethpb.SpecResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetDepositContract retrieves deposit contract address and genesis fork version.
func (bs *Server) GetDepositContract(ctx context.Context, req *ptypes.Empty) (*ethpb.DepositContractResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.GetDepositContract")
	defer span.End()

	buf := bytes.NewBuffer(params.BeaconConfig().GenesisForkVersion)
	chainId, err := binary.ReadUvarint(buf)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not obtain genesis fork version: %v", err)
	}
	byteAddress := [20]byte(bs.PowchainInfoFetcher.DepositContractAddress())
	hexAddress := hexutils.BytesToHex(byteAddress[:])

	return &ethpb.DepositContractResponse{
		Data: &ethpb.DepositContract{
			ChainId: uint64(chainId),
			Address: hexAddress,
		},
	}, nil
}
