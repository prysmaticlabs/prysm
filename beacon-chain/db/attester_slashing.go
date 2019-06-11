package db

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"go.opencensus.io/trace"
)

// SaveAttesterSlashing puts a attester slashing request into the beacon chain db.
func (db *BeaconDB) SaveAttesterSlashing(ctx context.Context, slashing *pb.AttesterSlashing) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.SaveAttesterSlashing")
	defer span.End()

	hash, err := hashutil.HashProto(slashing)
	if err != nil {
		return err
	}
	encodedAttesterSlashing, err := proto.Marshal(slashing)
	if err != nil {
		return err
	}
	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(blockOperationsBucket)
		return a.Put(hash[:], encodedAttesterSlashing)
	})
}

// HasAttesterSlashing checks if a attester slashing request exists.
func (db *BeaconDB) HasAttesterSlashing(hash [32]byte) bool {
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
