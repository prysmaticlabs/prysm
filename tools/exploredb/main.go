/**
 * Explore DB contents
 *
 * Given a beacon-chain DB, This tool provides many option to
 * inspect and explore it. For every non-empty bucket, print
 * the number of rows, bucket size,min/average/max size of values
 */

package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	log "github.com/sirupsen/logrus"
	"github.com/status-im/keycard-go/hexutils"
	bolt "go.etcd.io/bbolt"
)

var (
	datadir        = flag.String("datadir", "", "Path to data directory.")
	dbName         = flag.String("dbname", "", "database name.")
	bucketStats    = flag.Bool("bucket-stats", false, "Show all the bucket stats.")
	bucketContents = flag.Bool("bucket-contents", false, "Show contents of a given bucket.")
	bucketName     = flag.String("bucket-name", "", "bucket to show contents.")
	rowLimit       = flag.Uint64("limit", 10, "limit to rows.")
)

// used to parallelize all the bucket stats
type bucketStat struct {
	bucketName     string
	noOfRows       uint64
	totalKeySize   uint64
	totalValueSize uint64
	minKeySize     uint64
	maxKeySize     uint64
	minValueSize   uint64
	maxValueSize   uint64
}

// used to parallelize state bucket processing
type modifiedState struct {
	state     iface.BeaconState
	key       []byte
	valueSize uint64
	rowCount  uint64
}

// used to parallelize state summary bucket processing
type modifiedStateSummary struct {
	slot      types.Slot
	root      []byte
	key       []byte
	valueSize uint64
	rowCount  uint64
}

func main() {
	flag.Parse()

	// Check for the mandatory flags.
	if *datadir == "" {
		log.Fatal("Please specify --datadir <db path> to read the database")
	}
	if *dbName == "" {
		log.Fatal("Please specify --dbname <db file name> to specify the database file.")
	}

	// check if the database file is present.
	dbNameWithPath := filepath.Join(*datadir, *dbName)
	if _, err := os.Stat(*datadir); os.IsNotExist(err) {
		log.Fatalf("could not locate database file : %s, %v", dbNameWithPath, err)
	}

	// show stats of all the buckets.
	if *bucketStats {
		printBucketStats(dbNameWithPath)
		return
	}

	// show teh contents of the specified bucket.
	if *bucketContents {
		switch *bucketName {
		case "state", "state-summary":
			printBucketContents(dbNameWithPath, *rowLimit, *bucketName)
		default:
			log.Fatal("Oops, Only 'state' and 'state-summary' buckets are supported for now.")
		}
	}
}

func printBucketStats(dbNameWithPath string) {
	ctx := context.Background()
	groupSize := uint64(128)
	doneC := make(chan bool)
	statsC := make(chan *bucketStat, groupSize)
	go readBucketStats(ctx, dbNameWithPath, statsC)
	go printBucketStates(statsC, doneC)
	<-doneC
}

func printBucketContents(dbNameWithPath string, rowLimit uint64, bucketName string) {
	// get the keys within the supplied limit for the given bucket.
	bucketNameInBytes := []byte(bucketName)
	keys, sizes := keysOfBucket(dbNameWithPath, bucketNameInBytes, rowLimit)

	// create a new KV Store.
	dbDirectory := filepath.Dir(dbNameWithPath)
	db, openErr := kv.NewKVStore(context.Background(), dbDirectory, &kv.Config{})
	if openErr != nil {
		log.Fatalf("could not open db, %v", openErr)
	}

	// don't forget to close it when ejecting out of this function.
	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			log.Fatalf("could not close db, %v", closeErr)
		}
	}()

	// retrieve every element for keys in the list and call the respective display function.
	ctx := context.Background()
	groupSize := uint64(128)
	doneC := make(chan bool)
	switch bucketName {
	case "state":
		stateC := make(chan *modifiedState, groupSize)
		go readStates(ctx, db, stateC, keys, sizes)
		go printStates(stateC, doneC)

	case "state-summary":
		stateSummaryC := make(chan *modifiedStateSummary, groupSize)
		go readStateSummary(ctx, db, stateSummaryC, keys, sizes)
		go printStateSummary(stateSummaryC, doneC)
	}
	<-doneC
}

func readBucketStats(ctx context.Context, dbNameWithPath string, statsC chan<- *bucketStat) {
	// open the raw database file. If the file is busy, then exit.
	db, openErr := bolt.Open(dbNameWithPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if openErr != nil {
		log.Fatalf("could not open db to show bucket stats, %v", openErr)
	}

	// make sure we close the database before ejecting out of this function.
	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			log.Fatalf("could not close db after showing bucket stats, %v", closeErr)
		}
	}()

	// get a list of all the existing buckets.
	var buckets []string
	if viewErr1 := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, buc *bolt.Bucket) error {
			buckets = append(buckets, string(name))
			return nil
		})
	}); viewErr1 != nil {
		log.Fatalf("could not read buckets from db while getting list of buckets: %v", viewErr1)
	}

	// for every bucket, calculate the stats and send it for printing.
	// calculate the state of all the buckets in parallel.
	var wg sync.WaitGroup
	for _, bName := range buckets {
		wg.Add(1)
		go func(bukName string) {
			defer wg.Done()
			count := uint64(0)
			minValueSize := ^uint64(0)
			maxValueSize := uint64(0)
			totalValueSize := uint64(0)
			minKeySize := ^uint64(0)
			maxKeySize := uint64(0)
			totalKeySize := uint64(0)
			if viewErr2 := db.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(bukName))
				if forEachErr := b.ForEach(func(k, v []byte) error {
					count++
					valueSize := uint64(len(v))
					if valueSize < minValueSize {
						minValueSize = valueSize
					}
					if valueSize > maxValueSize {
						maxValueSize = valueSize
					}
					totalValueSize += valueSize

					keyize := uint64(len(k))
					if keyize < minKeySize {
						minKeySize = keyize
					}
					if keyize > maxKeySize {
						maxKeySize = keyize
					}
					totalKeySize += uint64(len(k))
					return nil
				}); forEachErr != nil {
					log.WithError(forEachErr).Errorf("could not process row %d for bucket: %s", count, bukName)
					return forEachErr
				}
				return nil
			}); viewErr2 != nil {
				log.WithError(viewErr2).Errorf("could not get stats for bucket: %s", bukName)
				return
			}
			stat := &bucketStat{
				bucketName:     bukName,
				noOfRows:       count,
				totalKeySize:   totalKeySize,
				totalValueSize: totalValueSize,
				minKeySize:     minKeySize,
				maxKeySize:     maxKeySize,
				minValueSize:   minValueSize,
				maxValueSize:   maxValueSize,
			}
			statsC <- stat
		}(bName)
	}
	wg.Wait()
	close(statsC)
}

func readStates(ctx context.Context, db *kv.Store, stateC chan<- *modifiedState, keys [][]byte, sizes []uint64) {
	for rowCount, key := range keys {
		st, stateErr := db.State(ctx, bytesutil.ToBytes32(key))
		if stateErr != nil {
			log.WithError(stateErr).Errorf("could not get state for key : %s", hexutils.BytesToHex(key))
			continue
		}
		mst := &modifiedState{
			state:     st,
			key:       key,
			valueSize: sizes[rowCount],
			rowCount:  uint64(rowCount),
		}
		stateC <- mst
	}
	close(stateC)
}

func readStateSummary(ctx context.Context, db *kv.Store, stateSummaryC chan<- *modifiedStateSummary, keys [][]byte, sizes []uint64) {
	for rowCount, key := range keys {
		ss, ssErr := db.StateSummary(ctx, bytesutil.ToBytes32(key))
		if ssErr != nil {
			log.WithError(ssErr).Errorf("could not get state summary for key : %s", hexutils.BytesToHex(key))
			continue
		}
		mst := &modifiedStateSummary{
			slot:      ss.Slot,
			root:      ss.Root,
			key:       key,
			valueSize: sizes[rowCount],
			rowCount:  uint64(rowCount),
		}
		stateSummaryC <- mst
	}
	close(stateSummaryC)
}

func printBucketStates(statsC <-chan *bucketStat, doneC chan<- bool) {
	for stat := range statsC {
		if stat.noOfRows != 0 {
			averageValueSize := stat.totalValueSize / stat.noOfRows
			averageKeySize := stat.totalKeySize / stat.noOfRows
			log.Infof("------ %s ---------", stat.bucketName)
			log.Infof("NumberOfRows     = %d", stat.noOfRows)
			log.Infof("TotalBucketSize  =  %s", humanize.Bytes(stat.totalValueSize+stat.totalKeySize))
			log.Infof("KeySize          =  %s, (min = %s, avg = %s, max = %s)",
				humanize.Bytes(stat.totalKeySize),
				humanize.Bytes(stat.minKeySize),
				humanize.Bytes(averageKeySize),
				humanize.Bytes(stat.maxKeySize))
			log.Infof("ValueSize        = %s, (min = %s, avg = %s, max = %s)",
				humanize.Bytes(stat.totalValueSize),
				humanize.Bytes(stat.minValueSize),
				humanize.Bytes(averageValueSize),
				humanize.Bytes(stat.maxValueSize))
		}
	}
	doneC <- true
}

func printStates(stateC <-chan *modifiedState, doneC chan<- bool) {
	for mst := range stateC {
		st := mst.state
		log.Infof("---- row = %04d ----", mst.rowCount)
		log.Infof("key                           : %s", hexutils.BytesToHex(mst.key))
		log.Infof("value                         : compressed size = %s", humanize.Bytes(mst.valueSize))
		t := time.Unix(int64(st.GenesisTime()), 0)
		log.Infof("genesis_time                  : %s", t.Format(time.UnixDate))
		log.Infof("genesis_validators_root       : %s", hexutils.BytesToHex(st.GenesisValidatorRoot()))
		log.Infof("slot                          : %d", st.Slot())
		log.Infof("fork                          : previous_version = %b,  current_version = %b", st.Fork().PreviousVersion, st.Fork().CurrentVersion)
		log.Infof("latest_block_header           : sizeSSZ = %s", humanize.Bytes(uint64(st.LatestBlockHeader().SizeSSZ())))
		size, count := sizeAndCountOfByteList(st.BlockRoots())
		log.Infof("block_roots                   : size = %s, count =  %d", humanize.Bytes(size), count)
		size, count = sizeAndCountOfByteList(st.StateRoots())
		log.Infof("state_roots                   : size = %s, count = %d", humanize.Bytes(size), count)
		size, count = sizeAndCountOfByteList(st.HistoricalRoots())
		log.Infof("historical_roots              : size = %s, count = %d", humanize.Bytes(size), count)
		log.Infof("eth1_data                     : sizeSSZ = %s", humanize.Bytes(uint64(st.Eth1Data().SizeSSZ())))
		size, count = sizeAndCountGeneric(st.Eth1DataVotes(), nil)
		log.Infof("eth1_data_votes               : sizeSSZ = %s, count = %d", humanize.Bytes(size), count)
		log.Infof("eth1_deposit_index            : %d", st.Eth1DepositIndex())
		size, count = sizeAndCountGeneric(st.Validators(), nil)
		log.Infof("validators                    : sizeSSZ = %s, count = %d", humanize.Bytes(size), count)
		size, count = sizeAndCountOfUin64List(st.Balances())
		log.Infof("balances                      : size = %s, count = %d", humanize.Bytes(size), count)
		size, count = sizeAndCountOfByteList(st.RandaoMixes())
		log.Infof("randao_mixes                  : size = %s, count = %d", humanize.Bytes(size), count)
		size, count = sizeAndCountOfUin64List(st.Slashings())
		log.Infof("slashings                     : size = %s, count = %d", humanize.Bytes(size), count)
		size, count = sizeAndCountGeneric(st.PreviousEpochAttestations())
		log.Infof("previous_epoch_attestations   : sizeSSZ = %s, count = %d", humanize.Bytes(size), count)
		size, count = sizeAndCountGeneric(st.CurrentEpochAttestations())
		log.Infof("current_epoch_attestations    : sizeSSZ = %s, count = %d", humanize.Bytes(size), count)
		log.Infof("justification_bits            : size =  %s, count = %d", humanize.Bytes(st.JustificationBits().Len()), st.JustificationBits().Count())
		log.Infof("previous_justified_checkpoint : sizeSSZ = %s", humanize.Bytes(uint64(st.PreviousJustifiedCheckpoint().SizeSSZ())))
		log.Infof("current_justified_checkpoint  : sizeSSZ = %s", humanize.Bytes(uint64(st.CurrentJustifiedCheckpoint().SizeSSZ())))
		log.Infof("finalized_checkpoint          : sizeSSZ = %s", humanize.Bytes(uint64(st.FinalizedCheckpoint().SizeSSZ())))

	}
	doneC <- true
}

func printStateSummary(stateSummaryC <-chan *modifiedStateSummary, doneC chan<- bool) {
	for msts := range stateSummaryC {
		log.Infof("row : %04d, slot : %d, root = %s", msts.rowCount, msts.slot, hexutils.BytesToHex(msts.root))
	}
	doneC <- true
}

func keysOfBucket(dbNameWithPath string, bucketName []byte, rowLimit uint64) ([][]byte, []uint64) {
	// open the raw database file. If the file is busy, then exit.
	db, openErr := bolt.Open(dbNameWithPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if openErr != nil {
		log.Fatalf("could not open db while getting keys of a bucket, %v", openErr)
	}

	// make sure we close the database before ejecting out of this function.
	defer func() {
		closeErr := db.Close()
		if closeErr != nil {
			log.Fatalf("could not close db while getting keys of a bucket, %v", closeErr)
		}
	}()

	// get all the keys of the given bucket.
	var keys [][]byte
	var sizes []uint64
	if viewErr := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		c := b.Cursor()
		count := uint64(0)
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if count >= rowLimit {
				return nil
			}
			keys = append(keys, k)
			sizes = append(sizes, uint64(len(v)))
			count++
		}
		return nil
	}); viewErr != nil {
		log.Fatalf("could not read keys of bucket from db: %v", viewErr)
	}
	return keys, sizes
}

func sizeAndCountOfByteList(list [][]byte) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	for _, root := range list {
		size += uint64(len(root))
		count += 1
	}
	return size, count
}

func sizeAndCountOfUin64List(list []uint64) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	for i := 0; i < len(list); i++ {
		size += uint64(8)
		count += 1
	}
	return size, count
}

func sizeAndCountGeneric(genericItems interface{}, err error) (uint64, uint64) {
	size := uint64(0)
	count := uint64(0)
	if err != nil {
		return size, count
	}

	switch items := genericItems.(type) {
	case []*ethpb.Eth1Data:
		for _, item := range items {
			size += uint64(item.SizeSSZ())
		}
		count = uint64(len(items))
	case []*ethpb.Validator:
		for _, item := range items {
			size += uint64(item.SizeSSZ())
		}
		count = uint64(len(items))
	case []*pbp2p.PendingAttestation:
		for _, item := range items {
			size += uint64(item.SizeSSZ())
		}
		count = uint64(len(items))
	default:
		return 0, 0
	}

	return size, count
}
