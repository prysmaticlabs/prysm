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
		case <-sub.Err():
			return
		case <-ctx.Done():
			return
		}
	}
}
