package kv

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	slashertypes "github.com/prysmaticlabs/prysm/slasher/db/types"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

func unmarshalAttSlashing(enc []byte) (*ethpb.AttesterSlashing, error) {
	protoSlashing := &ethpb.AttesterSlashing{}
	err := proto.Unmarshal(enc, protoSlashing)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoSlashing, nil
}

func unmarshalAttSlashings(encoded [][]byte) ([]*ethpb.AttesterSlashing, error) {
	attesterSlashings := make([]*ethpb.AttesterSlashing, len(encoded))
	for i, enc := range encoded {
		ps, err := unmarshalAttSlashing(enc)
		if err != nil {
			return nil, err
		}
		attesterSlashings[i] = ps
	}
	return attesterSlashings, nil
}

// AttesterSlashings accepts a status and returns all slashings with this status.
// returns empty []*ethpb.AttesterSlashing if no slashing has been found with this status.
func (s *Store) AttesterSlashings(ctx context.Context, status slashertypes.SlashingStatus) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.AttesterSlashings")
	defer span.End()
	encoded := make([][]byte, 0)
	err := s.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(slashingBucket).Cursor()
		prefix := encodeType(slashertypes.SlashingType(slashertypes.Attestation))
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			if v[0] == byte(status) {
				encoded = append(encoded, v[1:])
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return unmarshalAttSlashings(encoded)
}

// DeleteAttesterSlashing deletes an attester slashing proof from db.
func (s *Store) DeleteAttesterSlashing(ctx context.Context, attesterSlashing *ethpb.AttesterSlashing) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.deleteAttesterSlashing")
	defer span.End()
	root, err := hashutil.HashProto(attesterSlashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of attesterSlashing")
	}
	return s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(slashingBucket)
		k := encodeTypeRoot(slashertypes.SlashingType(slashertypes.Attestation), root)
		if err != nil {
			return errors.Wrap(err, "failed to get key for for attester slashing.")
		}
		if err := bucket.Delete(k); err != nil {
			return errors.Wrap(err, "failed to delete the slashing proof from slashing bucket")
		}
		return nil
	})
}

// HasAttesterSlashing returns true and slashing status if a slashing is found in the db.
func (s *Store) HasAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) (bool, slashertypes.SlashingStatus, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.HasAttesterSlashing")
	defer span.End()
	var status slashertypes.SlashingStatus
	var found bool
	root, err := hashutil.HashProto(slashing)
	if err != nil {
		return found, status, errors.Wrap(err, "failed to get hash root of attesterSlashing")
	}
	key := encodeTypeRoot(slashertypes.SlashingType(slashertypes.Attestation), root)
	err = s.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		enc := b.Get(key)
		if enc != nil {
			found = true
			status = slashertypes.SlashingStatus(enc[0])
		}
		return nil
	})
	return found, status, err
}

// SaveAttesterSlashing accepts a slashing proof and its status and writes it to disk.
func (s *Store) SaveAttesterSlashing(ctx context.Context, status slashertypes.SlashingStatus, slashing *ethpb.AttesterSlashing) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveAttesterSlashing")
	defer span.End()
	enc, err := proto.Marshal(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	root, err := hashutil.HashProto(slashing)
	if err != nil {
		return err
	}
	key := encodeTypeRoot(slashertypes.SlashingType(slashertypes.Attestation), root)
	return s.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		e := b.Put(key, append([]byte{byte(status)}, enc...))
		return e
	})
}

// SaveAttesterSlashings accepts a slice of slashing proof and its status and writes it to disk.
func (s *Store) SaveAttesterSlashings(ctx context.Context, status slashertypes.SlashingStatus, slashings []*ethpb.AttesterSlashing) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveAttesterSlashings")
	defer span.End()
	enc := make([][]byte, len(slashings))
	key := make([][]byte, len(slashings))
	var err error
	for i, slashing := range slashings {
		enc[i], err = proto.Marshal(slashing)
		if err != nil {
			return errors.Wrap(err, "failed to marshal")
		}
		root, err := hashutil.HashProto(slashing)
		if err != nil {
			return err
		}
		key[i] = encodeTypeRoot(slashertypes.SlashingType(slashertypes.Attestation), root)
	}

	return s.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		for i := 0; i < len(enc); i++ {
			e := b.Put(key[i], append([]byte{byte(status)}, enc[i]...))
			if e != nil {
				return e
			}
		}
		return nil
	})
}

// GetLatestEpochDetected returns the latest detected epoch from db.
func (s *Store) GetLatestEpochDetected(ctx context.Context) (types.Epoch, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.GetLatestEpochDetected")
	defer span.End()
	var epoch types.Epoch
	err := s.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		enc := b.Get([]byte(latestEpochKey))
		if enc == nil {
			epoch = 0
			return nil
		}
		epoch = types.Epoch(bytesutil.FromBytes8(enc))
		return nil
	})
	return epoch, err
}

// SetLatestEpochDetected sets the latest slashing detected epoch in db.
func (s *Store) SetLatestEpochDetected(ctx context.Context, epoch types.Epoch) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SetLatestEpochDetected")
	defer span.End()
	return s.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		err := b.Put([]byte(latestEpochKey), bytesutil.Bytes8(uint64(epoch)))
		return err
	})
}
