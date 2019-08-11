package db

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/big"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	historicalDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacondb_all_deposits",
		Help: "The number of total deposits in the beaconDB in-memory database",
	})
)

// InsertDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (db *BeaconDB) InsertDeposit(ctx context.Context, d *ethpb.Deposit, blockNum *big.Int, index int, depositRoot [32]byte) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.InsertDeposit")
	defer span.End()
	if d == nil || blockNum == nil {
		log.WithFields(logrus.Fields{
			"block":        blockNum,
			"deposit":      d,
			"index":        index,
			"deposit root": hex.EncodeToString(depositRoot[:]),
		}).Warn("Ignoring nil deposit insertion")
		return
	}
	db.depositsLock.Lock()
	defer db.depositsLock.Unlock()

	// Keep the slice sorted on insertion in order to avoid costly sorting on retrieval.
	insertionIndex := sort.Search(len(db.deposits), func(i int) bool { return db.deposits[i].Index >= index })
	newDeposits := append([]*DepositContainer{{Deposit: d, Block: blockNum, depositRoot: depositRoot, Index: index}}, db.deposits[insertionIndex:]...)
	db.deposits = append(db.deposits[:insertionIndex], newDeposits...)
	historicalDepositsCount.Inc()
}

// MarkPubkeyForChainstart sets the pubkey deposit status to true.
func (db *BeaconDB) MarkPubkeyForChainstart(ctx context.Context, pubkey string) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.MarkPubkeyForChainstart")
	defer span.End()
	db.chainstartPubkeysLock.Lock()
	defer db.chainstartPubkeysLock.Unlock()
	db.chainstartPubkeys[pubkey] = true
}

// PubkeyInChainstart returns bool for whether the pubkey passed in has deposited.
func (db *BeaconDB) PubkeyInChainstart(ctx context.Context, pubkey string) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PubkeyInChainstart")
	defer span.End()
	db.chainstartPubkeysLock.Lock()
	defer db.chainstartPubkeysLock.Unlock()
	if db.chainstartPubkeys != nil {
		return db.chainstartPubkeys[pubkey]
	}
	db.chainstartPubkeys = make(map[string]bool)
	return false
}

// AllDeposits returns a list of deposits all historical deposits until the given block number
// (inclusive). If no block number is specified then this method returns all historical deposits.
func (db *BeaconDB) AllDeposits(ctx context.Context, beforeBlk *big.Int) []*ethpb.Deposit {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AllDeposits")
	defer span.End()
	db.depositsLock.RLock()
	defer db.depositsLock.RUnlock()
	var deposits []*ethpb.Deposit
	for _, ctnr := range db.deposits {
		if beforeBlk == nil || beforeBlk.Cmp(ctnr.Block) > -1 {
			deposits = append(deposits, ctnr.Deposit)
		}
	}
	return deposits
}

// DepositsNumberAndRootAtHeight returns number of deposits made prior to block height and the
// deposit root that corresponds to the latest deposit at that block height.
func (db *BeaconDB) DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DepositsNumberAndRootAtHeight")
	defer span.End()
	db.depositsLock.RLock()
	defer db.depositsLock.RUnlock()
	upperBounds := sort.Search(len(db.deposits), func(i int) bool { return db.deposits[i].Block.Cmp(blockHeight) > 0 })
	// Send the deposit root of the empty trie, if block height is less than the block height of
	// the earliest deposit.
	if upperBounds == 0 {
		return 0, [32]byte{}
	}
	return uint64(upperBounds), db.deposits[upperBounds-1].depositRoot
}

// DepositByPubkey looks through historical deposits and finds the first one which
// contains a certain public key within its deposit data.
func (db *BeaconDB) DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DepositByPubkey")
	defer span.End()
	db.depositsLock.RLock()
	defer db.depositsLock.RUnlock()

	var deposit *ethpb.Deposit
	var blockNum *big.Int
	for _, ctnr := range db.deposits {
		if bytes.Equal(ctnr.Deposit.Data.PublicKey, pubKey) {
			deposit = ctnr.Deposit
			blockNum = ctnr.Block
			break
		}
	}
	return deposit, blockNum
}
