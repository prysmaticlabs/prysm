package sync

import (
	"context"
	"fmt"
	"sort"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

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

// randomSlice returns a slice of `count` random integers in the range [0, count[.
// Each item is unique.
func randomSlice(count uint64) []uint64 {
	slice := make([]uint64, count)

	for i := uint64(0); i < count; i++ {
		slice[i] = i
	}

	// Shuffle the slice.
	rand.NewGenerator().Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})

	return slice
}

// sortedSliceFromMap returns a sorted slices of keys from a map.
func sortedSliceFromMap(m map[uint64]bool) []uint64 {
	result := make([]uint64, 0, len(m))
	for k := range m {
		result = append(result, k)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}

// custodyColumnsFromPeer returns the columns the peer should custody.
func (s *Service) custodyColumnsFromPeer(pid peer.ID) (map[uint64]bool, error) {
	// Retrieve the custody count of the peer.
	custodySubnetCount := s.cfg.p2p.CustodyCountFromRemotePeer(pid)

	// Extract the node ID from the peer ID.
	nodeID, err := p2p.ConvertPeerIDToNodeID(pid)
	if err != nil {
		return nil, errors.Wrap(err, "extract node ID")
	}

	// Determine which columns the peer should custody.
	custodyColumns, err := peerdas.CustodyColumns(nodeID, custodySubnetCount)
	if err != nil {
		return nil, errors.Wrap(err, "custody columns")
	}

	return custodyColumns, nil
}

// sampleDataColumnsFromPeer samples data columns from a peer.
// It filters out columns that were not requested and columns with incorrect root.
// It returns the retrieved columns.
func (s *Service) sampleDataColumnsFromPeer(
	pid peer.ID,
	requestedColumns map[uint64]bool,
	root [fieldparams.RootLength]byte,
) (map[uint64]bool, error) {
	// Build the data column identifiers.
	dataColumnIdentifiers := make(types.DataColumnSidecarsByRootReq, 0, len(requestedColumns))
	for index := range requestedColumns {
		dataColumnIdentifiers = append(dataColumnIdentifiers, &eth.DataColumnIdentifier{
			BlockRoot:   root[:],
			ColumnIndex: index,
		})
	}

	// Send the request.
	roDataColumns, err := SendDataColumnSidecarByRoot(s.ctx, s.cfg.clock, s.cfg.p2p, pid, s.ctxMap, &dataColumnIdentifiers)
	if err != nil {
		return nil, errors.Wrap(err, "send data column sidecar by root")
	}

	retrievedColumns := make(map[uint64]bool, len(roDataColumns))

	// Remove retrieved items from rootsByDataColumnIndex.
	for _, roDataColumn := range roDataColumns {
		retrievedColumn := roDataColumn.ColumnIndex

		actualRoot := roDataColumn.BlockRoot()

		// Filter out columns with incorrect root.
		if actualRoot != root {
			// TODO: Should we decrease the peer score here?
			log.WithFields(logrus.Fields{
				"peerID":        pid,
				"requestedRoot": fmt.Sprintf("%#x", root),
				"actualRoot":    fmt.Sprintf("%#x", actualRoot),
			}).Warning("Actual root does not match requested root")

			continue
		}

		// Filter out columns that were not requested.
		if !requestedColumns[retrievedColumn] {
			// TODO: Should we decrease the peer score here?
			columnsToSampleList := sortedSliceFromMap(requestedColumns)

			log.WithFields(logrus.Fields{
				"peerID":           pid,
				"requestedColumns": columnsToSampleList,
				"retrievedColumn":  retrievedColumn,
			}).Warning("Retrieved column was not requested")

			continue
		}

		retrievedColumns[retrievedColumn] = true
	}

	if len(retrievedColumns) == len(requestedColumns) {
		// This is the happy path.
		log.WithFields(logrus.Fields{
			"peerID":           pid,
			"root":             fmt.Sprintf("%#x", root),
			"requestedColumns": sortedSliceFromMap(requestedColumns),
		}).Debug("All requested columns were successfully sampled from peer")

		return retrievedColumns, nil
	}

	// Some columns are missing.
	log.WithFields(logrus.Fields{
		"peerID":           pid,
		"root":             fmt.Sprintf("%#x", root),
		"requestedColumns": sortedSliceFromMap(requestedColumns),
		"retrievedColumns": sortedSliceFromMap(retrievedColumns),
	}).Warning("Some requested columns were not sampled from peer")

	return retrievedColumns, nil
}

// sampleDataColumnsFromPeers samples data columns from active peers.
// It returns the retrieved columns count.
// If one peer fails to return a column it should custody, the column is considered as missing.
func (s *Service) sampleDataColumnsFromPeers(
	columnsToSample []uint64,
	root [fieldparams.RootLength]byte,
) (uint64, error) {
	// Build all remaining columns to sample.
	remainingColumnsToSample := make(map[uint64]bool, len(columnsToSample))
	for _, column := range columnsToSample {
		remainingColumnsToSample[column] = true
	}

	// Get the active peers from the p2p service.
	activePids := s.cfg.p2p.Peers().Active()

	// Query all peers until either all columns to request are retrieved or all active peers are queried (whichever comes first).
	retrievedColumnsCount := 0

	for i := 0; len(remainingColumnsToSample) > 0 && i < len(activePids); i++ {
		// Get the peer ID.
		pid := activePids[i]

		// Get the custody columns of the peer.
		peerCustodyColumns, err := s.custodyColumnsFromPeer(pid)
		if err != nil {
			return 0, errors.Wrap(err, "custody columns from peer")
		}

		// Compute the intersection of the peer custody columns and the remaining columns to request.
		peerRequestedColumns := make(map[uint64]bool, len(peerCustodyColumns))
		for column := range remainingColumnsToSample {
			if peerCustodyColumns[column] {
				peerRequestedColumns[column] = true
			}
		}

		// Remove the newsly requested columns from the remaining columns to request.
		for column := range peerRequestedColumns {
			delete(remainingColumnsToSample, column)
		}

		// Sample data columns from the peer.
		peerRetrievedColumns, err := s.sampleDataColumnsFromPeer(pid, peerRequestedColumns, root)
		if err != nil {
			return 0, errors.Wrap(err, "sample data columns from peer")
		}

		// Update the retrieved columns.
		retrievedColumnsCount += len(peerRetrievedColumns)
	}

	return uint64(retrievedColumnsCount), nil
}

// incrementalDAS samples data columns from active peers using incremental DAS.
// https://ethresear.ch/t/lossydas-lossy-incremental-and-diagonal-sampling-for-data-availability/18963#incrementaldas-dynamically-increase-the-sample-size-10
func (s *Service) incrementalDAS(root [fieldparams.RootLength]byte, sampleCount uint64) error {
	// Retrieve the number of columns.
	columnsCount := params.BeaconConfig().NumberOfColumns

	// Ramdomize all columns.
	columns := randomSlice(columnsCount)

	// Define the first column to sample.
	missingColumnsCount := uint64(0)

	firstColumnToSample, extendedSampleCount := uint64(0), peerdas.ExtendedSampleCount(sampleCount, 0)

	for i := 1; ; i++ {
		if extendedSampleCount > columnsCount {
			// We already tried to sample all columns, this is the unhappy path.
			log.WithField("root", fmt.Sprintf("%#x", root)).Warning("Some columns are still missing after sampling all possible columns")
			return nil
		}

		columnsToSample := columns[firstColumnToSample:extendedSampleCount]
		columnsToSampleCount := extendedSampleCount - firstColumnToSample

		retrievedSampleCount, err := s.sampleDataColumnsFromPeers(columnsToSample, root)
		if err != nil {
			return errors.Wrap(err, "sample data columns from peers")
		}

		if retrievedSampleCount == columnsToSampleCount {
			// All columns were correctly sampled, this is the happy path.
			log.WithFields(logrus.Fields{
				"root":         fmt.Sprintf("%#x", root),
				"roundsNeeded": i,
			}).Debug("All columns were successfully sampled")
			return nil
		}

		if retrievedSampleCount > columnsToSampleCount {
			// This should never happen.
			return errors.New("retrieved more columns than requested")
		}

		// Some columns are missing, we need to extend the sample size.
		missingColumnsCount += columnsToSampleCount - retrievedSampleCount

		firstColumnToSample = extendedSampleCount
		oldExtendedSampleCount := extendedSampleCount
		extendedSampleCount = peerdas.ExtendedSampleCount(sampleCount, missingColumnsCount)

		log.WithFields(logrus.Fields{
			"root":                fmt.Sprintf("%#x", root),
			"round":               i,
			"missingColumnsCount": missingColumnsCount,
			"currentSampleCount":  oldExtendedSampleCount,
			"nextSampleCount":     extendedSampleCount,
		}).Debug("Some columns are still missing after sampling this round.")
	}
}

// DataColumnSamplingRoutine runs incremental DAS on block when received.
func (s *Service) DataColumnSamplingRoutine(ctx context.Context) {
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

			// Sample data columns with incremental DAS.
			if err := s.incrementalDAS(data.BlockRoot, params.BeaconConfig().SamplesPerSlot); err != nil {
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
