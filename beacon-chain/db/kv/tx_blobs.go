package kv

import (
	"context"
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

// SaveTxBlob by a beacon block root and transaction index.
func (s *Store) SaveTxBlob(
	ctx context.Context, beaconBlockRoot [32]byte, txIndex uint64, blob *ethpb.Blob,
) error {
	_, span := trace.StartSpan(ctx, "BeaconDB.SaveTxBlob")
	defer span.End()
	if blob == nil || len(blob.Blob) == 0 {
		err := errors.New("cannot save nil blob")
		tracing.AnnotateError(span, err)
		return err
	}
	key := txBlobKey(beaconBlockRoot, txIndex)
	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(powchainBucket)
		enc, err := proto.Marshal(blob)
		if err != nil {
			return err
		}
		return bkt.Put(key, enc)
	})
	tracing.AnnotateError(span, err)
	return err
}

// TxBlob retrieves a sharding transaction blob by a beacon block root and transaction index.
func (s *Store) TxBlob(ctx context.Context, beaconBlockRoot [32]byte, txIndex uint64) (*ethpb.Blob, error) {
	_, span := trace.StartSpan(ctx, "BeaconDB.TxBlob")
	defer span.End()

	var blob *ethpb.Blob
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(txBlobsBucket)
		enc := bkt.Get(txBlobKey(beaconBlockRoot, txIndex))
		if len(enc) == 0 {
			return nil
		}
		blob = &ethpb.Blob{}
		return proto.Unmarshal(enc, blob)
	})
	return blob, err
}

func txBlobKey(beaconBlockRoot [32]byte, txIndex uint64) []byte {
	encIndex := make([]byte, 8)
	binary.LittleEndian.PutUint64(encIndex, txIndex)
	return append(beaconBlockRoot[:], encIndex...)
}
