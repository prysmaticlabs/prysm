package main

import (
	"bytes"
	"context"
	"errors"
	"path"
	"time"

	"github.com/boltdb/bolt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
		keys := tx.Bucket(assignedPkBucket).Stats().KeyN
		assignedPkCount.Set(float64(keys))
		keys += tx.Bucket(unassignedPkBucket).Stats().KeyN
		allocatedPkCount.Add(float64(keys))
		return nil
	}); err != nil {
		panic(err)
	}

	return &db{db: boltdb}
}

// UnallocatedPK returns the first unassigned private key, if any are
// available.
func (d *db) UnallocatedPK(_ context.Context) ([]byte, error) {
	var pk []byte
	if err := d.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(unassignedPkBucket).Cursor()
		k, _ := c.First()

		pk = k
		return nil
	}); err != nil {
		return nil, err
	}
	return pk, nil
}

// PodPK returns an assigned private key to the given pod name, if one exists.
func (d *db) PodPK(_ context.Context, podName string) ([]byte, error) {
	var pk []byte
	if err := d.db.View(func(tx *bolt.Tx) error {
		pk = tx.Bucket(assignedPkBucket).Get([]byte(podName))
		return nil
	}); err != nil {
		return nil, err
	}

	return pk, nil
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

// RemovePKAssignment from pod and put the private key into the unassigned
// bucket.
func (d *db) RemovePKAssignment(_ context.Context, podName string) error {
	assignedPkCount.Dec()
	return d.db.Update(func(tx *bolt.Tx) error {
		pk := tx.Bucket(assignedPkBucket).Get([]byte(podName))
		if pk == nil {
			log.WithField("podName", podName).Warn("Nil private key returned from db")
			return nil
		}
		if err := tx.Bucket(assignedPkBucket).Delete([]byte(podName)); err != nil {
			return err
		}
		return tx.Bucket(unassignedPkBucket).Put(pk, dummyVal)
	})
}

// AssignExistingPK assigns a PK from the unassigned bucket to a given pod.
func (d *db) AssignExistingPK(_ context.Context, pk []byte, podName string) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		if !bytes.Equal(tx.Bucket(unassignedPkBucket).Get(pk), dummyVal) {
			return errors.New("private key not in unassigned bucket")
		}
		if err := tx.Bucket(unassignedPkBucket).Delete(pk); err != nil {
			return err
		}
		assignedPkCount.Inc()
		return tx.Bucket(assignedPkBucket).Put([]byte(podName), pk)
	})

	return nil
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
