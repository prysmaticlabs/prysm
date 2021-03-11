package slasher

import (
	"fmt"
	"strconv"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

// Group a list of attestations into batches by validator chunk index.
// This way, we can detect on the batch of attestations for each validator chunk index
// concurrently, and also allowing us to effectively use a single 2D chunk
// for slashing detection through this logical grouping.
func (s *Service) groupByValidatorChunkIndex(
	attestations []*slashertypes.IndexedAttestationWrapper,
) map[uint64][]*slashertypes.IndexedAttestationWrapper {
	groupedAttestations := make(map[uint64][]*slashertypes.IndexedAttestationWrapper)
	for _, att := range attestations {
		validatorChunkIndices := make(map[uint64]bool)
		for _, validatorIdx := range att.IndexedAttestation.AttestingIndices {
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
	attestations []*slashertypes.IndexedAttestationWrapper,
) map[uint64][]*slashertypes.IndexedAttestationWrapper {
	attestationsByChunkIndex := make(map[uint64][]*slashertypes.IndexedAttestationWrapper)
	for _, att := range attestations {
		chunkIdx := s.params.chunkIndex(att.IndexedAttestation.Data.Source.Epoch)
		attestationsByChunkIndex[chunkIdx] = append(attestationsByChunkIndex[chunkIdx], att)
	}
	return attestationsByChunkIndex
}

// Validates the attestation data integrity, ensuring we have no nil values for
// source, epoch, and that the source epoch of the attestation must be less than
// the target epoch, which is a precondition for performing slashing detection.
// This function also checks the attestation source epoch is within the history size
// we keep track of for slashing detection.
// This function returns a list of valid attestations, a list of attestations that are
// valid in the future, and the number of attestations dropped.
func (s *Service) validateAttestationIntegrity(
	atts []*slashertypes.IndexedAttestationWrapper, currentEpoch types.Epoch,
) (valid, validInFuture []*slashertypes.IndexedAttestationWrapper, numDropped int) {
	valid = make([]*slashertypes.IndexedAttestationWrapper, 0, len(atts))
	validInFuture = make([]*slashertypes.IndexedAttestationWrapper, 0, len(atts))

	for _, attWrapper := range atts {
		// If an attestation is malformed, we drop it.
		if attWrapper == nil ||
			attWrapper.IndexedAttestation == nil ||
			attWrapper.IndexedAttestation.Data == nil ||
			attWrapper.IndexedAttestation.Data.Source == nil ||
			attWrapper.IndexedAttestation.Data.Target == nil {
			numDropped++
			continue
		}

		// All valid attestations cannot have source epoch > target epoch.
		sourceEpoch := attWrapper.IndexedAttestation.Data.Source.Epoch
		targetEpoch := attWrapper.IndexedAttestation.Data.Target.Epoch
		if sourceEpoch > targetEpoch {
			numDropped++
			continue
		}

		// If an attestation's source is epoch is older than the max history length
		// we keep track of for slashing detection, we drop it.
		if sourceEpoch+types.Epoch(s.params.historyLength) <= currentEpoch {
			numDropped++
			continue
		}

		// If an attestations's target epoch is in the future, we defer processing for later.
		if targetEpoch > currentEpoch {
			validInFuture = append(validInFuture, attWrapper)
		} else {
			valid = append(valid, attWrapper)
		}
	}
	return
}

// Logs a slahing event with its particular details of the slashing
// itself as fields to our logger.
func logSlashingEvent(slashing *slashertypes.Slashing) {
	switch slashing.Kind {
	case slashertypes.DoubleVote:
		log.WithFields(logrus.Fields{
			"validatorIndex":  slashing.ValidatorIndex,
			"targetEpoch":     slashing.TargetEpoch,
			"signingRoot":     fmt.Sprintf("%#x", slashing.SigningRoot),
			"prevSigningRoot": fmt.Sprintf("%#x", slashing.PrevSigningRoot),
		}).Info("Attester double vote slashing")
	case slashertypes.SurroundingVote:
		log.WithFields(logrus.Fields{
			"validatorIndex":  slashing.ValidatorIndex,
			"prevSourceEpoch": slashing.PrevAttestation.Data.Source.Epoch,
			"prevTargetEpoch": slashing.PrevAttestation.Data.Target.Epoch,
			"sourceEpoch":     slashing.Attestation.Data.Source.Epoch,
			"targetEpoch":     slashing.Attestation.Data.Target.Epoch,
		}).Info("Attester surrounding vote slashing")
	case slashertypes.SurroundedVote:
		log.WithFields(logrus.Fields{
			"validatorIndex":  slashing.ValidatorIndex,
			"prevSourceEpoch": slashing.PrevAttestation.Data.Source.Epoch,
			"prevTargetEpoch": slashing.PrevAttestation.Data.Target.Epoch,
			"sourceEpoch":     slashing.Attestation.Data.Source.Epoch,
			"targetEpoch":     slashing.Attestation.Data.Target.Epoch,
		}).Info("Attester surrounded vote slashing")
	case slashertypes.DoubleProposal:
		log.WithFields(logrus.Fields{
			"validatorIndex":  slashing.ValidatorIndex,
			"slot":            slashing.BeaconBlock.Header.Slot,
			"prevSigningRoot": fmt.Sprintf("%#x", slashing.PrevSigningRoot),
			"signingRoot":     fmt.Sprintf("%#x", slashing.SigningRoot),
		}).Info("Proposer double proposal slashing")
	default:
		return
	}
}

// Log a double block proposal slashing given an incoming proposal and existing proposal signing root.
func logDoubleProposal(incomingProposal, existingProposal *slashertypes.SignedBlockHeaderWrapper) {
	logSlashingEvent(&slashertypes.Slashing{
		Kind:            slashertypes.DoubleProposal,
		ValidatorIndex:  incomingProposal.SignedBeaconBlockHeader.Header.ProposerIndex,
		PrevSigningRoot: existingProposal.SigningRoot,
		SigningRoot:     incomingProposal.SigningRoot,
		PrevBeaconBlock: existingProposal.SignedBeaconBlockHeader,
		BeaconBlock:     incomingProposal.SignedBeaconBlockHeader,
	})
}

// Turns a uint64 value to a string representation.
func uintToString(val uint64) string {
	return strconv.FormatUint(val, 10)
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
