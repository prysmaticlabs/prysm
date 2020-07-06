package kv

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.etcd.io/bbolt"
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
					if err := tx.Bucket(archivedIndexRootBucket).Put(bytesutil.Uint64ToBytes(2048), []byte("foo")); err != nil {
						return err
					}
					return tx.Bucket(migrationsBucket).Put(migrationArchivedIndex0Key, migrationCompleted)
				}); err != nil {
					t.Error(err)
				}
			},
			eval: func(t *testing.T, db *bbolt.DB) {
				if err := db.View(func(tx *bbolt.Tx) error {
					v := tx.Bucket(archivedIndexRootBucket).Get(bytesutil.Uint64ToBytes(2048))
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
					return tx.Bucket(archivedIndexRootBucket).Put(bytesutil.Uint64ToBytes(2048), []byte("foo"))
				}); err != nil {
					t.Error(err)
				}
			},
			eval: func(t *testing.T, db *bbolt.DB) {
				if err := db.View(func(tx *bbolt.Tx) error {
					original := uint64(2048)
					k := original / params.BeaconConfig().SlotsPerArchivedPoint
					if v := tx.Bucket(archivedIndexRootBucket).Get(bytesutil.Uint64ToBytes(k)); !bytes.Equal(v, []byte("foo")) {
						return fmt.Errorf("did not receive correct data for key %d, wanted 'foo' got %s", k, v)
					}
					if v := tx.Bucket(archivedIndexRootBucket).Get(bytesutil.Uint64ToBytes(original)); v != nil {
						return fmt.Errorf("expected no data for key %d, got %s", original, v)
					}
					return nil
				}); err != nil {
					t.Error(err)
				}
			},
		},
		{
			name: "deletes bitlist key/value",
			setup: func(t *testing.T, db *bbolt.DB) {
				if err := db.Update(func(tx *bbolt.Tx) error {
					return tx.Bucket(archivedIndexRootBucket).Put(savedStateSlotsKey, []byte("foo"))
				}); err != nil {
					t.Error(err)
				}
			},
			eval: func(t *testing.T, db *bbolt.DB) {
				if err := db.View(func(tx *bbolt.Tx) error {
					if val := tx.Bucket(slotsHasObjectBucket).Get(savedStateSlotsKey); val != nil {
						t.Errorf("Expected %v to be deleted but returned %v", savedStateSlotsKey, val)
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
