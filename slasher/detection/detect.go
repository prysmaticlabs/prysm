package detection

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	status "github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"go.opencensus.io/trace"
)

func (ds *Service) detectAttesterSlashings(
	ctx context.Context,
	att *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "detection.detectAttesterSlashings")
	defer span.End()
	results, err := ds.minMaxSpanDetector.DetectSlashingsForAttestation(ctx, att)
	if err != nil {
		return nil, err
	}
	// If the response is nil, there was no slashing detected.
	if len(results) == 0 {
		return nil, nil
	}

	var slashings []*ethpb.AttesterSlashing
	for _, result := range results {
		var slashing *ethpb.AttesterSlashing
		switch result.Kind {
		case types.DoubleVote:
			slashing, err = ds.detectDoubleVote(ctx, att, result)
			if err != nil {
				return nil, errors.Wrap(err, "could not detect double votes on attestation")
			}
		case types.SurroundVote:
			slashing, err = ds.detectSurroundVotes(ctx, att, result)
			if err != nil {
				return nil, errors.Wrap(err, "could not detect surround votes on attestation")
			}
		}
		if slashing != nil {
			slashings = append(slashings, slashing)
		}
	}
	if err = ds.slasherDB.SaveAttesterSlashings(ctx, status.Active, slashings); err != nil {
		return nil, err
	}
	return slashings, nil
}

// detectDoubleVote cross references the passed in attestation with the bloom filter maintained
// for every epoch for the validator in order to determine if it is a double vote.
func (ds *Service) detectDoubleVote(
	ctx context.Context,
	incomingAtt *ethpb.IndexedAttestation,
	detectionResult *types.DetectionResult,
) (*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "detection.detectDoubleVote")
	defer span.End()
	if detectionResult == nil || detectionResult.Kind != types.DoubleVote {
		return nil, nil
	}

	otherAtts, err := ds.slasherDB.IndexedAttestationsWithPrefix(ctx, detectionResult.SlashableEpoch, detectionResult.SigBytes[:])
	if err != nil {
		return nil, err
	}
	for _, att := range otherAtts {
		if att.Data == nil {
			continue
		}
		// If there are no shared indices, there is no validator to slash.
		if len(sliceutil.IntersectionUint64(att.AttestingIndices, incomingAtt.AttestingIndices)) == 0 {
			continue
		}

		if isDoubleVote(incomingAtt, att) {
			doubleVotesDetected.Inc()
			return &ethpb.AttesterSlashing{
				Attestation_1: incomingAtt,
				Attestation_2: att,
			}, nil
		}
	}
	return nil, nil
}

// detectSurroundVotes cross references the passed in attestation with the requested validator's
// voting history in order to detect any possible surround votes.
func (ds *Service) detectSurroundVotes(
	ctx context.Context,
	incomingAtt *ethpb.IndexedAttestation,
	detectionResult *types.DetectionResult,
) (*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "detection.detectSurroundVotes")
	defer span.End()
	if detectionResult == nil || detectionResult.Kind != types.SurroundVote {
		return nil, nil
	}

	otherAtts, err := ds.slasherDB.IndexedAttestationsWithPrefix(ctx, detectionResult.SlashableEpoch, detectionResult.SigBytes[:])
	if err != nil {
		return nil, err
	}
	for _, att := range otherAtts {
		if att.Data == nil {
			continue
		}
		// If there are no shared indices, there is no validator to slash.
		if len(sliceutil.IntersectionUint64(att.AttestingIndices, incomingAtt.AttestingIndices)) == 0 {
			continue
		}

		// Slashings must be submitted as the incoming attestation surrounding the saved attestation.
		// So we swap the order if needed.
		if isSurrounding(incomingAtt, att) {
			surroundingVotesDetected.Inc()
			return &ethpb.AttesterSlashing{
				Attestation_1: incomingAtt,
				Attestation_2: att,
			}, nil
		} else if isSurrounded(incomingAtt, att) {
			surroundedVotesDetected.Inc()
			return &ethpb.AttesterSlashing{
				Attestation_1: att,
				Attestation_2: incomingAtt,
			}, nil
		}
	}
	return nil, errors.New("unexpected false positive in surround vote detection")
}

func isDoubleVote(incomingAtt *ethpb.IndexedAttestation, prevAtt *ethpb.IndexedAttestation) bool {
	return !proto.Equal(incomingAtt.Data, prevAtt.Data) && incomingAtt.Data.Target.Epoch == prevAtt.Data.Target.Epoch
}

func isSurrounding(incomingAtt *ethpb.IndexedAttestation, prevAtt *ethpb.IndexedAttestation) bool {
	return incomingAtt.Data.Source.Epoch < prevAtt.Data.Source.Epoch &&
		incomingAtt.Data.Target.Epoch > prevAtt.Data.Target.Epoch
}

func isSurrounded(incomingAtt *ethpb.IndexedAttestation, prevAtt *ethpb.IndexedAttestation) bool {
	return incomingAtt.Data.Source.Epoch > prevAtt.Data.Source.Epoch &&
		incomingAtt.Data.Target.Epoch < prevAtt.Data.Target.Epoch
}
