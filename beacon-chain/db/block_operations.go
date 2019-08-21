package db

import (
	"context"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"go.opencensus.io/trace"
)

// ProposerSlashing retrieval from the db.
// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) ProposerSlashing(ctx context.Context, slashingRoot [32]byte) (*ethpb.ProposerSlashing, error) {
	return nil, errors.New("unimplemented")
}

// AttesterSlashing retrieval from the db.
// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) AttesterSlashing(ctx context.Context, slashingRoot [32]byte) (*ethpb.AttesterSlashing, error) {
	return nil, errors.New("unimplemented")
}

// SaveProposerSlashing to the db.
// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) SaveProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) error {
	return errors.New("unimplemented")
}

// SaveAttesterSlashing to the db.
// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) SaveAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) error {
	return errors.New("unimplemented")
}

// HasProposerSlashing by root.
// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) HasProposerSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	return false
}

// HasAttesterSlashing by root.
// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) HasAttesterSlashing(ctx context.Context, slashingRoot [32]byte) bool {
	return false
}

// DeleteProposerSlashing by root.
// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) DeleteProposerSlashing(ctx context.Context, slashingRoot [32]byte) error {
	return errors.New("unimplemented")
}

// DeleteAttesterSlashing by root.
// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) DeleteAttesterSlashing(ctx context.Context, slashingRoot [32]byte) error {
	return errors.New("unimplemented")
}

// SaveExit puts the exit request into the beacon chain db.
func (db *BeaconDB) SaveExit(ctx context.Context, exit *ethpb.VoluntaryExit) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.SaveExit")
	defer span.End()

	hash, err := hashutil.HashProto(exit)
	if err != nil {
		return err
	}
	encodedExit, err := proto.Marshal(exit)
	if err != nil {
		return err
	}
	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(blockOperationsBucket)
		return a.Put(hash[:], encodedExit)
	})
}

// HasExit checks if the exit request exists.
func (db *BeaconDB) HasExit(hash [32]byte) bool {
	exists := false
	if err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(blockOperationsBucket)
		exists = b.Get(hash[:]) != nil
		return nil
	}); err != nil {
		return false
	}
	return exists
}
