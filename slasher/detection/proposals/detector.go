// Package proposals defines an implementation of a double-propose
// detector in the slasher runtime.
package proposals

import (
	"bytes"
	"context"

	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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
func (d *ProposeDetector) DetectDoublePropose(
	ctx context.Context,
	incomingBlk *ethpb.SignedBeaconBlockHeader,
) (*ethpb.ProposerSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "detector.DetectDoublePropose")
	defer span.End()
	headersFromIdx, err := d.slasherDB.BlockHeaders(ctx, incomingBlk.Header.Slot, incomingBlk.Header.ProposerIndex)
	if err != nil {
		return nil, err
	}
	for _, blockHeader := range headersFromIdx {
		if bytes.Equal(blockHeader.Signature, incomingBlk.Signature) {
			continue
		}
		ps := &ethpb.ProposerSlashing{Header_1: incomingBlk, Header_2: blockHeader}
		if err := d.slasherDB.SaveProposerSlashing(ctx, status.Active, ps); err != nil {
			return nil, err
		}
		return ps, nil
	}
	if err := d.slasherDB.SaveBlockHeader(ctx, incomingBlk); err != nil {
		return nil, err
	}
	return nil, nil
}

// DetectDoubleProposeNoUpdate detects double proposals for a given block header by db search
// without storing the incoming block to db.
func (d *ProposeDetector) DetectDoubleProposeNoUpdate(
	ctx context.Context,
	incomingBlk *ethpb.BeaconBlockHeader,
) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "detector.DetectDoubleProposeNoUpdate")
	defer span.End()
	headersFromIdx, err := d.slasherDB.BlockHeaders(ctx, incomingBlk.Slot, incomingBlk.ProposerIndex)
	if err != nil {
		return false, err
	}
	for _, blockHeader := range headersFromIdx {
		sameBodyRoot := bytes.Equal(blockHeader.Header.BodyRoot, incomingBlk.BodyRoot)
		sameStateRoot := bytes.Equal(blockHeader.Header.StateRoot, incomingBlk.StateRoot)
		sameParentRoot := bytes.Equal(blockHeader.Header.ParentRoot, incomingBlk.ParentRoot)
		if sameBodyRoot && sameStateRoot && sameParentRoot {
			continue
		}
		return true, nil
	}
	return false, nil
}
