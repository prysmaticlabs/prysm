package kv

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

// SavePowchainData saves the pow chain data.
func (s *Store) SavePowchainData(ctx context.Context, data *db.ETH1ChainData) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SavePowchainData")
	defer span.End()

	if data == nil {
		err := errors.New("cannot save nil eth1data")
		traceutil.AnnotateError(span, err)
		return err
	}

	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(powchainBucket)
		enc, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		return bkt.Put(powchainDataKey, enc)
	})
	traceutil.AnnotateError(span, err)
	return err
}

// PowchainData retrieves the powchain data.
func (s *Store) PowchainData(ctx context.Context) (*db.ETH1ChainData, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PowchainData")
	defer span.End()

	var data *db.ETH1ChainData
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(powchainBucket)
		enc := bkt.Get(powchainDataKey)
		if len(enc) == 0 {
			return nil
		}
		data = &db.ETH1ChainData{}
		return proto.Unmarshal(enc, data)
	})
	return data, err
}
