package db

import (
	"context"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
)

type CanonicalChecker interface {
	IsCanonical(ctx context.Context, blockRoot [32]byte) (bool, error)
}

type FinalizedChecker interface {
	IsFinalizedBlock(ctx context.Context, blockRoot [32]byte) bool
}

type canonicalChecker struct {
	fc FinalizedChecker
}

func (cc *canonicalChecker) IsCanonical(ctx context.Context, root [32]byte) (bool, error) {
	return cc.fc.IsFinalizedBlock(ctx, root), nil
}

func NewCanonicalChecker(fc FinalizedChecker) CanonicalChecker {
	return &canonicalChecker{fc: fc}
}

type CurrentSlotter interface {
	CurrentSlot() types.Slot
}

type FinalizedCheckpointer interface {
	FinalizedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error)
}

type finalizedCurrentSlotter struct {
	fc FinalizedCheckpointer
	ctx context.Context
}

func (fc *finalizedCurrentSlotter) CurrentSlot() types.Slot {
	cp, err := fc.fc.FinalizedCheckpoint(fc.ctx)
	if err != nil {
		return 0
	}
	s, err := slots.EpochStart(cp.Epoch)
	if err != nil {
		return 0
	}
	return s
}

func FinalizedCurrentSlotter(fc FinalizedCheckpointer, ctx context.Context) CurrentSlotter {
	return &finalizedCurrentSlotter{fc: fc, ctx: ctx}
}