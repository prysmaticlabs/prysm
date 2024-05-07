package validator

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/math"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// proposalBlock accumulates data needed to respond to the proposer GetBeaconBlock request.
type proposalResponseConstructor struct {
	block           interfaces.SignedBeaconBlock
	overrideBuilder bool
	local           *PayloadOption
	builder         *PayloadOption
	winner          *PayloadOption
}

var errNoProposalSource = errors.New("proposal process did not pick between builder and local block")

// construct picks the best proposal and sets the execution header/payload attributes for it on the block.
// It also takes the parent state as an argument so that it can compute and set the state root using the
// completely updated block.
func (pc *proposalResponseConstructor) construct(ctx context.Context, st state.BeaconState, builderBoostFactor uint64) (*ethpb.GenericBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "proposalResponseConstructor.construct")
	defer span.End()
	if err := blocks.HasNilErr(pc.block); err != nil {
		return nil, err
	}
	best, err := choosePayload(ctx, pc, builderBoostFactor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error constructing execution payload for block: %v", err)
	}
	best, err = pc.complete(ctx, best, st)
	if err != nil {
		return nil, err
	}
	return constructGenericBeaconBlock(pc.block, best.bid, best.bundle)
}

func (pc *proposalResponseConstructor) complete(ctx context.Context, best *PayloadOption, st state.BeaconState) (*PayloadOption, error) {
	err := pc.completeWithBest(ctx, best, st)
	if err == nil {
		return best, nil
	}
	// We can fall back from the builder to local, but not the other way. If local fails, we're done.
	if best == pc.local {
		return nil, err
	}

	// Try again with the local payload. If this fails then we're truly done.
	return pc.local, pc.completeWithBest(ctx, pc.local, st)
}

func (pc *proposalResponseConstructor) completeWithBest(ctx context.Context, best *PayloadOption, st state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "proposalResponseConstructor.completeWithBest")
	defer span.End()
	if best.IsNil() {
		return errNoProposalSource
	}

	if err := pc.block.SetExecution(best.ExecutionData); err != nil {
		return err
	}
	if pc.block.Version() >= version.Deneb {
		kzgc := best.kzgCommitments
		if best.bundle != nil {
			kzgc = best.bundle.KzgCommitments
		}
		if err := pc.block.SetBlobKzgCommitments(kzgc); err != nil {
			return err
		}
	}

	root, err := transition.CalculateStateRoot(ctx, st, pc.block)
	if err != nil {
		return errors.Wrapf(err, "could not calculate state root for proposal with parent root=%#x at slot %d", pc.block.Block().ParentRoot(), st.Slot())
	}
	log.WithField("beaconStateRoot", fmt.Sprintf("%#x", root)).Debugf("Computed state root")
	pc.block.SetStateRoot(root[:])

	return nil
}

// constructGenericBeaconBlock constructs a `GenericBeaconBlock` based on the block version and other parameters.
func constructGenericBeaconBlock(blk interfaces.SignedBeaconBlock, bid math.Wei, bundle *enginev1.BlobsBundle) (*ethpb.GenericBeaconBlock, error) {
	if err := blocks.HasNilErr(blk); err != nil {
		return nil, err
	}
	blockProto, err := blk.Block().Proto()
	if err != nil {
		return nil, err
	}
	payloadValue := math.WeiToBigInt(bid).String()

	switch pb := blockProto.(type) {
	case *ethpb.BeaconBlockDeneb:
		denebContents := &ethpb.BeaconBlockContentsDeneb{Block: pb}
		if bundle != nil {
			denebContents.KzgProofs = bundle.Proofs
			denebContents.Blobs = bundle.Blobs
		}
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Deneb{Deneb: denebContents}, IsBlinded: false, PayloadValue: payloadValue}, nil
	case *ethpb.BlindedBeaconBlockDeneb:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedDeneb{BlindedDeneb: pb}, IsBlinded: true, PayloadValue: payloadValue}, nil
	case *ethpb.BeaconBlockCapella:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Capella{Capella: pb}, IsBlinded: false, PayloadValue: payloadValue}, nil
	case *ethpb.BlindedBeaconBlockCapella:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedCapella{BlindedCapella: pb}, IsBlinded: true, PayloadValue: payloadValue}, nil
	case *ethpb.BeaconBlockBellatrix:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Bellatrix{Bellatrix: pb}, IsBlinded: false, PayloadValue: payloadValue}, nil
	case *ethpb.BlindedBeaconBlockBellatrix:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: pb}, IsBlinded: true, PayloadValue: payloadValue}, nil
	case *ethpb.BeaconBlockAltair:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Altair{Altair: pb}}, nil
	case *ethpb.BeaconBlock:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Phase0{Phase0: pb}}, nil
	}
	return nil, fmt.Errorf("unknown .block version: %d", blk.Version())
}

// PayloadOption allows a payload to be conveniently coupled with its bid value (builder or local).
type PayloadOption struct {
	interfaces.ExecutionData
	bid            math.Wei
	bundle         *enginev1.BlobsBundle
	kzgCommitments [][]byte
}

// NewPayloadOption initializes a PayloadOption. This should only be used to represent payloads that have a bid,
// otherwise directly use an ExecutionData type.
func NewPayloadOption(p interfaces.ExecutionData, bid math.Wei, bundle *enginev1.BlobsBundle, kzgc [][]byte) (*PayloadOption, error) {
	if err := blocks.HasNilErr(p); err != nil {
		return nil, err
	}
	if bid == nil {
		bid = math.ZeroWei
	}
	return &PayloadOption{ExecutionData: p, bid: bid, bundle: bundle, kzgCommitments: kzgc}, nil
}

func (p *PayloadOption) IsNil() bool {
	return p == nil || p.ExecutionData.IsNil()
}

// ValueInGwei is a helper to converts the bid value to its gwei representation.
func (p *PayloadOption) ValueInGwei() math.Gwei {
	return math.WeiToGwei(p.bid)
}
