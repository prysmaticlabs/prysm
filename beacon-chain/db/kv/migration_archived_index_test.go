package kv

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"go.etcd.io/bbolt"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func Test_migrateArchivedIndex(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, db *bbolt.DB)
		eval  func(t *testing.T, db *bbolt.DB)
	}{
		{
			name: "only runs once",
			setup: func(t *testing.T, db *bbolt.DB) {
				if err := db.Update(func(tx *bbolt.Tx) error {
					if _, err := tx.CreateBucketIfNotExists(archivedRootBucket); err != nil {
						t.Error(err)
					}
					if err := tx.Bucket(archivedRootBucket).Put(bytesutil.Uint64ToBytesLittleEndian(2048), []byte("foo")); err != nil {
						return err
					}
					return tx.Bucket(migrationsBucket).Put(migrationArchivedIndex0Key, migrationCompleted)
				}); err != nil {
					t.Error(err)
				}
			},
			eval: func(t *testing.T, db *bbolt.DB) {
				if err := db.View(func(tx *bbolt.Tx) error {
					v := tx.Bucket(archivedRootBucket).Get(bytesutil.Uint64ToBytesLittleEndian(2048))
					if !bytes.Equal(v, []byte("foo")) {
						return fmt.Errorf("did not receive correct data for key 2048, wanted 'foo' got %s", v)
					}
					return nil
				}); err != nil {
					t.Error(err)
				}
			},
		},
		{
			name: "migrates and deletes entries",
			setup: func(t *testing.T, db *bbolt.DB) {
				if err := db.Update(func(tx *bbolt.Tx) error {
					if _, err := tx.CreateBucketIfNotExists(archivedRootBucket); err != nil {
						t.Error(err)
					}
					if _, err := tx.CreateBucketIfNotExists(slotsHasObjectBucket); err != nil {
						t.Error(err)
					}
					if err := tx.Bucket(archivedRootBucket).Put(bytesutil.Uint64ToBytesLittleEndian(2048), []byte("foo")); err != nil {
						return err
					}

					b, err := encode(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2048}})
					if err != nil {
						return err
					}
					return tx.Bucket(blocksBucket).Put([]byte("foo"), b)
				}); err != nil {
					t.Error(err)
				}
			},
			eval: func(t *testing.T, db *bbolt.DB) {
				if err := db.View(func(tx *bbolt.Tx) error {
					k := uint64(2048)
					if v := tx.Bucket(stateSlotIndicesBucket).Get(bytesutil.Uint64ToBytesBigEndian(k)); !bytes.Equal(v, []byte("foo")) {
						return fmt.Errorf("did not receive correct data for key %d, wanted 'foo' got %v", k, v)
					}
					return nil
				}); err != nil {
					t.Error(err)
				}
			},
		},
		{
			name: "deletes old buckets",
			setup: func(t *testing.T, db *bbolt.DB) {
				if err := db.Update(func(tx *bbolt.Tx) error {
					if _, err := tx.CreateBucketIfNotExists(archivedRootBucket); err != nil {
						t.Error(err)
					}
					if _, err := tx.CreateBucketIfNotExists(slotsHasObjectBucket); err != nil {
						t.Error(err)
					}
					return tx.Bucket(slotsHasObjectBucket).Put(savedStateSlotsKey, []byte("foo"))
				}); err != nil {
					t.Error(err)
				}
			},
			eval: func(t *testing.T, db *bbolt.DB) {
				if err := db.View(func(tx *bbolt.Tx) error {
					if tx.Bucket(slotsHasObjectBucket) != nil {
						t.Errorf("Expected %v to be deleted", savedStateSlotsKey)
					}
					if tx.Bucket(archivedRootBucket) != nil {
						t.Errorf("Expected %v to be deleted", savedStateSlotsKey)
					}
					return nil
				}); err != nil {
					t.Error(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t).db
			tt.setup(t, db)
			if err := db.Update(migrateArchivedIndex); err != nil {
				t.Errorf("migrateArchivedIndex(tx) error = %v", err)
			}
			tt.eval(t, db)
		})
	}
}
