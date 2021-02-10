package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/sirupsen/logrus"
)

// Process queued attestations every time an epoch ticker fires. We retrieve
// these attestations from a queue, then group them all by validator chunk index.
// This grouping will allow us to perform detection on batches of attestations
// per validator chunk index which can be done concurrently.
func (s *Service) processQueuedAttestations(ctx context.Context, epochTicker <-chan uint64) {
	for {
		select {
		case currentEpoch := <-epochTicker:
			s.queueLock.Lock()
			atts := s.attestationQueue
			s.attestationQueue = make([]*slashertypes.CompactAttestation, 0)
			s.queueLock.Unlock()
			log.WithFields(logrus.Fields{
				"currentEpoch": currentEpoch,
				"numAtts":      len(atts),
			}).Info("Epoch reached, processing queued atts for slashing detection")
			groupedAtts := s.groupByValidatorChunkIndex(atts)
			for validatorChunkIdx, attsBatch := range groupedAtts {
				s.detectAttestationBatch(attsBatch, validatorChunkIdx, types.Epoch(currentEpoch))
			}
		case <-ctx.Done():
			return
		}
	}
}

// Given a list of attestations all corresponding to a validator chunk index as well
// as the current epoch in time, we perform slashing detection over the batch.
// TODO(#8331): Implement.
func (s *Service) detectAttestationBatch(
	atts []*slashertypes.CompactAttestation, validatorChunkIndex uint64, currentEpoch types.Epoch,
) {

}
