package kv

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

var historicalStateDeletedKey = []byte("historical-states-deleted")
var archivedSlotsPerPointKey = []byte("slots-per-archived-point")

// HistoricalStatesDeleted verifies historical states exist in DB.
func (kv *Store) HistoricalStatesDeleted(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HistoricalStatesDeleted")
	defer span.End()

	if err := kv.verifySlotsPerArchivePoint(); err != nil {
		return err
	}

	var historicalStateDeleted bool
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(newStateServiceCompatibleBucket)
		v := bkt.Get(historicalStateDeletedKey)
		historicalStateDeleted = len(v) == 1 && v[0] == 0x01
		return nil
	}); err != nil {
		return err
	}

	if historicalStateDeleted {
		log.Warn("Regenerating and saving historical states. This may take a while. Skip this with --skip-regen-historical-states")
		if err := kv.regenHistoricalStates(ctx); err != nil {
			return errors.Wrap(err, "could not regenerate historical states, please retry")
		}
	}

	return kv.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(newStateServiceCompatibleBucket)
		return bkt.Put(historicalStateDeletedKey, []byte{0x00})
	})
}

// This verifies the slots per archived point has not been altered since it's used.
// The node does not allow slots per archived point to alter once it's in operation.
func (kv *Store) verifySlotsPerArchivePoint() error {
	return kv.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(newStateServiceCompatibleBucket)
		v := bkt.Get(archivedSlotsPerPointKey)
		if v == nil {
			if err := bkt.Put(archivedSlotsPerPointKey, bytesutil.Bytes8(params.BeaconConfig().SlotsPerArchivedPoint)); err != nil {
				return err
			}
		} else {
			slotsPerPoint := bytesutil.FromBytes8(v)
			if slotsPerPoint != params.BeaconConfig().SlotsPerArchivedPoint {
				return fmt.Errorf("could not update --slots-per-archive-point after it has been set. Please continue to use %d, or resync from genesis using %d",
					slotsPerPoint, params.BeaconConfig().SlotsPerArchivedPoint)
			}
		}
		return nil
	})
}
