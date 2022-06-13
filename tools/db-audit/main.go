package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz/detect"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	bolt "go.etcd.io/bbolt"
	"os"
	"time"
)

type flags struct {
	dbFile string
}

func (f *flags) init() {
	flag.StringVar(&f.dbFile, "db", "", "path to beaconchain.db")
	flag.Parse()
}

const (
	initMMapSize = 536870912
)

var buckets = struct{
	state []byte
	blockSlotRoots []byte
	stateSlotRoots []byte
}{
	state: []byte("state"),
	blockSlotRoots: []byte("block-slot-indices"),
	stateSlotRoots: []byte("state-slot-indices"),
}

func main() {
	f := new(flags)
	f.init()
	if err := run(context.Background(), f); err != nil {
		fmt.Printf("Fatal error, %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, f *flags) error {
	db, err := bolt.Open(
		f.dbFile,
		params.BeaconIoConfig().ReadWritePermissions,
		&bolt.Options{
			Timeout:         1 * time.Second,
			InitialMmapSize: initMMapSize,
		},
	)
	if err != nil {
		return err
	}

	store, err := kv.NewKVStoreWithDB(ctx, db)
	if err != nil {
		return err
	}

	for sr := range slotRootIter(db) {
		fmt.Printf("slot=%d, root=%#x", sr.slot, sr.root)
		if sr.slot > 900000 {
			st, err := store.State(ctx, sr.root)
			if err != nil {
				return errors.Wrapf(err, "unable to fetch state for root=%#x", sr.root)
			}
			sb, err := st.MarshalSSZ()
			if err != nil {
				return errors.Wrapf(err, "unable to marshal state w/ root=%#x", sr.root)
			}
			sz, err := computeSizes(sb)
			if err != nil {
				return errors.Wrapf(err, "unable to unmarshal to concrete type and compute sizes, root=%#x", sr.root)
			}
			fmt.Printf(", size=%d", sz.overall)
		}
		fmt.Println("")
	}
	return nil
}

type slotRoot struct {
	slot types.Slot
	root [32]byte
}

func slotRootIter(db *bolt.DB) chan slotRoot {
	ch := make(chan slotRoot)
	go func() {
		err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(buckets.stateSlotRoots)
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				ch <- slotRoot{slot: bytesutil.BytesToSlotBigEndian(k), root: bytesutil.ToBytes32(v)}
			}
			close(ch)
			return nil
		})
		if err != nil {
			panic(err)
		}
	}()
	return ch
}

var errStateNotFound = errors.New("could not look up state by block root")

func getStateBytes(db *bolt.DB, root [32]byte) ([]byte, error) {
	var sb []byte
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(buckets.state)
		sb = b.Get(root[:])
		if len(sb) == 0 {
			return errors.Wrapf(errStateNotFound, "root=%#x", root)
		}
		return nil
	})
	return sb, err
}

type sizes struct {
	overall int
}

func computeSizes(sb []byte) (*sizes, error) {
	vu, err := detect.FromState(sb)
	if err != nil {
		return nil, err
	}
	forkName := version.String(vu.Fork)
	switch vu.Fork {
	case version.Phase0:
		st := &ethpb.BeaconState{}
		if err := st.UnmarshalSSZ(sb); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal state, detected fork=%s", forkName)
		}
		return &sizes{overall: st.SizeSSZ()}, nil
	case version.Altair:
		st := &ethpb.BeaconStateAltair{}
		if err := st.UnmarshalSSZ(sb); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal state, detected fork=%s", forkName)
		}
		return &sizes{overall: st.SizeSSZ()}, nil
	case version.Bellatrix:
		st := &ethpb.BeaconStateBellatrix{}
		if err := st.UnmarshalSSZ(sb); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal state, detected fork=%s", forkName)
		}
		return &sizes{overall: st.SizeSSZ()}, nil
	default:
		return nil, fmt.Errorf("unable to initialize BeaconState for fork version=%s", forkName)
	}
}
