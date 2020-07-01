package kv

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/proto/beacon/db"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// SavePowchainData saves the pow chain data.
func (kv *Store) SavePowchainData(ctx context.Context, data *db.ETH1ChainData) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SavePowchainData")
	defer span.End()

	return kv.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(powchainBucket)
		enc, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		return bkt.Put(powchainDataKey, enc)
	})
}

// PowchainData retrieves the powchain data.
func (kv *Store) PowchainData(ctx context.Context) (*db.ETH1ChainData, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PowchainData")
	defer span.End()

	var data *db.ETH1ChainData
	err := kv.db.View(func(tx *bolt.Tx) error {
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
