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
	ticker := slotutil.NewEpochTicker(s.genesisTime, secondsPerEpoch)
	defer ticker.Done()
	for {
		select {
		case currentEpoch := <-ticker.C():
			atts := s.attestationQueue
			s.attestationQueue = make([]*ethpb.IndexedAttestation, 0)
			// TODO(Raul): Perform validation of attestations before
			// performing batch detection, such as checking for slot time requirements
			// and the prerequisite that source must be less than epoch.
			log.Infof("Epoch %d reached, processing %d queued atts", currentEpoch, len(atts))
			groupedAtts := s.groupByValidatorChunkIndex(atts)
			for validatorChunkIdx, attsBatch := range groupedAtts {
				s.detectAttestationBatch(attsBatch, validatorChunkIdx, types.Epoch(currentEpoch))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) detectAttestationBatch(
	atts []*ethpb.IndexedAttestation, validatorChunkIndex uint64, currentEpoch types.Epoch,
) {

}

func (s *Service) groupByValidatorChunkIndex(
	attestations []*ethpb.IndexedAttestation,
) map[uint64][]*ethpb.IndexedAttestation {
	groupedAttestations := make(map[uint64][]*ethpb.IndexedAttestation)
	for _, att := range attestations {
		validatorChunkIndices := make(map[uint64]bool)
		for _, validatorIdx := range att.AttestingIndices {
			validatorChunkIndex := s.params.validatorChunkIndex(types.ValidatorIndex(validatorIdx))
			validatorChunkIndices[validatorChunkIndex] = true
		}
		for validatorChunkIndex := range validatorChunkIndices {
			groupedAttestations[validatorChunkIndex] = append(
				groupedAttestations[validatorChunkIndex],
				att,
			)
		}
	}
	return groupedAttestations
}
