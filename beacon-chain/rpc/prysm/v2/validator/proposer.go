package validator

import (
	"context"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/interop"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/interfaces"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBlock is called by a proposer during its assigned slot to request a block to sign
// by passing in the slot and the signed randao reveal of the slot. This is used by a validator
// after the altair fork epoch has been encountered.
func (vs *Server) GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*prysmv2.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.GetBlock")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(req.Slot)))

	blkData, err := vs.V1Server.BuildBlockData(ctx, req)
	if err != nil {
		return nil, err
	}

	// Use zero hash as stub for state root to compute later.
	stateRoot := params.BeaconConfig().ZeroHash[:]

	// Ugly hack to allow this to compile both for mainnet
	// and minimal configs.
	mockAgg := &prysmv2.SyncAggregate{SyncCommitteeBits: []byte{}}
	var bVector []byte
	if mockAgg.SyncCommitteeBits.Len() == 512 {
		bVector = bitfield.NewBitvector512()
	} else {
		bVector = bitfield.NewBitvector32()
	}
	infiniteSignature := [96]byte{0xC0}
	blk := &prysmv2.BeaconBlock{
		Slot:          req.Slot,
		ParentRoot:    blkData.ParentRoot,
		StateRoot:     stateRoot,
		ProposerIndex: blkData.ProposerIdx,
		Body: &prysmv2.BeaconBlockBody{
			Eth1Data:          blkData.Eth1Data,
			Deposits:          blkData.Deposits,
			Attestations:      blkData.Attestations,
			RandaoReveal:      req.RandaoReveal,
			ProposerSlashings: blkData.ProposerSlashings,
			AttesterSlashings: blkData.AttesterSlashings,
			VoluntaryExits:    blkData.VoluntaryExits,
			Graffiti:          blkData.Graffiti[:],
			// TODO: Add in actual aggregates
			SyncAggregate: &prysmv2.SyncAggregate{
				SyncCommitteeBits:      bVector,
				SyncCommitteeSignature: infiniteSignature[:],
			},
		},
	}
	// Compute state root with the newly constructed block.
	stateRoot, err = vs.V1Server.ComputeStateRoot(
		ctx,
		interfaces.WrappedAltairSignedBeaconBlock(
			&prysmv2.SignedBeaconBlock{Block: blk, Signature: make([]byte, 96)},
		),
	)
	if err != nil {
		interop.WriteBlockToDisk(
			interfaces.WrappedAltairSignedBeaconBlock(
				&prysmv2.SignedBeaconBlock{Block: blk},
			), true, /*failed*/
		)
		return nil, status.Errorf(codes.Internal, "Could not compute state root: %v", err)
	}
	blk.StateRoot = stateRoot

	return blk, nil
}

// ProposeBlock is called by a proposer during its assigned slot to create a block in an attempt
// to get it processed by the beacon node as the canonical head.
func (vs *Server) ProposeBlock(ctx context.Context, rBlk *prysmv2.SignedBeaconBlock) (*ethpb.ProposeResponse, error) {
	blk := interfaces.WrappedAltairSignedBeaconBlock(rBlk)
	return vs.V1Server.ProposeBlockGeneric(ctx, blk)
}
