package kv

import (
	"context"
	"errors"

	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	v2 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

// SavePowchainData saves the pow chain data.
func (s *Store) SavePowchainData(ctx context.Context, data *v2.ETH1ChainData) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.SavePowchainData")
	defer span.End()

	if data == nil {
		err := errors.New("cannot save nil eth1data")
		tracing.AnnotateError(span, err)
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
	tracing.AnnotateError(span, err)
	return err
}

// PowchainData retrieves the powchain data.
func (s *Store) PowchainData(ctx context.Context) (*v2.ETH1ChainData, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.PowchainData")
	defer span.End()

	var data *v2.ETH1ChainData
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(powchainBucket)
		enc := bkt.Get(powchainDataKey)
		if len(enc) == 0 {
			return nil
		}
		data = &v2.ETH1ChainData{}
		return proto.Unmarshal(enc, data)
	})
	return data, err
}
