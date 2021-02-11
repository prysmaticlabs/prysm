package slasher

import (
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/sirupsen/logrus"
)

// Group a list of attestations into batches by validator chunk index.
// This way, we can detect on the batch of attestations for each validator chunk index
// concurrently, and also allowing us to effectively use a single 2D chunk
// for slashing detection through this logical grouping.
func (s *Service) groupByValidatorChunkIndex(
	attestations []*slashertypes.CompactAttestation,
) map[uint64][]*slashertypes.CompactAttestation {
	groupedAttestations := make(map[uint64][]*slashertypes.CompactAttestation)
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

// Group attestations by the chunk index their source epoch corresponds to.
func (s *Service) groupByChunkIndex(
	attestations []*slashertypes.CompactAttestation,
) map[uint64][]*slashertypes.CompactAttestation {
	attestationsByChunkIndex := make(map[uint64][]*slashertypes.CompactAttestation)
	for _, att := range attestations {
		chunkIdx := s.params.chunkIndex(types.Epoch(att.Source))
		attestationsByChunkIndex[chunkIdx] = append(attestationsByChunkIndex[chunkIdx], att)
	}
	return attestationsByChunkIndex
}

// Logs a slahing event with its particular details of the slashing
// itself as fields to our logger.
func logSlashingEvent(slashing *slashertypes.Slashing) {
	switch slashing.Kind {
	case slashertypes.DoubleVote:
		log.WithFields(logrus.Fields{
			"validatorIndex": slashing.ValidatorIndex,
			"targetEpoch":    slashing.TargetEpoch,
			"signingRoot":    fmt.Sprintf("%#x", slashing.SigningRoot),
		}).Info("Attester double vote")
	case slashertypes.SurroundingVote:
		log.WithFields(logrus.Fields{
			"validatorIndex":  slashing.ValidatorIndex,
			"prevSourceEpoch": slashing.PrevSourceEpoch,
			"prevTargetEpoch": slashing.PrevTargetEpoch,
			"sourceEpoch":     slashing.SourceEpoch,
			"targetEpoch":     slashing.TargetEpoch,
		}).Info("Attester surrounding vote")
	case slashertypes.SurroundedVote:
		log.WithFields(logrus.Fields{
			"validatorIndex":  slashing.ValidatorIndex,
			"prevSourceEpoch": slashing.PrevSourceEpoch,
			"prevTargetEpoch": slashing.PrevTargetEpoch,
			"sourceEpoch":     slashing.SourceEpoch,
			"targetEpoch":     slashing.TargetEpoch,
		}).Info("Attester surrounded vote")
	default:
		return
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
