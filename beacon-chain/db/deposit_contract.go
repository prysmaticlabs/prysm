package db

import (
	"bytes"
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

var depositContractAddressKey = []byte("deposit-contract")

// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) DepositContractAddress(ctx context.Context) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DepositContractAddress")
	defer span.End()

	var addr []byte
	err := db.view(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainInfoBucket)
		addr = chainInfo.Get(depositContractAddressKey)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
func (db *BeaconDB) SaveDepositContractAddress(ctx context.Context, addr common.Address) error {
	return errors.New("unimplemented")
}

// DEPRECATED: Use the kv store in beacon-chain/db/kv instead.
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
