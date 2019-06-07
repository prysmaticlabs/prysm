package db

import (
	"bytes"
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common"
	"go.opencensus.io/trace"
)

var depositContractAddressKey = []byte("deposit-contract")

// VerifyContractAddress that represents the data in this database. The
// contract address is the address of the deposit contract on the proof of work
// Ethereum chain. This value will never change or all of the data in the
// database would be made invalid.
func (db *BeaconDB) VerifyContractAddress(ctx context.Context, addr common.Address) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.VerifyContractAddress")
	defer span.End()

	return db.update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)

		expectedAddress := chainInfo.Get(depositContractAddressKey)
		if expectedAddress == nil {
			return chainInfo.Put(depositContractAddressKey, addr.Bytes())
		}

		if !bytes.Equal(expectedAddress, addr.Bytes()) {
			return fmt.Errorf("invalid deposit contract address, expected %#x - try running with --clear-db", expectedAddress)
		}

		return nil
	})
}
