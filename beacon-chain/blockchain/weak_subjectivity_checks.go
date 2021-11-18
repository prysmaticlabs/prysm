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

// NewWeakSubjectivityVerifier validates a checkpoint, and if valid, uses it to initialize a weak subjectivity verifier
func NewWeakSubjectivityVerifier(wsc *ethpb.Checkpoint, db weakSubjectivityDB) (*WeakSubjectivityVerifier, error) {
	// TODO(7342): Weak subjectivity checks are currently optional. When we require the flag to be specified
	// per 7342, a nil checkpoint, zero-root or zero-epoch should all fail validation
	// and return an error instead of creating a WeakSubjectivityVerifier that permits any chain history.
	if wsc == nil || len(wsc.Root) == 0 || wsc.Epoch == 0 {
		log.Warn("no valid weak subjectivity checkpoint specified, running without weak subjectivity verification")
		return &WeakSubjectivityVerifier{
			enabled: false,
		}, nil
	}
	startSlot, err := slots.EpochStart(wsc.Epoch)
	if err != nil {
		return nil, err
	}
	return &WeakSubjectivityVerifier{
		enabled:  true,
		verified: false,
		root:     bytesutil.ToBytes32(wsc.Root),
		epoch:    wsc.Epoch,
		db:       db,
		slot:     startSlot,
	}, nil
}

// VerifyWeakSubjectivity verifies the weak subjectivity root in the service struct.
// Reference design: https://github.com/ethereum/consensus-specs/blob/master/specs/phase0/weak-subjectivity.md#weak-subjectivity-sync-procedure
func (v *WeakSubjectivityVerifier) VerifyWeakSubjectivity(ctx context.Context, finalizedEpoch types.Epoch) error {
	if v.verified || !v.enabled {
		return nil
	}

	// Two conditions are described in the specs:
	// IF epoch_number > store.finalized_checkpoint.epoch,
	// then ASSERT during block sync that block with root block_root
	// is in the sync path at epoch epoch_number. Emit descriptive critical error if this assert fails,
	// then exit client process.
	// we do not handle this case ^, because we can only blocks that have been processed / are currently
	// in line for finalization, we don't have the ability to look ahead. so we only satisfy the following:
	// IF epoch_number <= store.finalized_checkpoint.epoch,
	// then ASSERT that the block in the canonical chain at epoch epoch_number has root block_root.
	// Emit descriptive critical error if this assert fails, then exit client process.
	if v.epoch > finalizedEpoch {
		return nil
	}
	log.Infof("Performing weak subjectivity check for root %#x in epoch %d", v.root, v.epoch)

	if !v.db.HasBlock(ctx, v.root) {
		return errors.Wrap(errWSBlockNotFound, fmt.Sprintf("missing root %#x", v.root))
	}
	filter := filters.NewFilter().SetStartSlot(v.slot).SetEndSlot(v.slot + params.BeaconConfig().SlotsPerEpoch)
	// A node should have the weak subjectivity block corresponds to the correct epoch in the DB.
	log.Infof("searching block roots index for weak subjectivity root=%#x", v.root)
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
