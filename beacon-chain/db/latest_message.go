package db

import (
	"context"
	"encoding/binary"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"go.opencensus.io/trace"
)

// SaveLatestMessage puts the validator's latest message record into the beacon chain db.
func (db *BeaconDB) SaveLatestMessage(ctx context.Context, index uint64, latestMsg *pb.LatestMessage) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.SaveLatestMessage")
	defer span.End()

	b := make([]byte, 64)
	binary.LittleEndian.PutUint64(b, uint64(index))

	encodedAtt, err := proto.Marshal(latestMsg)
	if err != nil {
		return err
	}

	return db.batch(func(tx *bolt.Tx) error {
		l := tx.Bucket(latestMessageBucket)
		return l.Put(b, encodedAtt)
	})
}

// LatestMessage retrieves validator's latest message from the db using its index.
func (db *BeaconDB) LatestMessage(index uint64) (*pb.LatestMessage, error) {
	var msg *pb.LatestMessage

	b := make([]byte, 64)
	binary.LittleEndian.PutUint64(b, uint64(index))

	err := db.view(func(tx *bolt.Tx) error {
		l := tx.Bucket(latestMessageBucket)

		enc := l.Get(b)
		if enc == nil {
			return nil
		}

		var err error
		msg, err = createLatestMessage(enc)
		return err
	})

	return msg, err
}

// HasLatestMessage checks if the validator has the last message stored.
func (db *BeaconDB) HasLatestMessage(index uint64) bool {
	b := make([]byte, 64)
	binary.LittleEndian.PutUint64(b, uint64(index))

	exists := false
	if err := db.view(func(tx *bolt.Tx) error {
		l := tx.Bucket(latestMessageBucket)
		exists = l.Get(b) != nil
		return nil
	}); err != nil {
		return false
	}

	return exists
}

func createLatestMessage(enc []byte) (*pb.LatestMessage, error) {
	l := &pb.LatestMessage{}
	if err := proto.Unmarshal(enc, l); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding: %v")
	}
	return l, nil
}
