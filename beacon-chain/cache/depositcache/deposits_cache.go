// Package depositcache is the source of validator deposits maintained
// in-memory by the beacon node – deposits processed from the
// eth1 powchain are then stored in this cache to be accessed by
// any other service during a beacon node's runtime.
package depositcache

import (
	"context"
	"encoding/hex"
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/cache"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/container/trie"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	historicalDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacondb_all_deposits",
		Help: "The number of total deposits in the beaconDB in-memory database",
	})
)

// FinalizedDeposits stores the trie of deposits that have been included
// in the beacon state up to the latest finalized checkpoint.
type FinalizedDeposits struct {
	deposits        *trie.SparseMerkleTrie
	merkleTrieIndex int64
}

// DepositCache stores all in-memory deposit objects. This
// stores all the deposit related data that is required by the beacon-node.
type DepositCache struct {
	// Beacon chain deposits in memory.
	pendingDeposits   []*ethpb.DepositContainer
	deposits          []*ethpb.DepositContainer
	finalizedDeposits FinalizedDeposits
	depositsByKey     map[[fieldparams.BLSPubkeyLength]byte][]*ethpb.DepositContainer
	depositsLock      sync.RWMutex
}

// New instantiates a new deposit cache
func New() (*DepositCache, error) {
	finalizedDepositsTrie, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	if err != nil {
		return nil, err
	}

	// finalizedDeposits.merkleTrieIndex is initialized to -1 because it represents the index of the last trie item.
	// Inserting the first item into the trie will set the value of the index to 0.
	return &DepositCache{
		pendingDeposits:   []*ethpb.DepositContainer{},
		deposits:          []*ethpb.DepositContainer{},
		depositsByKey:     map[[fieldparams.BLSPubkeyLength]byte][]*ethpb.DepositContainer{},
		finalizedDeposits: FinalizedDeposits{deposits: finalizedDepositsTrie, merkleTrieIndex: -1},
	}, nil
}

// InsertDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (dc *DepositCache) InsertDeposit(ctx context.Context, d *ethpb.Deposit, blockNum uint64, index int64, depositRoot [32]byte) error {
	_, span := trace.StartSpan(ctx, "DepositsCache.InsertDeposit")
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
	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()

	if int(index) != len(dc.deposits) {
		return errors.Errorf("wanted deposit with index %d to be inserted but received %d", len(dc.deposits), index)
	}
	// Keep the slice sorted on insertion in order to avoid costly sorting on retrieval.
	heightIdx := sort.Search(len(dc.deposits), func(i int) bool { return dc.deposits[i].Index >= index })
	depCtr := &ethpb.DepositContainer{Deposit: d, Eth1BlockHeight: blockNum, DepositRoot: depositRoot[:], Index: index}
	newDeposits := append(
		[]*ethpb.DepositContainer{depCtr},
		dc.deposits[heightIdx:]...)
	dc.deposits = append(dc.deposits[:heightIdx], newDeposits...)
	// Append the deposit to our map, in the event no deposits
	// exist for the pubkey , it is simply added to the map.
	pubkey := bytesutil.ToBytes48(d.Data.PublicKey)
	dc.depositsByKey[pubkey] = append(dc.depositsByKey[pubkey], depCtr)
	historicalDepositsCount.Inc()
	return nil
}

// InsertDepositContainers inserts a set of deposit containers into our deposit cache.
func (dc *DepositCache) InsertDepositContainers(ctx context.Context, ctrs []*ethpb.DepositContainer) {
	_, span := trace.StartSpan(ctx, "DepositsCache.InsertDepositContainers")
	defer span.End()
	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()

	sort.SliceStable(ctrs, func(i int, j int) bool { return ctrs[i].Index < ctrs[j].Index })
	dc.deposits = ctrs
	for _, c := range ctrs {
		// Use a new value, as the reference
		// of c changes in the next iteration.
		newPtr := c
		pKey := bytesutil.ToBytes48(newPtr.Deposit.Data.PublicKey)
		dc.depositsByKey[pKey] = append(dc.depositsByKey[pKey], newPtr)
	}
	historicalDepositsCount.Add(float64(len(ctrs)))
}

// InsertFinalizedDeposits inserts deposits up to eth1DepositIndex (inclusive) into the finalized deposits cache.
func (dc *DepositCache) InsertFinalizedDeposits(ctx context.Context,
	eth1DepositIndex int64, _ common.Hash, _ uint64) error {
	_, span := trace.StartSpan(ctx, "DepositsCache.InsertFinalizedDeposits")
	defer span.End()
	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()

	depositTrie := dc.finalizedDeposits.Deposits()
	insertIndex := int(dc.finalizedDeposits.merkleTrieIndex + 1)

	// Don't insert into finalized trie if there is no deposit to
	// insert.
	if len(dc.deposits) == 0 {
		return nil
	}
	// In the event we have less deposits than we need to
	// finalize we finalize till the index on which we do have it.
	if len(dc.deposits) <= int(eth1DepositIndex) {
		eth1DepositIndex = int64(len(dc.deposits)) - 1
	}
	// If we finalize to some lower deposit index, we
	// ignore it.
	if int(eth1DepositIndex) < insertIndex {
		return nil
	}
	for _, d := range dc.deposits {
		if d.Index <= dc.finalizedDeposits.merkleTrieIndex {
			continue
		}
		if d.Index > eth1DepositIndex {
			break
		}
		depHash, err := d.Deposit.Data.HashTreeRoot()
		if err != nil {
			return errors.Wrap(err, "could not hash deposit data")
		}
		if err = depositTrie.Insert(depHash[:], insertIndex); err != nil {
			return errors.Wrap(err, "could not insert deposit hash")
		}
		insertIndex++
	}
	tree, ok := depositTrie.(*trie.SparseMerkleTrie)
	if !ok {
		return errors.New("not a sparse merkle tree")
	}
	dc.finalizedDeposits = FinalizedDeposits{
		deposits:        tree,
		merkleTrieIndex: eth1DepositIndex,
	}
	return nil
}

// AllDepositContainers returns all historical deposit containers.
func (dc *DepositCache) AllDepositContainers(ctx context.Context) []*ethpb.DepositContainer {
	_, span := trace.StartSpan(ctx, "DepositsCache.AllDepositContainers")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	// Make a shallow copy of the deposits and return that. This way, the
	// caller can safely iterate over the returned list of deposits without
	// the possibility of new deposits showing up. If we were to return the
	// list without a copy, when a new deposit is added to the cache, it
	// would also be present in the returned value. This could result in a
	// race condition if the list is being iterated over.
	//
	// It's not necessary to make a deep copy of this list because the
	// deposits in the cache should never be modified. It is still possible
	// for the caller to modify one of the underlying deposits and modify
	// the cache, but that's not a race condition. Also, a deep copy would
	// take too long and use too much memory.
	deposits := make([]*ethpb.DepositContainer, len(dc.deposits))
	copy(deposits, dc.deposits)
	return deposits
}

// AllDeposits returns a list of historical deposits until the given block number
// (inclusive). If no block is specified then this method returns all historical deposits.
func (dc *DepositCache) AllDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit {
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	return dc.allDeposits(untilBlk)
}

func (dc *DepositCache) allDeposits(untilBlk *big.Int) []*ethpb.Deposit {
	var deposits []*ethpb.Deposit
	for _, ctnr := range dc.deposits {
		if untilBlk == nil || untilBlk.Uint64() >= ctnr.Eth1BlockHeight {
			deposits = append(deposits, ctnr.Deposit)
		}
	}
	return deposits
}

// DepositsNumberAndRootAtHeight returns number of deposits made up to blockheight and the
// root that corresponds to the latest deposit at that blockheight.
func (dc *DepositCache) DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte) {
	_, span := trace.StartSpan(ctx, "DepositsCache.DepositsNumberAndRootAtHeight")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()
	heightIdx := sort.Search(len(dc.deposits), func(i int) bool { return dc.deposits[i].Eth1BlockHeight > blockHeight.Uint64() })
	// send the deposit root of the empty trie, if eth1follow distance is greater than the time of the earliest
	// deposit.
	if heightIdx == 0 {
		return 0, [32]byte{}
	}
	return uint64(heightIdx), bytesutil.ToBytes32(dc.deposits[heightIdx-1].DepositRoot)
}

// DepositByPubkey looks through historical deposits and finds one which contains
// a certain public key within its deposit data.
func (dc *DepositCache) DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int) {
	_, span := trace.StartSpan(ctx, "DepositsCache.DepositByPubkey")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	var deposit *ethpb.Deposit
	var blockNum *big.Int
	deps, ok := dc.depositsByKey[bytesutil.ToBytes48(pubKey)]
	if !ok || len(deps) == 0 {
		return deposit, blockNum
	}
	// We always return the first deposit if a particular
	// validator key has multiple deposits assigned to
	// it.
	deposit = deps[0].Deposit
	blockNum = big.NewInt(int64(deps[0].Eth1BlockHeight))
	return deposit, blockNum
}

// FinalizedDeposits returns the finalized deposits trie.
func (dc *DepositCache) FinalizedDeposits(ctx context.Context) (cache.FinalizedDeposits, error) {
	_, span := trace.StartSpan(ctx, "DepositsCache.FinalizedDeposits")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	return &FinalizedDeposits{
		deposits:        dc.finalizedDeposits.deposits.Copy(),
		merkleTrieIndex: dc.finalizedDeposits.merkleTrieIndex,
	}, nil
}

// NonFinalizedDeposits returns the list of non-finalized deposits until the given block number (inclusive).
// If no block is specified then this method returns all non-finalized deposits.
func (dc *DepositCache) NonFinalizedDeposits(ctx context.Context, lastFinalizedIndex int64, untilBlk *big.Int) []*ethpb.Deposit {
	_, span := trace.StartSpan(ctx, "DepositsCache.NonFinalizedDeposits")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	if dc.finalizedDeposits.Deposits() == nil {
		return dc.allDeposits(untilBlk)
	}

	var deposits []*ethpb.Deposit
	for _, d := range dc.deposits {
		if (d.Index > lastFinalizedIndex) && (untilBlk == nil || untilBlk.Uint64() >= d.Eth1BlockHeight) {
			deposits = append(deposits, d.Deposit)
		}
	}

	return deposits
}

// PruneProofs removes proofs from all deposits whose index is equal or less than untilDepositIndex.
func (dc *DepositCache) PruneProofs(ctx context.Context, untilDepositIndex int64) error {
	_, span := trace.StartSpan(ctx, "DepositsCache.PruneProofs")
	defer span.End()
	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()

	if untilDepositIndex >= int64(len(dc.deposits)) {
		untilDepositIndex = int64(len(dc.deposits) - 1)
	}

	for i := untilDepositIndex; i >= 0; i-- {
		// Finding a nil proof means that all proofs up to this deposit have been already pruned.
		if dc.deposits[i].Deposit.Proof == nil {
			break
		}
		dc.deposits[i].Deposit.Proof = nil
	}

	return nil
}

// Deposits returns the cached internal deposit tree.
func (fd *FinalizedDeposits) Deposits() cache.MerkleTree {
	if fd.deposits != nil {
		return fd.deposits
	}
	return nil
}

// MerkleTrieIndex represents the last finalized index in
// the finalized deposit container.
func (fd *FinalizedDeposits) MerkleTrieIndex() int64 {
	return fd.merkleTrieIndex
}
