package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	bolt "go.etcd.io/bbolt"
)

var stateSizeBucket = []byte("state-sizes")
var bucketStatsBucket = []byte("bucket-stats")

func dbinit(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{stateSizeBucket, bucketStatsBucket} {
			_, err := tx.CreateBucketIfNotExists(b)
			if err != nil {
				return errors.Wrapf(err, "failed to create bucket %s", string(b))
			}
		}
		return nil
	})
}

func writeBucketStat(db *bolt.DB, name string, size int) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketStatsBucket)
		if b == nil {
			return fmt.Errorf("wtf %s is nil", string(bucketStatsBucket))
		}

		k := []byte(name)
		v := make([]byte, 4)
		binary.LittleEndian.PutUint32(v, uint32(size))
		err := b.Put(k, v)
		if err != nil {
			return errors.Wrapf(err, "error Put w/ key=%#x, val=%v", k, v)
		}
		return nil
	})
}

func getBucketStats(db *bolt.DB) (map[string]int, error) {
	s := make(map[string]int)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketStatsBucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			s[string(k)] = int(binary.LittleEndian.Uint32(v))
		}
		return nil
	})

	return s, err
}

func writeSummary(db *bolt.DB, sum SizeSummary) error {
	sumb, err := json.Marshal(sum)
	if err != nil {
		return errors.Wrap(err, "unable to marshal SizeSummary json struct")
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(stateSizeBucket)
		k := bytesutil.SlotToBytesBigEndian(sum.SlotRoot.Slot)
		fmt.Printf("writing key=%#x for slot=%d\n", k, sum.SlotRoot.Slot)
		err := b.Put(k, sumb)
		if err != nil {
			return errors.Wrapf(err, "error Put w/ key=%#x, val=%v", k, sum)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "error writing summary to db - %v", sum)
	}
	return nil
}

func summaryIter(db *bolt.DB) chan SizeSummary {
	ch := make(chan SizeSummary)
	go func() {
		err := db.View(func(tx *bolt.Tx) error {
			defer close(ch)
			b := tx.Bucket(stateSizeBucket)
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				sum := SizeSummary{}
				err := json.Unmarshal(v, &sum)
				if err != nil {
					return err
				}
				ch <- sum
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
	}()
	return ch
}

func summaryDump(db *bolt.DB) error {
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(stateSizeBucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("summary for slot=%d\n", bytesutil.BytesToSlotBigEndian(k))
			sum := SizeSummary{}
			err := json.Unmarshal(v, &sum)
			if err != nil {
				return err
			}
			fmt.Printf("%v\n", sum)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
