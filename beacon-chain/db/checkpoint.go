package db

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"go.opencensus.io/trace"
)


// SaveCheckpoint puts the checkpoint record into the beacon chain db.
func (db *BeaconDB) SaveCheckpoint(ctx context.Context, checkpoint *pb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.SaveCheckpoint")
	defer span.End()

	encodedAtt, err := proto.Marshal(checkpoint)
	if err != nil {
		return err
	}
	hash := hashutil.Hash(encodedAtt)

	return db.batch(func(tx *bolt.Tx) error {
		a := tx.Bucket(checkpointBucket)

		return a.Put(hash[:], encodedAtt)
	})
}
