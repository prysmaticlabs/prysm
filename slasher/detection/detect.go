package detection

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	status "github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"go.opencensus.io/trace"
)

// DetectAttesterSlashings detects double, surround and surrounding attestation offences given an attestation.
func (ds *Service) DetectAttesterSlashings(
	ctx context.Context,
	att *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "detection.DetectAttesterSlashings")
	defer span.End()
	results, err := ds.minMaxSpanDetector.DetectSlashingsForAttestation(ctx, att)
	if err != nil {
		return nil, err
	}
	// If the response is nil, there was no slashing detected.
	if len(results) == 0 {
		return nil, nil
	}

	resultsToAtts, err := ds.mapResultsToAtts(ctx, results)
	if err != nil {
		return nil, err
	}

	var slashings []*ethpb.AttesterSlashing
	for _, result := range results {
		resultKey := resultHash(result)
		var slashing *ethpb.AttesterSlashing
		switch result.Kind {
		case types.DoubleVote:
			slashing, err = ds.detectDoubleVote(ctx, resultsToAtts[resultKey], att, result)
			if err != nil {
				return nil, errors.Wrap(err, "could not detect double votes on attestation")
			}
		case types.SurroundVote:
			slashing, err = ds.detectSurroundVotes(ctx, resultsToAtts[resultKey], att, result)
			if err != nil {
				return nil, errors.Wrap(err, "could not detect surround votes on attestation")
			}
		}
		if slashing != nil {
			slashings = append(slashings, slashing)
		}
	}

	// Clear out any duplicate results.
	keys := make(map[[32]byte]bool)
	var slashingList []*ethpb.AttesterSlashing
	for _, ss := range slashings {
		hash, err := hashutil.HashProto(ss)
		if err != nil {
			return nil, errors.Wrap(err, "could not hash slashing")
		}
		if _, value := keys[hash]; !value {
			keys[hash] = true
			slashingList = append(slashingList, ss)
		}
	}
	if len(slashings) > 0 {
		if err = ds.slasherDB.SaveAttesterSlashings(ctx, status.Active, slashings); err != nil {
			return nil, err
		}
	}
	return slashingList, nil
}

// UpdateSpans passthrough function that updates span maps given an indexed attestation.
func (ds *Service) UpdateSpans(ctx context.Context, att *ethpb.IndexedAttestation) error {
	return ds.minMaxSpanDetector.UpdateSpans(ctx, att)
}

// detectDoubleVote cross references the passed in attestation with the bloom filter maintained
// for every epoch for the validator in order to determine if it is a double vote.
func (ds *Service) detectDoubleVote(
	ctx context.Context,
	possibleAtts []*ethpb.IndexedAttestation,
	incomingAtt *ethpb.IndexedAttestation,
	detectionResult *types.DetectionResult,
) (*ethpb.AttesterSlashing, error) {
	if detectionResult == nil || detectionResult.Kind != types.DoubleVote {
		return nil, nil
	}

	for _, att := range possibleAtts {
		if att.Data == nil {
			continue
		}

		if !isDoubleVote(incomingAtt, att) {
			continue
		}

		// If there are no shared indices, there is no validator to slash.
		if !sliceutil.IsInUint64(detectionResult.ValidatorIndex, att.AttestingIndices) {
			continue
		}

		doubleVotesDetected.Inc()
		return &ethpb.AttesterSlashing{
			Attestation_1: incomingAtt,
			Attestation_2: att,
		}, nil
	}
	return nil, nil
}

// detectSurroundVotes cross references the passed in attestation with the requested validator's
// voting history in order to detect any possible surround votes.
func (ds *Service) detectSurroundVotes(
	ctx context.Context,
	possibleAtts []*ethpb.IndexedAttestation,
	incomingAtt *ethpb.IndexedAttestation,
	detectionResult *types.DetectionResult,
) (*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "detection.detectSurroundVotes")
	defer span.End()
	if detectionResult == nil || detectionResult.Kind != types.SurroundVote {
		return nil, nil
	}

	for _, att := range possibleAtts {
		if att.Data == nil {
			continue
		}
		isSurround := isSurrounding(incomingAtt, att)
		isSurrounded := isSurrounding(att, incomingAtt)
		if !isSurround && !isSurrounded {
			continue
		}
		// If there are no shared indices, there is no validator to slash.
		if !sliceutil.IsInUint64(detectionResult.ValidatorIndex, att.AttestingIndices) {
			continue
		}

		// Slashings must be submitted as the incoming attestation surrounding the saved attestation.
		// So we swap the order if needed.
		if isSurround {
			surroundingVotesDetected.Inc()
			return &ethpb.AttesterSlashing{
				Attestation_1: incomingAtt,
				Attestation_2: att,
			}, nil
		} else if isSurrounded {
			surroundedVotesDetected.Inc()
			return &ethpb.AttesterSlashing{
				Attestation_1: att,
				Attestation_2: incomingAtt,
			}, nil
		}
	}
	return nil, errors.New("unexpected false positive in surround vote detection")
}

// DetectDoubleProposals checks if the given signed beacon block is a slashable offense and returns the slashing.
func (ds *Service) DetectDoubleProposals(ctx context.Context, incomingBlock *ethpb.SignedBeaconBlockHeader) (*ethpb.ProposerSlashing, error) {
	return ds.proposalsDetector.DetectDoublePropose(ctx, incomingBlock)
}

// DetectDoubleProposeNoUpdate checks if the given beacon block header is a slashable offense.
func (ds *Service) DetectDoubleProposeNoUpdate(ctx context.Context, incomingBlock *ethpb.BeaconBlockHeader) (bool, error) {
	return ds.proposalsDetector.DetectDoubleProposeNoUpdate(ctx, incomingBlock)
}

// mapResultsToAtts handles any duplicate detections by ensuring they reuse the same pool of attestations, instead of re-checking the DB for the same data.
func (ds *Service) mapResultsToAtts(ctx context.Context, results []*types.DetectionResult) (map[[32]byte][]*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "detection.mapResultsToAtts")
	defer span.End()
	resultsToAtts := make(map[[32]byte][]*ethpb.IndexedAttestation)
	for _, result := range results {
		resultKey := resultHash(result)
		if _, ok := resultsToAtts[resultKey]; ok {
			continue
		}
		matchingAtts, err := ds.slasherDB.IndexedAttestationsWithPrefix(ctx, result.SlashableEpoch, result.SigBytes[:])
		if err != nil {
			return nil, err
		}
		resultsToAtts[resultKey] = matchingAtts
	}
	return resultsToAtts, nil
}

func resultHash(result *types.DetectionResult) [32]byte {
	resultBytes := append(bytesutil.Bytes8(result.SlashableEpoch), result.SigBytes[:]...)
	return hashutil.Hash(resultBytes)
}

func isDoublePropose(
	incomingBlockHeader *ethpb.SignedBeaconBlockHeader,
	prevBlockHeader *ethpb.SignedBeaconBlockHeader,
) bool {
	return incomingBlockHeader.Header.ProposerIndex == prevBlockHeader.Header.ProposerIndex &&
		!bytes.Equal(incomingBlockHeader.Signature, prevBlockHeader.Signature) &&
		incomingBlockHeader.Header.Slot == prevBlockHeader.Header.Slot
}

func isDoubleVote(incomingAtt *ethpb.IndexedAttestation, prevAtt *ethpb.IndexedAttestation) bool {
	return !attestationutil.AttDataIsEqual(incomingAtt.Data, prevAtt.Data) && incomingAtt.Data.Target.Epoch == prevAtt.Data.Target.Epoch
}

func isSurrounding(incomingAtt *ethpb.IndexedAttestation, prevAtt *ethpb.IndexedAttestation) bool {
	return incomingAtt.Data.Source.Epoch < prevAtt.Data.Source.Epoch &&
		incomingAtt.Data.Target.Epoch > prevAtt.Data.Target.Epoch
}
