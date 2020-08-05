package debug

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBlock in an ssz-encoded format by block root.
func (ds *Server) GetBlock(
	ctx context.Context,
	req *pbrpc.BlockRequest,
) (*pbrpc.SSZResponse, error) {
	root := bytesutil.ToBytes32(req.BlockRoot)
	signedBlock, err := ds.BeaconDB.Block(ctx, root)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve block by root: %v", err)
	}
	if signedBlock == nil {
		return &pbrpc.SSZResponse{Encoded: make([]byte, 0)}, nil
	}
	encoded, err := signedBlock.MarshalSSZ()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal block: %v", err)
	}
	return &pbrpc.SSZResponse{
		Encoded: encoded,
	}, nil
}

// GetInclusionSlot of an attestation in block.
func (ds *Server) GetInclusionSlot(ctx context.Context, req *pbrpc.InclusionSlotRequest) (*pbrpc.InclusionSlotResponse, error) {
	ds.GenesisTimeFetcher.CurrentSlot()

	// Attestation has one epoch to get included in the chain. This blocks users from requesting too soon.
	epochBack := ds.GenesisTimeFetcher.CurrentSlot() - params.BeaconConfig().SlotsPerEpoch
	if epochBack < req.Slot {
		return nil, fmt.Errorf("attestation has one epoch window, please request slot older than %d", epochBack)
	}

	// Attestation could be in blocks between slot + 1 to slot + epoch_duration.
	startSlot := req.Slot + params.BeaconConfig().MinAttestationInclusionDelay
	endSlot := req.Slot + params.BeaconConfig().SlotsPerEpoch

	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot)
	blks, err := ds.BeaconDB.Blocks(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve blocks: %v", err)
	}

	inclusionSlot := uint64(1<<64 - 1)
	targetStates := make(map[[32]byte]*state.BeaconState)
	for _, blk := range blks {
		for _, a := range blk.Block.Body.Attestations {
			tr := bytesutil.ToBytes32(a.Data.Target.Root)
			s, ok := targetStates[tr]
			if !ok {
				s, err = ds.StateGen.StateByRoot(ctx, tr)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "Could not retrieve state: %v", err)
				}
				targetStates[tr] = s
			}
			c, err := helpers.BeaconCommitteeFromState(s, a.Data.Slot, a.Data.CommitteeIndex)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get committee: %v", err)
			}
			indices := attestationutil.AttestingIndices(a.AggregationBits, c)
			for _, i := range indices {
				if req.Id == i && req.Slot == a.Data.Slot {
					inclusionSlot = blk.Block.Slot
					break
				}
			}
		}
	}

	return &pbrpc.InclusionSlotResponse{Slot: inclusionSlot}, nil
}
