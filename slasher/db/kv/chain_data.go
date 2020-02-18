package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"go.opencensus.io/trace"
)

// ChainHead retrieves the persisted chain head from the database accordingly.
func (db *Store) ChainHead(ctx context.Context) (*ethpb.ChainHead, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.ChainHead")
	defer span.End()
	var res *ethpb.ChainHead
	if err := db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(chainDataBucket)
		enc := bucket.Get([]byte(chainHeadKey))
		if enc == nil {
			return nil
		}
		res = &ethpb.ChainHead{}
		return proto.Unmarshal(enc, res)
	}); err != nil {
		return nil, err
	}
	return res, nil
}

// SaveChainHead accepts a beacon chain head object and persists it to the DB.
func (db *Store) SaveChainHead(ctx context.Context, head *ethpb.ChainHead) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveChainHead")
	defer span.End()
	enc, err := proto.Marshal(head)
	if err != nil {
		return errors.Wrap(err, "failed to encode chain head")
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(chainDataBucket)
		if err := bucket.Put([]byte(chainHeadKey), enc); err != nil {
			return errors.Wrap(err, "failed to save chain head to db")
		}
		return err
	})
}
