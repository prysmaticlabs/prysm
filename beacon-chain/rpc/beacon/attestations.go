package beacon

import (
	"context"
	"sort"
	"strconv"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// sortableAttestations implements the Sort interface to sort attestations
// by slot as the canonical sorting attribute.
type sortableAttestations []*ethpb.Attestation

func (s sortableAttestations) Len() int      { return len(s) }
func (s sortableAttestations) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortableAttestations) Less(i, j int) bool {
	return s[i].Data.Slot < s[j].Data.Slot
}

// ListAttestations retrieves attestations by block root, slot, or epoch.
// Attestations are sorted by data slot by default.
//
// The server may return an empty list when no attestations match the given
// filter criteria. This RPC should not return NOT_FOUND. Only one filter
// criteria should be used.
func (bs *Server) ListAttestations(
	ctx context.Context, req *ethpb.ListAttestationsRequest,
) (*ethpb.ListAttestationsResponse, error) {
	if int(req.PageSize) > flags.Get().MaxPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "Requested page size %d can not be greater than max size %d",
			req.PageSize, flags.Get().MaxPageSize)
	}
	var atts []*ethpb.Attestation
	var err error
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListAttestationsRequest_Genesis:
		genBlk, err := bs.BeaconDB.GenesisBlock(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not genesis block: %v", err)
		}
		if genBlk == nil {
			return nil, status.Error(codes.Internal, "Could not find genesis block")
		}
		genesisRoot, err := ssz.HashTreeRoot(genBlk.Block)
		if err != nil {
			return nil, err
		}
		atts, err = bs.BeaconDB.Attestations(ctx, filters.NewFilter().SetHeadBlockRoot(genesisRoot[:]))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch genesis attestations: %v", err)
		}
	case *ethpb.ListAttestationsRequest_HeadBlockRoot:
		atts, err = bs.BeaconDB.Attestations(ctx, filters.NewFilter().SetHeadBlockRoot(q.HeadBlockRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch attestations: %v", err)
		}
	case *ethpb.ListAttestationsRequest_SourceEpoch:
		atts, err = bs.BeaconDB.Attestations(ctx, filters.NewFilter().SetSourceEpoch(q.SourceEpoch))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch attestations: %v", err)
		}
	case *ethpb.ListAttestationsRequest_SourceRoot:
		atts, err = bs.BeaconDB.Attestations(ctx, filters.NewFilter().SetSourceRoot(q.SourceRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch attestations: %v", err)
		}
	case *ethpb.ListAttestationsRequest_TargetEpoch:
		atts, err = bs.BeaconDB.Attestations(ctx, filters.NewFilter().SetTargetEpoch(q.TargetEpoch))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch attestations: %v", err)
		}
	case *ethpb.ListAttestationsRequest_TargetRoot:
		atts, err = bs.BeaconDB.Attestations(ctx, filters.NewFilter().SetTargetRoot(q.TargetRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not fetch attestations: %v", err)
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "Must specify a filter criteria for fetching attestations")
	}
	// We sort attestations according to the Sortable interface.
	sort.Sort(sortableAttestations(atts))
	numAttestations := len(atts)

	// If there are no attestations, we simply return a response specifying this.
	// Otherwise, attempting to paginate 0 attestations below would result in an error.
	if numAttestations == 0 {
		return &ethpb.ListAttestationsResponse{
			Attestations:  make([]*ethpb.Attestation, 0),
			TotalSize:     int32(0),
			NextPageToken: strconv.Itoa(0),
		}, nil
	}

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numAttestations)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not paginate attestations: %v", err)
	}
	return &ethpb.ListAttestationsResponse{
		Attestations:  atts[start:end],
		TotalSize:     int32(numAttestations),
		NextPageToken: nextPageToken,
	}, nil
}

// StreamAttestations to clients at the end of every slot. This method retrieves the
// aggregated attestations currently in the pool at the start of a slot and sends
// them over a gRPC stream.
func (bs *Server) StreamAttestations(
	_ *ptypes.Empty, stream ethpb.BeaconChain_StreamAttestationsServer,
) error {
	for {
		select {
		case <-bs.SlotTicker.C():
			atts := bs.AttestationsPool.AggregatedAttestations()
			for i := 0; i < len(atts); i++ {
				if err := stream.Send(atts[i]); err != nil {
					return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
				}
			}
		case <-bs.Ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		}
	}
}

// AttestationPool retrieves pending attestations.
//
// The server returns a list of attestations that have been seen but not
// yet processed. Pool attestations eventually expire as the slot
// advances, so an attestation missing from this request does not imply
// that it was included in a block. The attestation may have expired.
// Refer to the ethereum 2.0 specification for more details on how
// attestations are processed and when they are no longer valid.
// https://github.com/ethereum/eth2.0-specs/blob/dev/specs/core/0_beacon-chain.md#attestations
func (bs *Server) AttestationPool(
	ctx context.Context, req *ethpb.AttestationPoolRequest,
) (*ethpb.AttestationPoolResponse, error) {
	if int(req.PageSize) > flags.Get().MaxPageSize {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Requested page size %d can not be greater than max size %d",
			req.PageSize,
			flags.Get().MaxPageSize,
		)
	}
	atts := bs.AttestationsPool.AggregatedAttestations()
	numAtts := len(atts)
	if numAtts == 0 {
		return &ethpb.AttestationPoolResponse{
			Attestations:  make([]*ethpb.Attestation, 0),
			TotalSize:     int32(0),
			NextPageToken: strconv.Itoa(0),
		}, nil
	}
	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numAtts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not paginate attestations: %v", err)
	}
	return &ethpb.AttestationPoolResponse{
		Attestations:  atts[start:end],
		TotalSize:     int32(numAtts),
		NextPageToken: nextPageToken,
	}, nil
}
