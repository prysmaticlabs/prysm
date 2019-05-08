package db

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func getRandPubKey(t *testing.T) *bls.PublicKey {
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		t.Fatalf("Don't can create priv key: %v", err)
	}
	return priv.PublicKey()
}

func TestGetBucket_bucketNotExistAndNotCreating(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	err := db.db.View(func(tx *bolt.Tx) error {
		if getBucket(tx, getRandPubKey(t), 1, []byte{2}, false) != nil {
			t.Fatal("getBucket returned the bucket for the nonexistent key")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Don't can do View: %v", err)
	}
}

func TestGetBucket_CreateNotExistBucket(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	err := db.db.Update(func(tx *bolt.Tx) error {
		if getBucket(tx, getRandPubKey(t), 3, []byte{4}, true) == nil {
			t.Fatal("getBucket - no bucket was created")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("can not do View: %v", err)
	}
}

func TestGetBucket_getExistBucket(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	// create Bucket and mark it
	pubKey := getRandPubKey(t)
	err := db.db.Update(func(tx *bolt.Tx) error {
		b := getBucket(tx, pubKey, 5, []byte{6}, true)
		if b == nil {
			t.Fatal("can not create a bucket")
		}
		err := b.Put([]byte("mark"), []byte("mark!!!"))
		return err
	})
	if err != nil {
		t.Fatalf("can not create a basket: %v", err)
	}

	// test getBucket
	err = db.db.View(func(tx *bolt.Tx) error {
		b := getBucket(tx, pubKey, 5, []byte{6}, false)
		if b == nil {
			t.Fatal("bucket not found")
		}
		if !bytes.Equal(b.Get([]byte("mark")), []byte("mark!!!")) {
			t.Fatal("bucket exists, but mark don't found")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Don't can do View: %v", err)
	}
}

func TestLastInAllForks(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	// prepare keys for insert
	pubKeys := []*bls.PublicKey{getRandPubKey(t), getRandPubKey(t), getRandPubKey(t), getRandPubKey(t), getRandPubKey(t)}

	// insert 5 pubKey, 3 fork for each, and 10 item for each pubkey+fork
	// and add for each pubkey in fork number 1 item number 10
	err := db.db.Update(func(tx *bolt.Tx) error {
		for fork := 0; fork < 3; fork++ {
			for _, pubKey := range pubKeys {
				b := getBucket(tx, pubKey, uint64(fork), proposedBlockBucket, true)
				if b == nil {
					t.Fatal("can not create a bucket")
				}

				if fork == 1 {
					err := b.Put(bytesutil.Bytes8(10), []byte{'l', 'a', 's', 't'})
					if err != nil {
						return err
					}
				}
				for j := 0; j < 10; j++ {
					err := b.Put(bytesutil.Bytes8(uint64(j)), bytesutil.Bytes8(uint64(j)))
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("can not create a basket: %v", err)
	}

	for _, pubKey := range pubKeys {
		err = db.lastInAllForks(pubKey, proposedBlockBucket, func(_, foundLast []byte) error {
			if !bytes.Equal(foundLast, []byte{'l', 'a', 's', 't'}) {
				t.Fatalf("found not last item. found %v", foundLast)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("error when call lastInAllForks: %v", err)
		}
	}
}
