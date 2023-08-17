package backfill

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/proto/dbval"
	"google.golang.org/protobuf/proto"
)

// NewStatus correctly initializes a StatusUpdater value with the required database value.
func NewStatus(store BackfillDB) *StatusUpdater {
	return &StatusUpdater{
		store: store,
	}
}

// StatusUpdater provides a way to update and query the status of a backfill process that may be necessary to track when
// a node was initialized via checkpoint sync. With checkpoint sync, there will be a gap in node history from genesis
// until the checkpoint sync origin block. StatusUpdater provides the means to update the value keeping track of the lower
// end of the missing block range via the FillFwd() method, to check whether a Slot is missing from the database
// via the SlotCovered() method, and to see the current StartGap() and EndGap().
type StatusUpdater struct {
	sync.RWMutex
	store       BackfillDB
	genesisSync bool
	status      *dbval.BackfillStatus
}

// SlotCovered determines if the given slot is covered by the current chain history.
// If the slot is <= backfill low slot, or >= backfill high slot, the result is true.
// If the slot is between the backfill low and high slots, the result is false.
func (s *StatusUpdater) SlotCovered(sl primitives.Slot) bool {
	s.RLock()
	defer s.RUnlock()
	// short circuit if the node was synced from genesis
	if s.genesisSync || sl == 0 || s.status.LowSlot <= uint64(sl) {
		return true
	}
	return false
}

// FillBack moves the upper bound of the backfill status to the given slot & root,
// saving the new state to the database and then updating StatusUpdater's in-memory copy with the saved value.
func (s *StatusUpdater) FillBack(ctx context.Context, block blocks.ROBlock) error {
	r := block.Root()
	pr := block.Block().ParentRoot()
	status := s.Status()
	status.LowSlot = uint64(block.Block().Slot())
	status.LowRoot = r[:]
	status.LowParentRoot = pr[:]
	return s.updateStatus(ctx, status)
}

// recover will check to see if the db is from a legacy checkpoint sync and either build a new BackfillStatus
// or label the node as synced from genesis.
func (s *StatusUpdater) recoverLegacy(ctx context.Context) error {
	cpr, err := s.store.OriginCheckpointBlockRoot(ctx)
	if errors.Is(err, db.ErrNotFoundOriginBlockRoot) {
		s.genesisSync = true
		return nil
	}

	cpb, err := s.store.Block(ctx, cpr)
	if err != nil {
		return errors.Wrapf(err, "error retrieving block for origin checkpoint root=%#x", cpr)
	}
	if err := blocks.BeaconBlockIsNil(cpb); err != nil {
		return errors.Wrapf(err, "nil block found for origin checkpoint root=%#x", cpr)
	}
	os := uint64(cpb.Block().Slot())
	lpr := cpb.Block().ParentRoot()
	bs := &dbval.BackfillStatus{
		LowSlot:       os,
		LowRoot:       cpr[:],
		LowParentRoot: lpr[:],
		OriginSlot:    os,
		OriginRoot:    cpr[:],
	}
	return s.updateStatus(ctx, bs)
}

// Reload queries the database for backfill status, initializing the internal data and validating the database state.
func (s *StatusUpdater) Reload(ctx context.Context) error {
	status, err := s.store.BackfillStatus(ctx)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return s.recoverLegacy(ctx)
		}
	}
	return s.updateStatus(ctx, status)
}

func (s *StatusUpdater) updateStatus(ctx context.Context, bs *dbval.BackfillStatus) error {
	s.Lock()
	defer s.Unlock()
	if proto.Equal(s.status, bs) {
		return nil
	}
	if err := s.store.SaveBackfillStatus(ctx, bs); err != nil {
		return err
	}

	s.status = bs
	return nil
}

// originState looks up the state for the checkpoint sync origin. This is a hack, because StatusUpdater is the only
// thing that needs db access and it has the origin root handy, so it's convenient to look it up here. The state is
// needed by the verifier.
func (s *StatusUpdater) originState(ctx context.Context) (state.BeaconState, error) {
	return s.store.StateOrError(ctx, bytesutil.ToBytes32(s.Status().OriginRoot))
}

func (s *StatusUpdater) Status() *dbval.BackfillStatus {
	s.RLock()
	defer s.RUnlock()
	return proto.Clone(s.status).(*dbval.BackfillStatus)
}

// BackfillDB describes the set of DB methods that the StatusUpdater type needs to function.
type BackfillDB interface {
	SaveBackfillStatus(context.Context, *dbval.BackfillStatus) error
	BackfillStatus(context.Context) (*dbval.BackfillStatus, error)
	OriginCheckpointBlockRoot(context.Context) ([32]byte, error)
	Block(context.Context, [32]byte) (interfaces.ReadOnlySignedBeaconBlock, error)
	SaveBlock(ctx context.Context, signed interfaces.ReadOnlySignedBeaconBlock) error
	GenesisBlockRoot(context.Context) ([32]byte, error)
	StateOrError(ctx context.Context, blockRoot [32]byte) (state.BeaconState, error)
}
