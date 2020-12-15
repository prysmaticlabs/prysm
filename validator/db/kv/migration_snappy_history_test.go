package kv

import (
	"bytes"
	"testing"

	"github.com/golang/snappy"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	bolt "go.etcd.io/bbolt"
)

func Test_migrateSnappyAttestationHistory(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, db *bolt.DB)
		eval  func(t *testing.T, db *bolt.DB)
	}{
		{
			name: "only runs once",
			setup: func(t *testing.T, db *bolt.DB) {
				if err := db.Update(func(tx *bolt.Tx) error {
					if err := tx.Bucket(historicAttestationsBucket).Put([]byte("foo"), []byte{1,1,1,1,1,1,1,1,1}); err != nil {
						return err
					}
					return tx.Bucket(migrationsBucket).Put(migrationSnappyAttestationHistory0Key, migrationCompleted)
				}); err != nil {
					t.Fatal(err)
				}
			},
			eval: func(t *testing.T, db *bolt.DB) {
				if err := db.View(func(tx *bolt.Tx) error {
					data := tx.Bucket(historicAttestationsBucket).Get([]byte("foo"))
					expected := []byte{1,1,1,1,1,1,1,1,1} // unchanged
					if !bytes.Equal(data, expected) {
						t.Fatalf("expected data did not match reality. Got %x wanted %x", data, expected)
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "compresses data",
			setup: func(t *testing.T, db *bolt.DB) {
				if err := db.Update(func(tx *bolt.Tx) error {
					return tx.Bucket(historicAttestationsBucket).Put([]byte("foo"), []byte{1,1,1,1,1,1,1,1,1})
				}); err != nil {
					t.Fatal(err)
				}
			},
			eval: func(t *testing.T, db *bolt.DB) {
				if err := db.View(func(tx *bolt.Tx) error {
					data := tx.Bucket(historicAttestationsBucket).Get([]byte("foo"))
					expected := snappy.Encode(nil /*dst*/, []byte{1,1,1,1,1,1,1,1,1})
					if !bytes.Equal(data, expected) {
						t.Fatalf("expected data did not match reality. Got %x wanted %x", data, expected)
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t, nil).db
			tt.setup(t, db)
			assert.NoError(t, db.Update(migrateSnappyAttestationHistory), "migrateSnappyAttestationHistory(tx) error")
			tt.eval(t, db)
		})
	}
}
