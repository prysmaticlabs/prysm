package kv

import (
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common"
	"go.opencensus.io/trace"
)

// DepositContractAddress returns contract address is the address of
// the deposit contract on the proof of work chain.
func (k *Store) DepositContractAddress(ctx context.Context) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DepositContractAddress")
	defer span.End()
	var addr []byte
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainMetadataBucket)
		addr = chainInfo.Get(depositContractAddressKey)
		return nil
	})
	return addr, nil
}

// SaveDepositContractAddress to the db. It returns an error if an address has been previously saved.
func (k *Store) SaveDepositContractAddress(ctx context.Context, addr common.Address) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.VerifyContractAddress")
	defer span.End()

	return k.db.Update(func(tx *bolt.Tx) error {
		chainInfo := tx.Bucket(chainMetadataBucket)
		expectedAddress := chainInfo.Get(depositContractAddressKey)
		if expectedAddress != nil {
			return fmt.Errorf("cannot override deposit contract address: %v", expectedAddress)
		}
		return chainInfo.Put(depositContractAddressKey, addr.Bytes())
	})
}
