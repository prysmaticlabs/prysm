package blockchain

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/config/params"
	butil "github.com/prysmaticlabs/prysm/encoding/bytes"
)

// VerifyWeakSubjectivityRoot verifies the weak subjectivity root in the service struct.
// Reference design: https://github.com/ethereum/consensus-specs/blob/master/specs/phase0/weak-subjectivity.md#weak-subjectivity-sync-procedure
func (s *Service) VerifyWeakSubjectivityRoot(ctx context.Context) error {
	// TODO(7342): Remove the following to fully use weak subjectivity in production.
	if s.cfg.WeakSubjectivityCheckpt == nil || len(s.cfg.WeakSubjectivityCheckpt.Root) == 0 || s.cfg.WeakSubjectivityCheckpt.Epoch == 0 {
		return nil
	}

	// Do nothing if the weak subjectivity has previously been verified,
	// or weak subjectivity epoch is higher than last finalized epoch.
	if s.wsVerified {
		return nil
	}
	if s.cfg.WeakSubjectivityCheckpt.Epoch > s.finalizedCheckpt.Epoch {
		return nil
	}

	r := butil.ToBytes32(s.cfg.WeakSubjectivityCheckpt.Root)
	log.Infof("Performing weak subjectivity check for root %#x in epoch %d", r, s.cfg.WeakSubjectivityCheckpt.Epoch)
	// Save initial sync cached blocks to DB.
	if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
		return err
	}
	// A node should have the weak subjectivity block in the DB.
	if !s.cfg.BeaconDB.HasBlock(ctx, r) {
		return fmt.Errorf("node does not have root in DB: %#x", r)
	}

	startSlot, err := core.StartSlot(s.cfg.WeakSubjectivityCheckpt.Epoch)
	if err != nil {
		return err
	}
	// A node should have the weak subjectivity block corresponds to the correct epoch in the DB.
	filter := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(startSlot + params.BeaconConfig().SlotsPerEpoch)
	roots, err := s.cfg.BeaconDB.BlockRoots(ctx, filter)
	if err != nil {
		return err
	}
	for _, root := range roots {
		if r == root {
			log.Info("Weak subjectivity check has passed")
			s.wsVerified = true
			return nil
		}
	}

	return fmt.Errorf("node does not have root in db corresponding to epoch: %#x %d", r, s.cfg.WeakSubjectivityCheckpt.Epoch)
}
