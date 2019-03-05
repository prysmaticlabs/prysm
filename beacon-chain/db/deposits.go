package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// DepositState fetches the deposits state from POW chain from the DB.
func (db *BeaconDB) DepositState() (*pb.POWDepositState, error) {
	var powDepositState *pb.POWDepositState
	err := db.view(func(tx *bolt.Tx) error {
		powDepInfo := tx.Bucket(depositBucket)
		enc := powDepInfo.Get(stateLookupKey)
		if enc == nil {
			return nil
		}

		var err error
		powDepositState, err = createDepositState(enc)
		return err
	})

	return powDepositState, err
}

// SaveDepositState State updates the POW chain deposits state.
func (db *BeaconDB) SaveDepositState(powDepositState *pb.POWDepositState) error {
	return db.update(func(tx *bolt.Tx) error {
		powDepInfo := tx.Bucket(depositBucket)
		powDepositStateEnc, err := proto.Marshal(powDepositState)
		if err != nil {
			return err
		}
		return powDepInfo.Put(stateLookupKey, powDepositStateEnc)
	})
}

// ResetDepositState resets the last deposit state from db in order to re read the powchain deposit logs
func (db *BeaconDB) ResetDepositState() error {
	return db.SaveDepositState(&pb.POWDepositState{})
}

func createDepositState(enc []byte) (*pb.POWDepositState, error) {
	protoDepositState := &pb.POWDepositState{}
	err := proto.Unmarshal(enc, protoDepositState)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}
	return protoDepositState, nil
}
