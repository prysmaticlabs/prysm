package blockchain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
)

var errWSBlockNotFound = errors.New("weak subjectivity root not found in db")
var errWSBlockNotFoundInEpoch = errors.New("weak subjectivity root not found in db within epoch")

type weakSubjectivityDB interface {
	HasBlock(ctx context.Context, blockRoot [32]byte) bool
	BlockRoots(ctx context.Context, f *filters.QueryFilter) ([][32]byte, error)
}

type WeakSubjectivityVerifier struct {
	enabled  bool
	verified bool
	root     [32]byte
	epoch    types.Epoch
	slot     types.Slot
	db       weakSubjectivityDB
}

// validates a checkpoint, and if valid, uses it to initialize a weak subjectivity verifier
func NewWeakSubjectivityVerifier(wsc *ethpb.Checkpoint, db weakSubjectivityDB) (*WeakSubjectivityVerifier, error) {
	// TODO(7342): Remove the following to fully use weak subjectivity in production.
	if wsc == nil || len(wsc.Root) == 0 || wsc.Epoch == 0 {
		return &WeakSubjectivityVerifier{
			enabled: false,
		}, nil
	}
	startSlot, err := slots.EpochStart(wsc.Epoch)
	if err != nil {
		return nil, err
	}
	return &WeakSubjectivityVerifier{
		enabled: true,
		root:    bytesutil.ToBytes32(wsc.Root),
		epoch:   wsc.Epoch,
		db:      db,
		slot:    startSlot,
	}, nil
}

// VerifyWeakSubjectivity verifies the weak subjectivity root in the service struct.
// Reference design: https://github.com/ethereum/consensus-specs/blob/master/specs/phase0/weak-subjectivity.md#weak-subjectivity-sync-procedure
func (v *WeakSubjectivityVerifier) VerifyWeakSubjectivity(ctx context.Context, finalizedEpoch types.Epoch) error {
	if v.verified || !v.enabled {
		return nil
	}
	if v.epoch > finalizedEpoch {
		return nil
	}
	log.Infof("Performing weak subjectivity check for root %#x in epoch %d", v.root, v.epoch)

	// TODO the original code is forcing a sync of init blocks, can we avoid doing that?
	// if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {

	if !v.db.HasBlock(ctx, v.root) {
		return errors.Wrap(errWSBlockNotFound, fmt.Sprintf("missing root %#x", v.root))
	}
	filter := filters.NewFilter().SetStartSlot(v.slot).SetEndSlot(v.slot + params.BeaconConfig().SlotsPerEpoch)
	// A node should have the weak subjectivity block corresponds to the correct epoch in the DB.
	roots, err := v.db.BlockRoots(ctx, filter)
	if err != nil {
		return errors.Wrap(err, "error while retrieving block roots to verify weak subjectivity")
	}
	for _, root := range roots {
		if v.root == root {
			log.Info("Weak subjectivity check has passed")
			v.verified = true
			return nil
		}
	}

	return errors.Wrap(errWSBlockNotFoundInEpoch, fmt.Sprintf("root=%#x, epoch=%d", v.root, v.epoch))
}
