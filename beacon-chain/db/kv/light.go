package kv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

var (
	lightClientBucket                      = []byte("light")
	lightClientLatestNonFinalizedUpdateKey = []byte("latest-non-finalized")
	lightClientLatestFinalizedUpdateKey    = []byte("latest-finalized")
	lightClientFinalizedCheckpointKey      = []byte("finalized-checkpoint")
)

var (
	ErrNotFound = errors.New("not found")
)

func (s *Store) LightClientBestUpdateForPeriod(
	ctx context.Context, period uint64,
) (*ethpb.LightClientUpdate, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveLightClientFinalizedUpdate")
	defer span.End()
	update := &ethpb.LightClientUpdate{}
	if err := s.db.View(func(tx *bolt.Tx) error {
		lightBkt := tx.Bucket(lightClientBucket)
		updateBytes := lightBkt.Get(bytesutil.Uint64ToBytesBigEndian(period))
		if update == nil {
			return ErrNotFound
		}
		return proto.Unmarshal(updateBytes, update)
	}); err != nil {
		return nil, err
	}
	return update, nil
}

func (s *Store) SaveLightClientBestUpdateForPeriod(
	ctx context.Context, period uint64, update *ethpb.LightClientUpdate,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveLightClientFinalizedUpdate")
	defer span.End()
	enc, err := proto.Marshal(update)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		lightBkt := tx.Bucket(lightClientBucket)
		return lightBkt.Put(bytesutil.Uint64ToBytesBigEndian(period), enc)
	})
}

func (s *Store) LightClientLatestNonFinalizedUpdate(ctx context.Context) (*ethpb.LightClientUpdate, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LightClientFinalizedUpdate")
	defer span.End()
	update := &ethpb.LightClientUpdate{}
	if err := s.db.View(func(tx *bolt.Tx) error {
		lightBkt := tx.Bucket(lightClientBucket)
		updateBytes := lightBkt.Get(lightClientLatestNonFinalizedUpdateKey)
		if update == nil {
			return ErrNotFound
		}
		return proto.Unmarshal(updateBytes, update)
	}); err != nil {
		return nil, err
	}
	return update, nil
}

func (s *Store) SaveLightClientLatestNonFinalizedUpdate(ctx context.Context, update *ethpb.LightClientUpdate) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LightClientFinalizedUpdate")
	defer span.End()
	enc, err := proto.Marshal(update)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		lightBkt := tx.Bucket(lightClientBucket)
		return lightBkt.Put(lightClientLatestNonFinalizedUpdateKey, enc)
	})
}

func (s *Store) LightClientLatestFinalizedUpdate(ctx context.Context) (*ethpb.LightClientUpdate, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LightClientFinalizedUpdate")
	defer span.End()
	update := &ethpb.LightClientUpdate{}
	if err := s.db.View(func(tx *bolt.Tx) error {
		lightBkt := tx.Bucket(lightClientBucket)
		updateBytes := lightBkt.Get(lightClientLatestFinalizedUpdateKey)
		if update == nil {
			return ErrNotFound
		}
		return proto.Unmarshal(updateBytes, update)
	}); err != nil {
		return nil, err
	}
	return update, nil
}

func (s *Store) SaveLightClientLatestFinalizedUpdate(ctx context.Context, update *ethpb.LightClientUpdate) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveLightClientFinalizedUpdate")
	defer span.End()

	enc, err := proto.Marshal(update)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		lightBkt := tx.Bucket(lightClientBucket)
		return lightBkt.Put(lightClientLatestFinalizedUpdateKey, enc)
	})
}

func (s *Store) LightClientFinalizedCheckpoint(ctx context.Context) (*ethpb.Checkpoint, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveLightClientFinalizedCheckpoint")
	defer span.End()
	checkpoint := &ethpb.Checkpoint{}
	if err := s.db.View(func(tx *bolt.Tx) error {
		lightBkt := tx.Bucket(lightClientBucket)
		checkpointBytes := lightBkt.Get(lightClientFinalizedCheckpointKey)
		if checkpointBytes == nil {
			return ErrNotFound
		}
		return proto.Unmarshal(checkpointBytes, checkpoint)
	}); err != nil {
		return nil, err
	}
	return checkpoint, nil
}

func (s *Store) SaveLightClientFinalizedCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveLightClientFinalizedCheckpoint")
	defer span.End()
	enc, err := proto.Marshal(checkpoint)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		lightBkt := tx.Bucket(lightClientBucket)
		return lightBkt.Put(lightClientFinalizedCheckpointKey, enc)
	})
}
