package detection

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

func (ds *Service) detectAttesterSlashings(
	ctx context.Context,
	att *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	slashings := make([]*ethpb.AttesterSlashing, 0)
	for i := 0; i < len(att.AttestingIndices); i++ {
		valIdx := att.AttestingIndices[i]
		surroundedAttSlashings, err := ds.detectSurroundVotes(ctx, valIdx, att)
		if err != nil {
			return nil, errors.Wrap(err, "could not detect surround votes on attestation")
		}
		doubleAttSlashings, err := ds.detectDoubleVotes(ctx, valIdx, att)
		if err != nil {
			return nil, errors.Wrap(err, "could not detect double votes on attestation")
		}
		if len(surroundedAttSlashings) > 0 {
			log.Infof("Found %d slashings for val idx %d", len(surroundedAttSlashings), valIdx)
		}
		newSlashings := append(surroundedAttSlashings, doubleAttSlashings...)
		slashings = append(slashings, newSlashings...)
	}
	return slashings, nil
}

// detectDoubleVote cross references the passed in attestation with the bloom filter maintained
// for every epoch for the validator in order to determine if it is a double vote.
func (ds *Service) detectDoubleVotes(
	ctx context.Context,
	validatorIdx uint64,
	incomingAtt *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	res, err := ds.minMaxSpanDetector.DetectSlashingForValidator(
		ctx,
		validatorIdx,
		incomingAtt.Data,
	)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	if res.Kind != types.DoubleVote {
		return nil, nil
	}

	var slashings []*ethpb.AttesterSlashing
	otherAtts, err := ds.slasherDB.IndexedAttestationsForEpoch(ctx, res.SlashableEpoch)
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

		if isSurrounding(incomingAtt, att) || isSurrounded(incomingAtt, att) {
			slashings = append(slashings, &ethpb.AttesterSlashing{
				Attestation_1: att,
				Attestation_2: incomingAtt,
			})
		}
	}
	if len(slashings) == 0 {
		return nil, errors.New("unexpected false positive in surround vote detection")
	}
	return slashings, nil
}

// detectSurroundVotes cross references the passed in attestation with the requested validator's
// voting history in order to detect any possible surround votes.
func (ds *Service) detectSurroundVotes(
	ctx context.Context,
	validatorIdx uint64,
	incomingAtt *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	res, err := ds.minMaxSpanDetector.DetectSlashingForValidator(
		ctx,
		validatorIdx,
		incomingAtt.Data,
	)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	if res.Kind != types.SurroundVote {
		return nil, nil
	}

	var slashings []*ethpb.AttesterSlashing
	otherAtts, err := ds.slasherDB.IndexedAttestationsForEpoch(ctx, res.SlashableEpoch)
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

		if isSurrounding(incomingAtt, att) {
			slashings = append(slashings, &ethpb.AttesterSlashing{
				Attestation_1: incomingAtt,
				Attestation_2: att,
			})
		} else if isSurrounded(incomingAtt, att) {
			slashings = append(slashings, &ethpb.AttesterSlashing{
				Attestation_1: att,
				Attestation_2: incomingAtt,
			})
		}
	}
	if len(slashings) == 0 {
		return nil, errors.New("unexpected false positive in surround vote detection")
	}
	return slashings, nil
}

func isDoubleVote(incomingAtt *ethpb.IndexedAttestation, prevAtt *ethpb.IndexedAttestation) bool {
	return proto.Equal(incomingAtt.Data, prevAtt.Data) && incomingAtt.Data.Target.Epoch == prevAtt.Data.Target.Epoch
}

func isSurrounding(incomingAtt *ethpb.IndexedAttestation, prevAtt *ethpb.IndexedAttestation) bool {
	return incomingAtt.Data.Source.Epoch < prevAtt.Data.Source.Epoch &&
		incomingAtt.Data.Target.Epoch > prevAtt.Data.Target.Epoch
}

func isSurrounded(incomingAtt *ethpb.IndexedAttestation, prevAtt *ethpb.IndexedAttestation) bool {
	return incomingAtt.Data.Source.Epoch > prevAtt.Data.Source.Epoch &&
		incomingAtt.Data.Target.Epoch < prevAtt.Data.Target.Epoch
}
