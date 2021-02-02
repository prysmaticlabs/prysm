package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
)

func (s *Service) processQueuedAttestations(ctx context.Context) {
	secondsPerEpoch := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch
	ticker := slotutil.GetEpochTicker(s.genesisTime, secondsPerEpoch)
	defer ticker.Done()
	for {
		select {
		case currentEpoch := <-ticker.C():
			atts := s.attQueue.dequeue()
			log.Infof("Epoch %d reached, processing %d queued atts", currentEpoch, len(atts))
			if !validateAttestations(atts) {
				// TODO: Defer is ready at a future time.
				continue
			}
			// Group by validator index and process batches.
			// TODO: Perform concurrently with wait groups...?
			groupedAtts := s.groupByValidatorChunkIndex(atts)
			for validatorChunkIdx, attBatch := range groupedAtts {
				s.detectAttestationBatch(validatorChunkIdx, attBatch, currentEpoch)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) detectAttestationBatch(
	validatorChunkIdx uint64, atts []*ethpb.IndexedAttestation, currentEpoch types.Epoch,
) {

}

func (s *Service) groupByValidatorChunkIndex(
	attestations []*ethpb.IndexedAttestation,
) map[uint64][]*ethpb.IndexedAttestation {
	groupedAttestations := make(map[uint64][]*ethpb.IndexedAttestation)
	for _, att := range attestations {
		subqueueIndices := make(map[uint64]bool)
		for _, validatorIdx := range att.AttestingIndices {
			chunkIdx := s.params.validatorChunkIndex(types.ValidatorIndex(validatorIdx))
			subqueueIndices[chunkIdx] = true
		}
		for subqueueIdx := range subqueueIndices {
			groupedAttestations[subqueueIdx] = append(
				groupedAttestations[subqueueIdx],
				att,
			)
		}
	}
	return groupedAttestations
}
