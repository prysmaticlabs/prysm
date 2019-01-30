package db

import (
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// SaveDeposit puts the deposit record into the beacon chain db.
func (db *BeaconDB) SaveDeposit(deposit *pb.Deposit) error {
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

// HasDeposit checks if the deposit exists.
func (db *BeaconDB) HasDeposit(hash [32]byte) bool {
	exists := false
	// #nosec G104
	db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(operationsBucket)

		exists = b.Get(hash[:]) != nil
		return nil
	})
	return exists
}
