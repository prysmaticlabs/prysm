package sync

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/prysmaticlabs/prysm/v5/async"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/peerdas"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/types"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/crypto/rand"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

const PeerRefreshInterval = 1 * time.Minute

// DataColumnSampler defines the interface for sampling data columns from peers for requested block root and samples count.
type DataColumnSampler interface {
	// Run starts the data column sampling service.
	Run(ctx context.Context, stateNotifier statefeed.Notifier)
}

var _ DataColumnSampler = (*DataColumnSampler1D)(nil)

// DataColumnSampler1D is a 1D data column sampler for PeerDAS 1D.
type DataColumnSampler1D struct {
	sync.RWMutex

	p2p p2p.P2P

	// peerToColumnMap maps a peer to the columns it is responsible for custody.
	peerToColumnMap map[peer.ID]map[uint64]bool

	// columnToPeerMap maps a column to the peer responsible for custody.
	columnToPeerMap map[uint64]map[peer.ID]bool
}

// NewDataColumnSampler1D creates a new 1D data column sampler.
func NewDataColumnSampler1D(p2p p2p.P2P) *DataColumnSampler1D {
	columnToPeerMap := make(map[uint64]map[peer.ID]bool, params.BeaconConfig().NumberOfColumns)
	for i := uint64(0); i < params.BeaconConfig().NumberOfColumns; i++ {
		columnToPeerMap[i] = make(map[peer.ID]bool)
	}

	return &DataColumnSampler1D{
		p2p:             p2p,
		peerToColumnMap: make(map[peer.ID]map[uint64]bool),
	}
}

// Run implements DataColumnSampler.
func (d *DataColumnSampler1D) Run(ctx context.Context, stateNotifier statefeed.Notifier) {
	// initialize peer info first.
	d.refreshPeerInfo()

	// periodically refresh peer info to keep peer <-> column mapping up to date.
	async.RunEvery(ctx, PeerRefreshInterval, d.refreshPeerInfo)

	// start the sampling loop.
	d.samplingLoop(ctx, stateNotifier)
}

// sampleDataColumns samples data columns from active peers.
// It should return an error if sampling fails (depends on the actual failing scenario).
func (d *DataColumnSampler1D) sampleDataColumns(blockRoot [32]byte, samplesCount uint64) error {
	peerToColumns, err := d.distributeSamplesToPeer(samplesCount)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(context.Background())
	for pid, columns := range peerToColumns {
		pid, columns := pid, columns

		eg.Go(func() error {
			return d.sampleDataColumnsFromPeer(ctx, pid, blockRoot, columns)
		})
	}

	if err := eg.Wait(); err != nil {
		log.WithFields(logrus.Fields{
			"blockRoot":    fmt.Sprintf("%#x", blockRoot),
			"samplesCount": samplesCount,
			"error":        err.Error(),
		}).Error("Failed to sample data columns from peers")
		return errors.Wrap(err, "error sampling data columns")
	}

	return nil
}

// distributeSamplesToPeer dynamically matches samples to peers based on the number of peers the node have.
// It will try to intelligently choose samples to request from peers that:
// *. minimizes the chance of false positive result
// *. maximizes the change to evenly distribute the samples to the peers if possible.
func (d *DataColumnSampler1D) distributeSamplesToPeer(samplesCount uint64) (map[peer.ID]map[uint64]bool, error) {
	res := make(map[peer.ID]map[uint64]bool)

	columnsToSample := randomIntegers(samplesCount, params.BeaconConfig().NumberOfColumns)
	for col := range columnsToSample {
		peers := d.columnToPeerMap[col]
		if len(peers) == 0 {
			return nil, errors.Errorf("no peers responsible for column %d", col)
		}

		// randomly choose a peer to sample the column.
		pid := randomPeerFromMap(peers)
		if _, ok := res[pid]; !ok {
			res[pid] = make(map[uint64]bool)
		}
		res[pid][col] = true
	}

	return res, nil
}

// sampleDataColumnsFromPeer samples data columns from a peer.
func (d *DataColumnSampler1D) sampleDataColumnsFromPeer(ctx context.Context, pid peer.ID, blockRoot [32]byte, columns map[uint64]bool) error {
	// SendDataColumnSidecarByRoot()

	// dataColumnIdentifiers := make(types.BlobSidecarsByRootReq, 0, len(columns))
	panic("not implemented")
}

// Refresh peer information.
func (d *DataColumnSampler1D) refreshPeerInfo() {
	for _, pid := range d.p2p.Peers().Active() {
		if _, ok := d.peerToColumnMap[pid]; ok {
			continue
		}

		peerCustodiedSubnetCount := d.p2p.CustodyCountFromRemotePeer(pid)
		nodeID, err := p2p.ConvertPeerIDToNodeID(pid)
		if err != nil {
			log.WithError(err).WithField("peerID", pid).Error("Failed to convert peer ID to node ID")
			continue
		}

		peerCustodiedColumns, err := peerdas.CustodyColumns(nodeID, peerCustodiedSubnetCount)
		if err != nil {
			log.WithError(err).WithField("peerID", pid).Error("Failed to determine peer custody columns")
			continue
		}

		d.peerToColumnMap[pid] = peerCustodiedColumns
		for column := range peerCustodiedColumns {
			d.columnToPeerMap[column][pid] = true
		}
	}
}

func (d *DataColumnSampler1D) samplingLoop(ctx context.Context, stateNotifier statefeed.Notifier) {
	// Create a subscription to the state feed.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := stateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	for {
		select {
		case evt := <-stateChannel:
			if evt.Type != statefeed.BlockProcessed {
				continue
			}

			data, ok := evt.Data.(*statefeed.BlockProcessedData)
			if !ok {
				log.Error("Event feed data is not of type *statefeed.BlockProcessedData")
				continue
			}

			if !data.Verified {
				// We only process blocks that have been verified
				log.Error("Data is not verified")
				continue
			}

			if data.SignedBlock.Version() < version.Deneb {
				log.Debug("Pre Deneb block, skipping data column sampling")
				continue
			}

			// Get the commitments for this block.
			commitments, err := data.SignedBlock.Block().Body().BlobKzgCommitments()
			if err != nil {
				log.WithError(err).Error("Failed to get blob KZG commitments")
				continue
			}

			// Skip if there are no commitments.
			if len(commitments) == 0 {
				log.Debug("No commitments in block, skipping data column sampling")
				continue
			}

			if err := d.sampleDataColumns(data.BlockRoot, params.BeaconConfig().SamplesPerSlot); err != nil {
				log.WithError(err).Error("Failed to sample data columns")
			}
		case err := <-stateSub.Err():
			log.WithError(err).Error("DataColumnSampler1D subscription to state feed failed")
		case <-ctx.Done():
			log.Debug("Context canceled, exiting data column sampling loop.")
			return
		}
	}
}

// reandomIntegers returns a map of `count` random integers in the range [0, max[.
func randomIntegers(count uint64, max uint64) map[uint64]bool {
	result := make(map[uint64]bool, count)
	randGenerator := rand.NewGenerator()

	for uint64(len(result)) < count {
		n := randGenerator.Uint64() % max
		result[n] = true
	}

	return result
}

// sortedListFromMap returns a sorted list of keys from a map.
func sortedListFromMap(m map[uint64]bool) []uint64 {
	result := make([]uint64, 0, len(m))
	for k := range m {
		result = append(result, k)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}

func randomPeerFromMap(m map[peer.ID]bool) peer.ID {
	list := make([]peer.ID, 0, len(m))
	for k := range m {
		list = append(list, k)
	}

	return list[rand.NewGenerator().Uint64()%uint64(len(list))]
}

// extractNodeID extracts the node ID from a peer ID.
func extractNodeID(pid peer.ID) ([32]byte, error) {
	var nodeID [32]byte

	// Retrieve the public key object of the peer under "crypto" form.
	pubkeyObjCrypto, err := pid.ExtractPublicKey()
	if err != nil {
		return nodeID, errors.Wrap(err, "extract public key")
	}

	// Extract the bytes representation of the public key.
	compressedPubKeyBytes, err := pubkeyObjCrypto.Raw()
	if err != nil {
		return nodeID, errors.Wrap(err, "public key raw")
	}

	// Retrieve the public key object of the peer under "SECP256K1" form.
	pubKeyObjSecp256k1, err := btcec.ParsePubKey(compressedPubKeyBytes)
	if err != nil {
		return nodeID, errors.Wrap(err, "parse public key")
	}

	// Concatenate the X and Y coordinates represented in bytes.
	buf := make([]byte, 64)
	math.ReadBits(pubKeyObjSecp256k1.X(), buf[:32])
	math.ReadBits(pubKeyObjSecp256k1.Y(), buf[32:])

	// Get the node ID by hashing the concatenated X and Y coordinates.
	nodeIDBytes := crypto.Keccak256(buf)
	copy(nodeID[:], nodeIDBytes)

	return nodeID, nil
}

// sampleDataColumnFromPeer samples data columns from a peer.
// It returns the missing columns after sampling.
func (s *Service) sampleDataColumnFromPeer(
	pid peer.ID,
	columnsToSample map[uint64]bool,
	requestedRoot [fieldparams.RootLength]byte,
) (map[uint64]bool, error) {
	// Define missing columns.
	missingColumns := make(map[uint64]bool, len(columnsToSample))
	for index := range columnsToSample {
		missingColumns[index] = true
	}

	// Retrieve the custody count of the peer.
	peerCustodiedSubnetCount := s.cfg.p2p.CustodyCountFromRemotePeer(pid)

	// Extract the node ID from the peer ID.
	nodeID, err := extractNodeID(pid)
	if err != nil {
		return nil, errors.Wrap(err, "extract node ID")
	}

	// Determine which columns the peer should custody.
	peerCustodiedColumns, err := peerdas.CustodyColumns(nodeID, peerCustodiedSubnetCount)
	if err != nil {
		return nil, errors.Wrap(err, "custody columns")
	}

	peerCustodiedColumnsList := sortedListFromMap(peerCustodiedColumns)

	// Compute the intersection of the columns to sample and the columns the peer should custody.
	peerRequestedColumns := make(map[uint64]bool, len(columnsToSample))
	for column := range columnsToSample {
		if peerCustodiedColumns[column] {
			peerRequestedColumns[column] = true
		}
	}

	peerRequestedColumnsList := sortedListFromMap(peerRequestedColumns)

	// Get the data column identifiers to sample from this peer.
	dataColumnIdentifiers := make(types.DataColumnSidecarsByRootReq, 0, len(peerRequestedColumns))
	for index := range peerRequestedColumns {
		dataColumnIdentifiers = append(dataColumnIdentifiers, &eth.DataColumnIdentifier{
			BlockRoot:   requestedRoot[:],
			ColumnIndex: index,
		})
	}

	// Return early if there are no data columns to sample.
	if len(dataColumnIdentifiers) == 0 {
		log.WithFields(logrus.Fields{
			"peerID":           pid,
			"custodiedColumns": peerCustodiedColumnsList,
			"requestedColumns": peerRequestedColumnsList,
		}).Debug("Peer does not custody any of the requested columns")
		return columnsToSample, nil
	}

	// Sample data columns.
	roDataColumns, err := SendDataColumnSidecarByRoot(s.ctx, s.cfg.clock, s.cfg.p2p, pid, s.ctxMap, &dataColumnIdentifiers)
	if err != nil {
		return nil, errors.Wrap(err, "send data column sidecar by root")
	}

	peerRetrievedColumns := make(map[uint64]bool, len(roDataColumns))

	// Remove retrieved items from rootsByDataColumnIndex.
	for _, roDataColumn := range roDataColumns {
		retrievedColumn := roDataColumn.ColumnIndex

		actualRoot := roDataColumn.BlockRoot()
		if actualRoot != requestedRoot {
			// TODO: Should we decrease the peer score here?
			log.WithFields(logrus.Fields{
				"peerID":        pid,
				"requestedRoot": fmt.Sprintf("%#x", requestedRoot),
				"actualRoot":    fmt.Sprintf("%#x", actualRoot),
			}).Warning("Actual root does not match requested root")

			continue
		}

		peerRetrievedColumns[retrievedColumn] = true

		if !columnsToSample[retrievedColumn] {
			// TODO: Should we decrease the peer score here?
			log.WithFields(logrus.Fields{
				"peerID":           pid,
				"retrievedColumn":  retrievedColumn,
				"requestedColumns": peerRequestedColumnsList,
			}).Warning("Retrieved column is was not requested")
		}

		delete(missingColumns, retrievedColumn)
	}

	peerRetrievedColumnsList := sortedListFromMap(peerRetrievedColumns)
	remainingMissingColumnsList := sortedListFromMap(missingColumns)

	log.WithFields(logrus.Fields{
		"peerID":                  pid,
		"custodiedColumns":        peerCustodiedColumnsList,
		"requestedColumns":        peerRequestedColumnsList,
		"retrievedColumns":        peerRetrievedColumnsList,
		"remainingMissingColumns": remainingMissingColumnsList,
	}).Debug("Peer data column sampling summary")

	return missingColumns, nil
}

// sampleDataColumns samples data columns from active peers.
func (s *Service) sampleDataColumns(requestedRoot [fieldparams.RootLength]byte, samplesCount uint64) error {
	// Determine `samplesCount` random column indexes.
	requestedColumns := randomIntegers(samplesCount, params.BeaconConfig().NumberOfColumns)

	missingColumns := make(map[uint64]bool, len(requestedColumns))
	for index := range requestedColumns {
		missingColumns[index] = true
	}

	// Get the active peers from the p2p service.
	activePeers := s.cfg.p2p.Peers().Active()

	var err error

	// Sampling is done sequentially peer by peer.
	// TODO: Add parallelism if (probably) needed.
	for _, pid := range activePeers {
		// Early exit if all needed columns are already sampled. (This is the happy path.)
		if len(missingColumns) == 0 {
			break
		}

		// Sample data columns from the peer.
		missingColumns, err = s.sampleDataColumnFromPeer(pid, missingColumns, requestedRoot)
		if err != nil {
			return errors.Wrap(err, "sample data column from peer")
		}
	}

	requestedColumnsList := sortedListFromMap(requestedColumns)

	if len(missingColumns) == 0 {
		log.WithField("requestedColumns", requestedColumnsList).Debug("Successfully sampled all requested columns")
		return nil
	}

	missingColumnsList := sortedListFromMap(missingColumns)
	log.WithFields(logrus.Fields{
		"requestedColumns": requestedColumnsList,
		"missingColumns":   missingColumnsList,
	}).Warning("Failed to sample some requested columns")

	return nil
}

func (s *Service) dataColumnSampling(ctx context.Context) {
	// Create a subscription to the state feed.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.cfg.stateNotifier.StateFeed().Subscribe(stateChannel)

	// Unsubscribe from the state feed when the function returns.
	defer stateSub.Unsubscribe()

	for {
		select {
		case e := <-stateChannel:
			if e.Type != statefeed.BlockProcessed {
				continue
			}

			data, ok := e.Data.(*statefeed.BlockProcessedData)
			if !ok {
				log.Error("Event feed data is not of type *statefeed.BlockProcessedData")
				continue
			}

			if !data.Verified {
				// We only process blocks that have been verified
				log.Error("Data is not verified")
				continue
			}

			if data.SignedBlock.Version() < version.Deneb {
				log.Debug("Pre Deneb block, skipping data column sampling")
				continue
			}

			// Get the commitments for this block.
			commitments, err := data.SignedBlock.Block().Body().BlobKzgCommitments()
			if err != nil {
				log.WithError(err).Error("Failed to get blob KZG commitments")
				continue
			}

			// Skip if there are no commitments.
			if len(commitments) == 0 {
				log.Debug("No commitments in block, skipping data column sampling")
				continue
			}

			dataColumnSamplingCount := params.BeaconConfig().SamplesPerSlot

			// Sample data columns.
			if err := s.sampleDataColumns(data.BlockRoot, dataColumnSamplingCount); err != nil {
				log.WithError(err).Error("Failed to sample data columns")
			}

		case <-s.ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return

		case err := <-stateSub.Err():
			log.WithError(err).Error("Subscription to state feed failed")
		}
	}
}
