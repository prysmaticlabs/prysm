package main

import (
	"bytes"
	"context"
	"path"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/cluster"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/keystore"
)

var (
	allocatedPkCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "allocated_pk_count",
		Help: "The number of allocated private keys",
	})
	assignedPkCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "assigned_pk_count",
		Help: "The number of private keys currently assigned to alive pods",
	})
)

var (
	dbFileName         = "pk.db"
	assignedPkBucket   = []byte("assigned_pks")
	unassignedPkBucket = []byte("unassigned_pks")
	dummyVal           = []byte{1}
)

type db struct {
	db *bolt.DB
}

func newDB(dbPath string) *db {
	datafile := path.Join(dbPath, dbFileName)
	boltdb, err := bolt.Open(datafile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		panic(err)
	}

	// Initialize buckets
	if err := boltdb.Update(func(tx *bolt.Tx) error {
		for _, bkt := range [][]byte{assignedPkBucket, unassignedPkBucket} {
			if _, err := tx.CreateBucketIfNotExists(bkt); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}

	if err := boltdb.View(func(tx *bolt.Tx) error {
		keys := 0

		// Iterate over all of the pod assigned keys (one to many).
		c := tx.Bucket(assignedPkBucket).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			pks := &pb.PrivateKeys{}
			if err := proto.Unmarshal(v, pks); err != nil {
				return err
			}
			keys += len(pks.PrivateKeys)
		}
		assignedPkCount.Set(float64(keys))

		// Add the unassigned keys count (one to one).
		keys += tx.Bucket(unassignedPkBucket).Stats().KeyN
		allocatedPkCount.Add(float64(keys))
		return nil
	}); err != nil {
		panic(err)
	}

	return &db{db: boltdb}
}

// UnallocatedPKs returns unassigned private keys, if any are available.
func (d *db) UnallocatedPKs(_ context.Context, numKeys uint64) (*pb.PrivateKeys, error) {
	pks := &pb.PrivateKeys{}
	if err := d.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(unassignedPkBucket).Cursor()
		i := uint64(0)
		for k, _ := c.First(); k != nil && i < numKeys; k, _ = c.Next() {
			pks.PrivateKeys = append(pks.PrivateKeys, k)
			i++
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return pks, nil
}

// PodPK returns an assigned private key to the given pod name, if one exists.
func (d *db) PodPKs(_ context.Context, podName string) (*pb.PrivateKeys, error) {
	pks := &pb.PrivateKeys{}
	if err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(assignedPkBucket).Get([]byte(podName))

		return proto.Unmarshal(b, pks)
	}); err != nil {
		return nil, err
	}

	return pks, nil
}

// AllocateNewPkToPod records new private key assignment in DB.
func (d *db) AllocateNewPkToPod(
	_ context.Context,
	pk *keystore.Key,
	podName string,
) error {
	allocatedPkCount.Inc()
	assignedPkCount.Inc()
	return d.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(assignedPkBucket).Put(
			[]byte(podName),
			pk.SecretKey.Marshal(),
		)
	})
}

// RemovePKAssignments from pod and put the private keys into the unassigned
// bucket.
func (d *db) RemovePKAssignment(_ context.Context, podName string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		data := tx.Bucket(assignedPkBucket).Get([]byte(podName))
		if data == nil {
			log.WithField("podName", podName).Warn("Nil private key returned from db")
			return nil
		}
		pks := &pb.PrivateKeys{}
		if err := proto.Unmarshal(data, pks); err != nil {
			return err
		}
		if err := tx.Bucket(assignedPkBucket).Delete([]byte(podName)); err != nil {
			return err
		}
		assignedPkCount.Sub(float64(len(pks.PrivateKeys)))
		for _, pk := range pks.PrivateKeys {
			if err := tx.Bucket(unassignedPkBucket).Put(pk, dummyVal); err != nil {
				return err
			}
		}
		return nil
	})
}

// AssignExistingPKs assigns a PK from the unassigned bucket to a given pod.
func (d *db) AssignExistingPKs(_ context.Context, pks *pb.PrivateKeys, podName string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		for _, pk := range pks.PrivateKeys {
			if bytes.Equal(tx.Bucket(unassignedPkBucket).Get(pk), dummyVal) {
				if err := tx.Bucket(unassignedPkBucket).Delete(pk); err != nil {
					return err
				}
			}
		}
		assignedPkCount.Add(float64(len(pks.PrivateKeys)))

		// If pod assignment exists, append to it.
		if existing := tx.Bucket(assignedPkBucket).Get([]byte(podName)); existing != nil {
			existingKeys := &pb.PrivateKeys{}
			if err := proto.Unmarshal(existing, existingKeys); err != nil {
				pks.PrivateKeys = append(pks.PrivateKeys, existingKeys.PrivateKeys...)
			}
		}

		data, err := proto.Marshal(pks)
		if err != nil {
			return err
		}
		return tx.Bucket(assignedPkBucket).Put([]byte(podName), data)
	})
}

// AllocatedPodNames returns the string list of pod names with current private
// key allocations.
func (d *db) AllocatedPodNames(_ context.Context) ([]string, error) {
	var podNames []string
	if err := d.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(assignedPkBucket).ForEach(func(k, v []byte) error {
			podNames = append(podNames, string(k))
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return podNames, nil
}

func (d *db) Allocations() (map[string][][]byte, error) {
	m := make(map[string][][]byte)
	if err := d.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(assignedPkBucket).ForEach(func(k, v []byte) error {
			pks := &pb.PrivateKeys{}
			if err := proto.Unmarshal(v, pks); err != nil {
				return err
			}
			pubkeys := make([][]byte, len(pks.PrivateKeys))
			for i, pk := range pks.PrivateKeys {
				k, err := bls.SecretKeyFromBytes(pk)
				if err != nil {
					return err
				}

				pubkeys[i] = k.PublicKey().Marshal()
			}
			m[string(k)] = pubkeys

			return nil
		})
	}); err != nil {
		// do something
		return nil, err
	}

	return m, nil
}
