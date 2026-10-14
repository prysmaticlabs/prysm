package kv

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
)

// Vote window is two slots, older reservations are pruned.
const availableVoteRetention = 64

// Index value per key: settled, or pending while the last row hasn't gone out.
var (
	finalitySettled = []byte{1}
	finalityPending = []byte{2}
)

// FinalityHistory returns the rows this key signed, in order. A known key with lost history refuses to sign.
func (s *Store) FinalityHistory(_ context.Context, pubKey [fieldparams.BLSPubkeyLength]byte) (history []*ethpb.AttestationDataDecoupled, pending, known bool, err error) {
	err = s.view(func(tx *bolt.Tx) error {
		mark := tx.Bucket(finalityKeysBucket).Get(pubKey[:])
		known, pending = mark != nil, bytes.Equal(mark, finalityPending)
		b := tx.Bucket(finalityHistoryBucket).Bucket(pubKey[:])
		if b == nil {
			if known {
				return errors.New("finality signing history is missing")
			}
			return nil
		}
		return b.ForEach(func(_, v []byte) error {
			d := &ethpb.AttestationDataDecoupled{}
			if err := d.UnmarshalSSZ(v); err != nil {
				return err
			}
			history = append(history, d)
			return nil
		})
	})
	return history, pending, known, err
}

// MarkFinalityKey registers the key, so lost history is detectable.
func (s *Store) MarkFinalityKey(_ context.Context, pubKey [fieldparams.BLSPubkeyLength]byte) error {
	return s.update(func(tx *bolt.Tx) error {
		if _, err := tx.Bucket(finalityHistoryBucket).CreateBucketIfNotExists(pubKey[:]); err != nil {
			return err
		}
		return tx.Bucket(finalityKeysBucket).Put(pubKey[:], finalitySettled)
	})
}

// SaveFinalityAttestation stores a row before it is signed, pending until submitted.
func (s *Store) SaveFinalityAttestation(_ context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, d *ethpb.AttestationDataDecoupled) error {
	enc, err := d.MarshalSSZ()
	if err != nil {
		return err
	}
	return s.update(func(tx *bolt.Tx) error {
		if err := tx.Bucket(finalityKeysBucket).Put(pubKey[:], finalityPending); err != nil {
			return err
		}
		b, err := tx.Bucket(finalityHistoryBucket).CreateBucketIfNotExists(pubKey[:])
		if err != nil {
			return err
		}
		seq, err := b.NextSequence()
		if err != nil {
			return err
		}
		return b.Put(bytesutil.Uint64ToBytesBigEndian(seq), enc)
	})
}

// MarkFinalitySubmitted settles the key if d is still the last row. A late ack must not clear a newer pending row.
func (s *Store) MarkFinalitySubmitted(_ context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, d *ethpb.AttestationDataDecoupled) error {
	enc, err := d.MarshalSSZ()
	if err != nil {
		return err
	}
	return s.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(finalityHistoryBucket).Bucket(pubKey[:])
		if b == nil {
			return nil
		}
		if _, last := b.Cursor().Last(); !bytes.Equal(last, enc) {
			return nil
		}
		return tx.Bucket(finalityKeysBucket).Put(pubKey[:], finalitySettled)
	})
}

// ReserveAvailableVote allows one head per slot. The same head again is fine.
func (s *Store) ReserveAvailableVote(_ context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, slot primitives.Slot, root [32]byte) error {
	return s.update(func(tx *bolt.Tx) error {
		b, err := tx.Bucket(availableVotesBucket).CreateBucketIfNotExists(pubKey[:])
		if err != nil {
			return err
		}
		key := bytesutil.SlotToBytesBigEndian(slot)
		if have := b.Get(key); have != nil {
			if !bytes.Equal(have, root[:]) {
				return fmt.Errorf("slot %d already reserved for head %#x", slot, have)
			}
			return nil
		}
		if err := b.Put(key, root[:]); err != nil {
			return err
		}
		var stale [][]byte
		c := b.Cursor()
		for k, _ := c.First(); k != nil && bytesutil.BytesToSlotBigEndian(k)+availableVoteRetention < slot; k, _ = c.Next() {
			stale = append(stale, k)
		}
		for _, k := range stale {
			if err := b.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}
