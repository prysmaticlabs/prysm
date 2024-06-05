package blockchain

import (
	"fmt"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/peerdas"
	"github.com/prysmaticlabs/prysm/v5/cmd/beacon-chain/flags"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
)

// blobDataColumnNotifier is a notifier that notifies when all the data columns required to custody are available for a blockRoot.
type blobDataColumnNotifier struct {
	sync.RWMutex

	// notifiers is a map of blockRoot to a channel that will be closed when all the data columns required to custody is available for that blockRoot.
	notifiers map[[32]byte]chan struct{}

	// seenIndex is a map of blockRoot to a set of column indexes that have been seen for that blockRoot.
	seenIndex map[[32]byte]map[uint64]bool

	// columnsNeedsCustody is column set that needs to be custodied by the current node, this does not change after initialization.
	columnsNeedsCustody map[uint64]bool
}

func newBlobDataColumnNotifier(nodeID enode.ID) (*blobDataColumnNotifier, error) {
	subnetCount := params.BeaconConfig().CustodyRequirement
	if flags.Get().SubscribeToAllSubnets {
		subnetCount = params.BeaconConfig().DataColumnSidecarSubnetCount
	}

	columnsNeedsCustody, err := peerdas.CustodyColumns(nodeID, subnetCount)
	if err != nil {
		return nil, err
	}

	return &blobDataColumnNotifier{
		notifiers:           make(map[[32]byte]chan struct{}),
		seenIndex:           make(map[[32]byte]map[uint64]bool),
		columnsNeedsCustody: columnsNeedsCustody,
	}, nil
}

// dataAvailable returns a channel that will be closed when all the data columns required to custody are available for the given blockRoot.
func (bn *blobDataColumnNotifier) dataAvailable(blockRoot [32]byte) <-chan struct{} {
	bn.Lock()
	defer bn.Unlock()

	notifier, ok := bn.notifiers[blockRoot]
	if !ok {
		bn.initializeBlockRoot(blockRoot)
		notifier = bn.notifiers[blockRoot]
	}

	return notifier
}

// receiveBlobDataColumn notifies the notifier that a new blob data column is available for the given blockRoot.
func (bn *blobDataColumnNotifier) receiveBlobDataColumn(blockRoot [32]byte, columnIndex uint64) {
	if columnIndex >= fieldparams.NumberOfColumns {
		return
	}

	bn.Lock()
	defer bn.Unlock()

	seen := bn.seenIndex[blockRoot]
	if seen == nil {
		bn.initializeBlockRoot(blockRoot)
		seen = bn.seenIndex[blockRoot]
	}
	if seen[columnIndex] {
		return
	}
	seen[columnIndex] = true

	if !bn.allColumnsDownloaded(seen) {
		return
	}

	notifier, _ := bn.notifiers[blockRoot]
	notifier <- struct{}{}
}

// missingBlobDataColumns returns the list of missing data columns for the given blockRoot.
func (bn *blobDataColumnNotifier) missingBlobDataColumns(blockRoot [32]byte) []uint64 {
	bn.RLock()
	defer bn.RUnlock()

	missing := make([]uint64, 0)
	seen := bn.seenIndex[blockRoot]
	for k := range bn.columnsNeedsCustody {
		if !seen[k] {
			missing = append(missing, k)
		}
	}

	sort.Slice(missing, func(i, j int) bool {
		return missing[i] < missing[j]
	})
	return missing
}

// initialize initializes the data needed for a given blockRoot, it should only be called within the lock.
func (bn *blobDataColumnNotifier) initializeBlockRoot(blockRoot [32]byte) {
	bn.notifiers[blockRoot] = make(chan struct{}, 1)
	bn.seenIndex[blockRoot] = make(map[uint64]bool)
}

func (bn *blobDataColumnNotifier) deleteBlockRoot(blockRoot [32]byte) {
	delete(bn.notifiers, blockRoot)
	delete(bn.seenIndex, blockRoot)
}

// allColumnsDownloaded returns true if all the data columns required to custody are available for the given blockRoot.
func (bn *blobDataColumnNotifier) allColumnsDownloaded(seenColumns map[uint64]bool) bool {
	fmt.Println(seenColumns)
	fmt.Println(bn.columnsNeedsCustody)
	for k := range bn.columnsNeedsCustody {
		if !seenColumns[k] {
			return false
		}
	}

	fmt.Println("all columns downloaded")
	return true
}
