package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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

// Validates the attestation data integrity, ensuring we have no nil values for
// source, epoch, and that the source epoch of the attestation must be less than
// the target epoch, which is a precondition for performing slashing detection.
func validateAttestationIntegrity(att *ethpb.IndexedAttestation) bool {
	if att == nil || att.Data == nil || att.Data.Source == nil || att.Data.Target == nil {
		return false
	}
	return att.Data.Source.Epoch < att.Data.Target.Epoch
}
