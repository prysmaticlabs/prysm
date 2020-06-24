package main

import (
	"bytes"
	"context"
	"errors"
	"path"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/cluster"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
)

var (
	allocatedPkCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "allocated_pk_count",
		Help: "The number of allocated private keys",
	})
	assignedPkCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "assigned_pk_count",
		Help: "The number of private keys currently assigned to alive pods",
	})
	bannedPKCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "banned_pk_count",
		Help: "The number of private keys which have been removed that are of exited validators",
	})
)

var (
	dbFileName         = "pk.db"
	assignedPkBucket   = []byte("assigned_pks")
	unassignedPkBucket = []byte("unassigned_pks")
	deletedKeysBucket  = []byte("deleted_pks")
	dummyVal           = []byte{1}
)

type keyMap struct {
	podName    string
	privateKey []byte
	index      int
}

type db struct {
	db *bolt.DB
}

func newDB(dbPath string) *db {
	datafile := path.Join(dbPath, dbFileName)
	boltdb, err := bolt.Open(datafile, params.BeaconIoConfig().FilePermission, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		panic(err)
	}

	// Initialize buckets
	if err := boltdb.Update(func(tx *bolt.Tx) error {
		for _, bkt := range [][]byte{assignedPkBucket, unassignedPkBucket, deletedKeysBucket} {
			if _, err := tx.CreateBucketIfNotExists(bkt); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}

	// Populate metrics on start.
	if err := boltdb.View(func(tx *bolt.Tx) error {
		// Populate banned key count.
		bannedPKCount.Set(float64(tx.Bucket(deletedKeysBucket).Stats().KeyN))

		keys := 0

		// Iterate over all of the pod assigned keys (one to many).
		c := tx.Bucket(assignedPkBucket).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			pks := &pb.PrivateKeys{}
			if err := proto.Unmarshal(v, pks); err != nil {
				log.WithError(err).Error("Unable to unmarshal private key")
				continue
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

func (d *db) DeleteUnallocatedKey(_ context.Context, privateKey []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket(unassignedPkBucket).Delete(privateKey); err != nil {
			return err
		}
		if err := tx.Bucket(deletedKeysBucket).Put(privateKey, dummyVal); err != nil {
			return err
		}
		bannedPKCount.Inc()
		allocatedPkCount.Dec()
		return nil
	})
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
		pks := &pb.PrivateKeys{}
		if b := tx.Bucket(assignedPkBucket).Get([]byte(podName)); b != nil {
			if err := proto.Unmarshal(b, pks); err != nil {
				return err
			}
		}
		pks.PrivateKeys = append(pks.PrivateKeys, pk.SecretKey.Marshal())
		b, err := proto.Marshal(pks)
		if err != nil {
			return err
		}
		return tx.Bucket(assignedPkBucket).Put(
			[]byte(podName),
			b,
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
			log.WithError(err).Error("Failed to unmarshal pks, deleting from db")
			return tx.Bucket(assignedPkBucket).Delete([]byte(podName))
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
				log.WithError(err).Error("Could not unmarshal private key")
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

func (d *db) KeyMap() ([][]byte, map[[48]byte]keyMap, error) {
	m := make(map[[48]byte]keyMap)
	pubkeys := make([][]byte, 0)
	if err := d.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(assignedPkBucket).ForEach(func(k, v []byte) error {
			pks := &pb.PrivateKeys{}
			if err := proto.Unmarshal(v, pks); err != nil {
				return err
			}
			for i, pk := range pks.PrivateKeys {
				seckey, err := bls.SecretKeyFromBytes(pk)
				if err != nil {
					log.WithError(err).Warn("Could not deserialize secret key... removing")
					return tx.Bucket(assignedPkBucket).Delete(k)
				}

				keytoSet := bytesutil.ToBytes48(seckey.PublicKey().Marshal())
				m[keytoSet] = keyMap{
					podName:    string(k),
					privateKey: pk,
					index:      i,
				}
				pubkeys = append(pubkeys, seckey.PublicKey().Marshal())
			}
			return nil
		})
	}); err != nil {
		// do something
		return nil, nil, err
	}

	return pubkeys, m, nil
}

// RemovePKFromPod and throw it away.
func (d *db) RemovePKFromPod(podName string, key []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		data := tx.Bucket(assignedPkBucket).Get([]byte(podName))
		if data == nil {
			log.WithField("podName", podName).Warn("Nil private key returned from db")
			return nil
		}
		pks := &pb.PrivateKeys{}
		if err := proto.Unmarshal(data, pks); err != nil {
			log.WithError(err).Error("Unable to unmarshal private keys, deleting assignment from db")
			return tx.Bucket(assignedPkBucket).Delete([]byte(podName))
		}
		found := false
		for i, k := range pks.PrivateKeys {
			if bytes.Equal(k, key) {
				found = true
				pks.PrivateKeys = append(pks.PrivateKeys[:i], pks.PrivateKeys[i+1:]...)
				break
			}
		}
		if !found {
			return errors.New("private key not assigned to pod")
		}
		marshaled, err := proto.Marshal(pks)
		if err != nil {
			return err
		}
		bannedPKCount.Inc()
		allocatedPkCount.Dec()
		assignedPkCount.Dec()
		nowBytes, err := time.Now().MarshalBinary()
		if err != nil {
			return err
		}
		if err := tx.Bucket(deletedKeysBucket).Put(key, nowBytes); err != nil {
			return err
		}
		return tx.Bucket(assignedPkBucket).Put([]byte(podName), marshaled)
	})
}
