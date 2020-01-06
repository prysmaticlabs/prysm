package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/proto/beacon/db"
	"go.opencensus.io/trace"
)

// SavePowchainData saves the pow chain data.
func (k *Store) SavePowchainData(ctx context.Context, data *db.ETH1ChainData) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SavePowchainData")
	defer span.End()

	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(powchainBucket)
		enc, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		return bkt.Put(powchainDataKey, enc)
	})
}

// PowchainData retrieves the powchain data.
func (k *Store) PowchainData(ctx context.Context) (*db.ETH1ChainData, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PowchainData")
	defer span.End()

	var data *db.ETH1ChainData
	err := k.db.View(func(tx *bolt.Tx) error {
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
