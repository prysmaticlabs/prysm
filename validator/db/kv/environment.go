package kv

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// GenesisValidatorRoot returns the genesis validator root or nil if the db doesnt contain its value.
func (store *Store) GenesisValidatorRoot(ctx context.Context) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.ProposalHistoryForSlot")
	defer span.End()

	var err error
	var genesisValidatorRoot []byte
	err = store.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(environment)
		gvr := bucket.Get([]byte(genesisValidatorRootKey))
		if len(gvr) == 0 {
			genesisValidatorRoot = nil
			return nil
		}
		copy(genesisValidatorRoot, gvr)
		return nil
	})
	return genesisValidatorRoot, err
}

// SaveGenesisValidatorRoot saves the genesis validator root if there is no value set for it in db already.
// returns error if genesis validator root already exists in db.
func (store *Store) SaveGenesisValidatorRoot(
	ctx context.Context,
	genesisValidatorRoot []byte,
) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveGenesisValidatorRoot")
	defer span.End()

	return store.update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(environment)
		if err != nil {
			return err
		}
		if gvr := bucket.Get([]byte(genesisValidatorRootKey)); len(gvr) != 0 && !bytes.Equal(gvr, genesisValidatorRoot) {
			log.Fatalf("Attempt to change genesis validator root data in db violated the slashing protection scheme. wanted: %#x got: %#x", gvr, genesisValidatorRoot)
			return errors.New("Attempt to change genesis validator root data in db violated the slashing protection scheme")
		}
		if err := bucket.Put([]byte(genesisValidatorRootKey), genesisValidatorRoot); err != nil {
			return errors.Wrap(err, "failed to set genesis validator root")
		}
		return nil
	})
}
