package depositsnapshot

import (
	"context"
	"encoding/hex"
	"sort"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	historicalDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacondb_all_deposits",
		Help: "The number of total deposits in the beaconDB in-memory database",
	})
	log = logrus.WithField("prefix", "depositcache")
)

// InsertDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (c *Cache) InsertDeposit(ctx context.Context, d *ethpb.Deposit, blockNum uint64, index int64, depositRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.InsertDeposit")
	defer span.End()
	if d == nil {
		log.WithFields(logrus.Fields{
			"block":        blockNum,
			"deposit":      d,
			"index":        index,
			"deposit root": hex.EncodeToString(depositRoot[:]),
		}).Warn("Ignoring nil deposit insertion")
		return errors.New("nil deposit inserted into the cache")
	}
	c.depositsLock.Lock()
	defer c.depositsLock.Unlock()

	if int(index) != len(c.deposits) {
		return errors.Errorf("wanted deposit with index %d to be inserted but received %d", len(c.deposits), index)
	}
	// Keep the slice sorted on insertion in order to avoid costly sorting on retrieval.
	heightIdx := sort.Search(len(c.deposits), func(i int) bool { return c.deposits[i].Index >= index })
	depCtr := &ethpb.DepositContainer{Deposit: d, Eth1BlockHeight: blockNum, DepositRoot: depositRoot[:], Index: index}
	newDeposits := append(
		[]*ethpb.DepositContainer{depCtr},
		c.deposits[heightIdx:]...)
	c.deposits = append(c.deposits[:heightIdx], newDeposits...)
	// Append the deposit to our map, in the event no deposits
	// exist for the pubkey , it is simply added to the map.
	pubkey := bytesutil.ToBytes48(d.Data.PublicKey)
	c.depositsByKey[pubkey] = append(c.depositsByKey[pubkey], depCtr)
	historicalDepositsCount.Inc()
	return nil
}
