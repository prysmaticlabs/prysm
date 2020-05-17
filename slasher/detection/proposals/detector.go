// Package proposals defines an implementation of a double-propose
// detector in the slasher runtime.
package proposals

import (
	"bytes"
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/db"
	status "github.com/prysmaticlabs/prysm/slasher/db/types"
	"go.opencensus.io/trace"
)

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
	headersFromIdx, err := dd.slasherDB.BlockHeaders(ctx, incomingBlk.Header.Slot, incomingBlk.Header.ProposerIndex)
	if err != nil {
		return nil, err
	}
	for _, blockHeader := range headersFromIdx {
		if bytes.Equal(blockHeader.Signature, incomingBlk.Signature) {
			continue
		}
		ps := &ethpb.ProposerSlashing{Header_1: incomingBlk, Header_2: blockHeader}
		if err := dd.slasherDB.SaveProposerSlashing(ctx, status.Active, ps); err != nil {
			return nil, err
		}
		return ps, nil
	}
	if err := dd.slasherDB.SaveBlockHeader(ctx, incomingBlk); err != nil {
		return nil, err
	}
	return nil, nil
}
