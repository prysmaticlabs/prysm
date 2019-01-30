package db

import (
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// SaveDeposit puts the exit request into the beacon chain db.
func (db *BeaconDB) SaveExit(deposit *pb.Exit) error {
	hash, err := hashutil.HashProto(deposit)
	if err != nil {
		return err
	}
	encodedState, err := proto.Marshal(deposit)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(operationsBucket)

		return a.Put(hash[:], encodedState)
	})
}

// HasExit checks if the exit request exists.
func (db *BeaconDB) HasExit(hash [32]byte) bool {
	exists := false
	// #nosec G104
	db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(operationsBucket)

		exists = b.Get(hash[:]) != nil
		return nil
	})
	return exists
}
