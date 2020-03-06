package detection

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

func (ds *Service) detectAttesterSlashings(
	ctx context.Context,
	att *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "detection.detectAttesterSlashings")
	defer span.End()
	slashings := make([]*ethpb.AttesterSlashing, 0)
	for i := 0; i < len(att.AttestingIndices); i++ {
		valIdx := att.AttestingIndices[i]
		result, err := ds.minMaxSpanDetector.DetectSlashingForValidator(ctx, valIdx, att.Data)
		if err != nil {
			return nil, err
		}
		// If the response is nil, there was no slashing detected.
		if result == nil {
			continue
		}

		var slashing *ethpb.AttesterSlashing
		switch result.Kind {
		case types.DoubleVote:
			slashing, err = ds.detectDoubleVote(ctx, att, result)
			logrus.Debugf("Detected a possible double vote for val idx %d", valIdx)
			if err != nil {
				return nil, errors.Wrap(err, "could not detect double votes on attestation")
			}
		case types.SurroundVote:
			logrus.Debugf("Detected a possible surround vote for val idx %d", valIdx)
			slashing, err = ds.detectSurroundVotes(ctx, att, result)
			if err != nil {
				return nil, errors.Wrap(err, "could not detect surround votes on attestation")
			}
		}
		slashings = append(slashings, slashing)
	}

	// Clear out any duplicate slashings.
	keys := make(map[[32]byte]bool)
	var slashingList []*ethpb.AttesterSlashing
	for _, ss := range slashings {
		hash, err := hashutil.HashProto(ss)
		if err != nil {
			return nil, err
		}
		if _, value := keys[hash]; !value {
			keys[hash] = true
			slashingList = append(slashingList, ss)
		}
	}

	return slashingList, nil
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
			return &ethpb.AttesterSlashing{
				Attestation_1: incomingAtt,
				Attestation_2: att,
			}, nil
		} else if isSurrounded(incomingAtt, att) {
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
