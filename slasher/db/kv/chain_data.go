package kv

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

// ChainHead retrieves the persisted chain head from the database accordingly.
func (s *Store) ChainHead(ctx context.Context) (*ethpb.ChainHead, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.ChainHead")
	defer span.End()
	var res *ethpb.ChainHead
	if err := s.update(func(tx *bolt.Tx) error {
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
func (s *Store) SaveChainHead(ctx context.Context, head *ethpb.ChainHead) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveChainHead")
	defer span.End()
	enc, err := proto.Marshal(head)
	if err != nil {
		return errors.Wrap(err, "failed to encode chain head")
	}
	return s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(chainDataBucket)
		if err := bucket.Put([]byte(chainHeadKey), enc); err != nil {
			return errors.Wrap(err, "failed to save chain head to s")
		}
		return err
	})
}
