package db

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/big"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
func (db *BeaconDB) InsertDeposit(ctx context.Context, d *pb.Deposit, blockNum *big.Int, depositRoot [32]byte) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.InsertDeposit")
	defer span.End()
	if d == nil || blockNum == nil {
		log.WithFields(logrus.Fields{
			"block":        blockNum,
			"deposit":      d,
			"deposit root": hex.EncodeToString(depositRoot[:]),
		}).Debug("Ignoring nil deposit insertion")
		return
	}
	db.depositsLock.Lock()
	defer db.depositsLock.Unlock()
	// keep the slice sorted on insertion in order to avoid costly sorting on retrival.
	heightIdx := sort.Search(len(db.deposits), func(i int) bool { return db.deposits[i].Block.Cmp(blockNum) >= 0 })
	db.deposits = append(db.deposits[:heightIdx], append([]*DepositContainer{{deposit: d, Block: blockNum, DepositRoot: depositRoot}}, db.deposits[heightIdx:]...)...)
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
// (inclusive). If no block is specified then this method returns all historical deposits.
func (db *BeaconDB) AllDeposits(ctx context.Context, beforeBlk *big.Int) []*pb.Deposit {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AllDeposits")
	defer span.End()
	db.depositsLock.RLock()
	defer db.depositsLock.RUnlock()

	var deposits []*pb.Deposit
	for _, ctnr := range db.deposits {
		if beforeBlk == nil || beforeBlk.Cmp(ctnr.Block) > -1 {
			deposits = append(deposits, ctnr.deposit)
		}
	}
	return deposits
}

// DepositsContainersTillBlock returns the deposit containers sorted by block height where
// block height lower then beforeBlk.
func (db *BeaconDB) DepositsContainersTillBlock(ctx context.Context, beforeBlk *big.Int) []*DepositContainer {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DepositsContainersTillBlock")
	defer span.End()
	db.depositsLock.RLock()
	defer db.depositsLock.RUnlock()

	var deposits []*DepositContainer
	for _, ctnr := range db.deposits {
		if beforeBlk == nil || beforeBlk.Cmp(ctnr.Block) > -1 {
			deposits = append(deposits, ctnr)
		}
	}
	return deposits
}

// DepositByPubkey looks through historical deposits and finds one which contains
// a certain public key within its deposit data.
func (db *BeaconDB) DepositByPubkey(ctx context.Context, pubKey []byte) (*pb.Deposit, *big.Int) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DepositByPubkey")
	defer span.End()
	db.depositsLock.RLock()
	defer db.depositsLock.RUnlock()

	var deposit *pb.Deposit
	var blockNum *big.Int
	for _, ctnr := range db.deposits {
		if bytes.Equal(ctnr.deposit.Data.Pubkey, pubKey) {
			deposit = ctnr.deposit
			blockNum = ctnr.Block
			break
		}
	}
	return deposit, blockNum
}
