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
	"fmt"
	"os"
	"path/filepath"
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

type ModifiedState struct {
	state     iface.BeaconState
	key       []byte
	valueSize uint64
	rowCount  uint64
}

type ModifiedStateSummary struct {
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
		showBucketStats(dbNameWithPath)
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

func showBucketStats(dbNameWithPath string) {
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

	// for every bucket, calculate the stats and display them.
	// TODO: parallelize the execution
	for _, bName := range buckets {
		count := uint64(0)
		minValueSize := ^uint64(0)
		maxValueSize := uint64(0)
		totalValueSize := uint64(0)
		minKeySize := ^uint64(0)
		maxKeySize := uint64(0)
		totalKeySize := uint64(0)
		if viewErr2 := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bName))
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
				log.Errorf("could not process row %d for bucket: %s, %v", count, bName, forEachErr)
				return forEachErr
			}
			return nil
		}); viewErr2 != nil {
			log.Errorf("could not get stats for bucket: %s, %v", bName, viewErr2)
			continue
		}

		if count != 0 {
			averageValueSize := totalValueSize / count
			averageKeySize := totalKeySize / count
			fmt.Println("------ ", bName, " --------")
			fmt.Println("NumberOfRows     = ", count)
			fmt.Println("TotalBucketSize  = ", humanize.Bytes(totalValueSize+totalKeySize))
			fmt.Println("KeySize          = ", humanize.Bytes(totalKeySize), "(min = "+humanize.Bytes(minKeySize)+", avg = "+humanize.Bytes(averageKeySize)+", max = "+humanize.Bytes(maxKeySize)+")")
			fmt.Println("ValueSize        = ", humanize.Bytes(totalValueSize), "(min = "+humanize.Bytes(minValueSize)+", avg = "+humanize.Bytes(averageValueSize)+", max = "+humanize.Bytes(maxValueSize)+")")
		}
	}
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

	// dont forget to close it when ejecting out of this function.
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
		stateC := make(chan *ModifiedState, groupSize)
		go readStates(ctx, db, stateC, keys, sizes)
		go printStates(stateC, doneC)

	case "state-summary":
		stateSummaryC := make(chan *ModifiedStateSummary, groupSize)
		go readStateSummary(ctx, db, stateSummaryC, keys, sizes)
		go printStateSummary(stateSummaryC, doneC)
	}
	<-doneC
}

func readStates(ctx context.Context, db *kv.Store, stateC chan<- *ModifiedState, keys [][]byte, sizes []uint64) {
	for rowCount, key := range keys {
		st, stateErr := db.State(ctx, bytesutil.ToBytes32(key))
		if stateErr != nil {
			log.Errorf("could not get state for key : , %v", stateErr)
		}
		mst := &ModifiedState{
			state:     st,
			key:       key,
			valueSize: sizes[rowCount],
			rowCount:  uint64(rowCount),
		}
		stateC <- mst
	}
	close(stateC)
}

func readStateSummary(ctx context.Context, db *kv.Store, stateSummaryC chan<- *ModifiedStateSummary, keys [][]byte, sizes []uint64) {
	for rowCount, key := range keys {
		ss, ssErr := db.StateSummary(ctx, bytesutil.ToBytes32(key))
		if ssErr != nil {
			log.Errorf("could not get state summary for key : , %v", ssErr)
		}
		mst := &ModifiedStateSummary{
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

func printStates(stateC <-chan *ModifiedState, doneC chan<- bool) {
	for mst := range stateC {
		st := mst.state
		rowStr := fmt.Sprintf("---- row = %04d ----", mst.rowCount)
		fmt.Println(rowStr)
		fmt.Println("key                           :", hexutils.BytesToHex(mst.key))
		fmt.Println("value                         : compressed size = ", humanize.Bytes(mst.valueSize))
		fmt.Println("genesis_time                  :", st.GenesisTime())
		fmt.Println("genesis_validators_root       :", hexutils.BytesToHex(st.GenesisValidatorRoot()))
		fmt.Println("slot                          :", st.Slot())
		fmt.Println("fork                          : previous_version: ", st.Fork().PreviousVersion, ",  current_version: ", st.Fork().CurrentVersion)
		fmt.Println("latest_block_header           : sizeSSZ = ", humanize.Bytes(uint64(st.LatestBlockHeader().SizeSSZ())))
		size, count := sizeAndCountOfByteList(st.BlockRoots())
		fmt.Println("block_roots                   : size =  ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfByteList(st.StateRoots())
		fmt.Println("state_roots                   : size =  ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfByteList(st.HistoricalRoots())
		fmt.Println("historical_roots              : size =  ", humanize.Bytes(size), ", count =  ", count)
		fmt.Println("eth1_data                     : sizeSSZ =  ", humanize.Bytes(uint64(st.Eth1Data().SizeSSZ())))
		size, count = sizeAndCountGeneric(st.Eth1DataVotes(), nil)
		fmt.Println("eth1_data_votes               : sizeSSZ = ", humanize.Bytes(size), ", count =  ", count)
		fmt.Println("eth1_deposit_index            :", st.Eth1DepositIndex())
		size, count = sizeAndCountGeneric(st.Validators(), nil)
		fmt.Println("validators                    : sizeSSZ = ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfUin64List(st.Balances())
		fmt.Println("balances                      : size = ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfByteList(st.RandaoMixes())
		fmt.Println("randao_mixes                  : size = ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountOfUin64List(st.Slashings())
		fmt.Println("slashings                     : size =  ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountGeneric(st.PreviousEpochAttestations())
		fmt.Println("previous_epoch_attestations   : sizeSSZ ", humanize.Bytes(size), ", count =  ", count)
		size, count = sizeAndCountGeneric(st.CurrentEpochAttestations())
		fmt.Println("current_epoch_attestations    : sizeSSZ =  ", humanize.Bytes(size), ", count =  ", count)
		fmt.Println("justification_bits            : size =  ", humanize.Bytes(st.JustificationBits().Len()), ", count =  ", st.JustificationBits().Count())
		fmt.Println("previous_justified_checkpoint : sizeSSZ =  ", humanize.Bytes(uint64(st.PreviousJustifiedCheckpoint().SizeSSZ())))
		fmt.Println("current_justified_checkpoint  : sizeSSZ =  ", humanize.Bytes(uint64(st.CurrentJustifiedCheckpoint().SizeSSZ())))
		fmt.Println("finalized_checkpoint          : sizeSSZ =  ", humanize.Bytes(uint64(st.FinalizedCheckpoint().SizeSSZ())))

	}
	doneC <- true
}

func printStateSummary(stateSummaryC <-chan *ModifiedStateSummary, doneC chan<- bool) {
	for msts := range stateSummaryC {
		rowCountStr := fmt.Sprintf("row : %04d, ", msts.rowCount)
		fmt.Println(rowCountStr, "slot : ", msts.slot, ", root : ", hexutils.BytesToHex(msts.root))
	}
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
