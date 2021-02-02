package slasher

import "context"

func (s *Service) receiveAttestations(ctx context.Context) {
	sub := s.serviceCfg.IndexedAttsFeed.Subscribe(s.indexedAttsChan)
	defer close(s.indexedAttsChan)
	defer sub.Unsubscribe()
	for {
		select {
		case att := <-s.indexedAttsChan:
			log.Infof("Got attestation with indices %v", att.AttestingIndices)
			groupedAtts := s.groupByValidatorChunkIndex(atts)
			for subqueueIdx, atts := range groupedAtts {
				s.detectAttestationBatch(subqueueIdx, atts, currentEpoch)
			}
		case err := <-sub.Err():
			log.WithError(err).Debug("Subscriber closed with error")
			return
		case <-ctx.Done():
			return
		}
	}
}
