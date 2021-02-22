package slasher

import (
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/params"
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
		}).Info("Attester double vote slashing")
	case slashertypes.SurroundingVote:
		log.WithFields(logrus.Fields{
			"validatorIndex":  slashing.ValidatorIndex,
			"prevSourceEpoch": slashing.PrevSourceEpoch,
			"prevTargetEpoch": slashing.PrevTargetEpoch,
			"sourceEpoch":     slashing.SourceEpoch,
			"targetEpoch":     slashing.TargetEpoch,
		}).Info("Attester surrounding vote slashing")
	case slashertypes.SurroundedVote:
		log.WithFields(logrus.Fields{
			"validatorIndex":  slashing.ValidatorIndex,
			"prevSourceEpoch": slashing.PrevSourceEpoch,
			"prevTargetEpoch": slashing.PrevTargetEpoch,
			"sourceEpoch":     slashing.SourceEpoch,
			"targetEpoch":     slashing.TargetEpoch,
		}).Info("Attester surrounded vote slashing")
	case slashertypes.DoubleProposal:
		log.WithFields(logrus.Fields{
			"validatorIndex":  slashing.ValidatorIndex,
			"slot":            slashing.Slot,
			"prevSigningRoot": fmt.Sprintf("%#x", slashing.PrevSigningRoot),
			"signingRoot":     fmt.Sprintf("%#x", slashing.SigningRoot),
		}).Info("Proposer double proposal slashing")
	default:
		return
	}
}

// Log a double block proposal slashing given an incoming proposal and existing proposal signing root.
func logDoubleProposal(incomingProposal *slashertypes.CompactBeaconBlock, existingSigningRoot [32]byte) {
	logSlashingEvent(&slashertypes.Slashing{
		Kind:            slashertypes.DoubleProposal,
		ValidatorIndex:  types.ValidatorIndex(incomingProposal.ProposerIndex),
		SigningRoot:     incomingProposal.SigningRoot,
		PrevSigningRoot: existingSigningRoot,
		Slot:            incomingProposal.Slot,
	})
}

// If an existing signing root does not match an incoming proposal signing root,
// we then have a double block proposer slashing event.
func isDoubleProposal(incomingSigningRoot, existingSigningRoot [32]byte) bool {
	// If the existing signing root is the zero hash, we do not consider
	// this a double proposal.
	if existingSigningRoot == params.BeaconConfig().ZeroHash {
		return false
	}
	return incomingSigningRoot != existingSigningRoot
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
