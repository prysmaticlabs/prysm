/**
 * Explore DB contents
 *
 * Given a beacon-chain DB, This tool provides many option to
 * inspect and explore it.
 * i.e. For every bucket, print the number of rows, bucket size, min/average/max size of values
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var (
	datadir = flag.String("datadir", "", "Path to data directory.")
)

const (
	DatabaseFileName = "beaconchain.db"
)

func main() {
	flag.Parse()
	if *datadir == "" {
		log.Fatal("Please specify --datadir to read beaconchain.db")
	}

	// check if the database file is present
	dbName := filepath.Join(*datadir, DatabaseFileName)
	if _, err := os.Stat(*datadir); os.IsNotExist(err) {
		log.Fatalf("database file is not present, %v", err)
	}

	// open the beacon-chain database
	// if some other process has the file lock, it will quit after a second
	db, err := bolt.Open(dbName, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatalf("could not open db, %v", err)
	}
	defer db.Close()

	// get a list of all the existing buckets
	buckets := make(map[string]*bolt.Bucket)
	if err = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, buc *bolt.Bucket) error {
			buckets[string(name)] = buc
			return nil
		})
	}); err != nil {
		log.Fatalf("could not find buckets, %v", err)
	}

	showBucketStats(db, buckets)
}

func showBucketStats(db *bolt.DB, buckets map[string]*bolt.Bucket) {
	for bName, _ := range buckets {
		count := uint64(0)
		minValueSize := ^uint64(0)
		maxValueSize := uint64(0)
		totalValueSize := uint64(0)
		minKeySize := ^uint64(0)
		maxKeySize := uint64(0)
		totalKeySize := uint64(0)
		if err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bName))
			if err := b.ForEach(func(k, v []byte) error {
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
				count++
				return nil
			}); err != nil {
				log.Errorf("could not process row %d for bucket: %s, %v", count, bName, err)
				return err
			}
			return nil
		}); err != nil {
			log.Errorf("could not get stats for bucket: %s, %v", bName, err)
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
