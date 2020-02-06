package blockchain

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"go.opencensus.io/trace"
)

// This gets head from the fork choice service and saves head related items
// (ie root, block, state) to the local service cache.
func (s *Service) updateHead(ctx context.Context, balances []uint64) error {
	ctx, span := trace.StartSpan(ctx, "blockchain.updateHead")
	defer span.End()

	// To get the proper head update, a node first checks its best justified
	// can become justified. This is designed to prevent bounce attack and
	// ensure head gets its best justified info.
	if s.bestJustifiedCheckpt.Epoch > s.justifiedCheckpt.Epoch {
		s.justifiedCheckpt = s.bestJustifiedCheckpt
	}

	// Get head from the fork choice service.
	f := s.finalizedCheckpt
	j := s.justifiedCheckpt
	headRoot, err := s.forkChoiceStore.Head(ctx, j.Epoch, bytesutil.ToBytes32(j.Root), balances, f.Epoch)
	if err != nil {
		return err
	}

	// Save head to the local service cache.
	return s.saveHead(ctx, headRoot)
}

// This saves head info to the local service cache, it also saves head
// to the DB.
func (s *Service) saveHead(ctx context.Context, headRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockchain.saveHead")
	defer span.End()

	cachedHeadRoot, err := s.HeadRoot(ctx)
	if err != nil {
		return err
	}
	// Do nothing if head hasn't changed.
	if headRoot == bytesutil.ToBytes32(cachedHeadRoot) {
		return nil
	}

	s.headLock.Lock()
	defer s.headLock.Unlock()

	// Get the new head block from DB.
	newHead, err := s.beaconDB.Block(ctx, headRoot)
	if err != nil {
		return err
	}
	if newHead == nil || newHead.Block == nil {
		return errors.New("cannot save nil head block")
	}

	// Get the new head state from DB.
	headState, err := s.beaconDB.State(ctx, headRoot)
	if err != nil {
		return errors.Wrap(err, "could not retrieve head state in DB")
	}

	// Cache the new head info.
	s.headSlot = newHead.Block.Slot
	s.canonicalRoots[newHead.Block.Slot] = headRoot[:]
	s.headBlock = proto.Clone(newHead).(*ethpb.SignedBeaconBlock)
	s.headState = headState

	// Save the new head root to DB.
	if err := s.beaconDB.SaveHeadBlockRoot(ctx, headRoot); err != nil {
		return errors.Wrap(err, "could not save head root in DB")
	}

	return nil
}
