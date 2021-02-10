package slasher

import (
	"context"

	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

// Receive indexed attestations from some source event feed,
// validating their integrity before appending them to an attestation queue
// for batch processing in a separate routine.
func (s *Service) receiveAttestations(ctx context.Context) {
	sub := s.serviceCfg.IndexedAttsFeed.Subscribe(s.indexedAttsChan)
	defer close(s.indexedAttsChan)
	defer sub.Unsubscribe()
	for {
		select {
		case att := <-s.indexedAttsChan:
			// TODO(#8331): Defer attestations from the future for later processing.
			if !validateAttestationIntegrity(att) {
				continue
			}
			compactAtt := &slashertypes.CompactAttestation{
				AttestingIndices: att.AttestingIndices,
				Source:           att.Data.Source.Epoch,
				Target:           att.Data.Target.Epoch,
			}
			s.queueLock.Lock()
			s.attestationQueue = append(s.attestationQueue, compactAtt)
			s.queueLock.Unlock()
		case err := <-sub.Err():
			log.WithError(err).Debug("Subscriber closed with error")
			return
		case <-ctx.Done():
			return
		}
	}
}
