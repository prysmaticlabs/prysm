package db

import (
"context"

"github.com/boltdb/bolt"
"github.com/gogo/protobuf/proto"
pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
"github.com/prysmaticlabs/prysm/shared/hashutil"
"go.opencensus.io/trace"
)

// SaveProposerSlashing puts a proposer slashing request into the beacon chain db.
func (db *BeaconDB) SaveProposerSlashing(ctx context.Context, slashing *pb.ProposerSlashing) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.SaveProposerSlashing")
	defer span.End()

	hash, err := hashutil.HashProto(slashing)
	if err != nil {
		return err
	}
	encodedProposerSlashing, err := proto.Marshal(slashing)
	if err != nil {
		return err
	}
	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(blockOperationsBucket)
		return a.Put(hash[:], encodedProposerSlashing)
	})
}

// HasProposerSlashing checks if a proposer slashing request exists.
func (db *BeaconDB) HasProposerSlashing(hash [32]byte) bool {
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
