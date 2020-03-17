package proposals

import (
	"bytes"
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/slasher/db"
	status "github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/proposals/iface"
	"go.opencensus.io/trace"
)

var _ = iface.ProposalsDetector(&ProposeDetector{})

// ProposeDetector defines a struct which can detect slashable
// block proposals.
type ProposeDetector struct {
	slasherDB db.Database
}

// NewProposeDetector creates a new instance of a struct.
func NewProposeDetector(db db.Database) *ProposeDetector {
	return &ProposeDetector{
		slasherDB: db,
	}
}

// DetectDoublePropose detects double proposals given a block by looking in the db.
func (dd *ProposeDetector) DetectDoublePropose(
	ctx context.Context,
	incomingBlk *ethpb.SignedBeaconBlockHeader,
) (*ethpb.ProposerSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "detector.DetectDoublePropose")
	defer span.End()
	epoch := helpers.SlotToEpoch(incomingBlk.Header.Slot)
	//validatorIdx:=blk.Header.ProposerIndex
	proposerIdx := uint64(0)
	bha, err := dd.slasherDB.BlockHeaders(ctx, epoch, proposerIdx)
	if err != nil {
		return nil, err
	}
	for _, bh := range bha {
		if bytes.Equal(bh.Signature, incomingBlk.Signature) {
			continue
		}
		ps := &ethpb.ProposerSlashing{ProposerIndex: proposerIdx, Header_1: incomingBlk, Header_2: bh}
		err := dd.slasherDB.SaveProposerSlashing(ctx, status.Active, ps)
		if err != nil {
			return nil, err
		}
		return ps, nil
	}
	return nil, nil
}
