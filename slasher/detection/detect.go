package detection

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations"
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
		doubleAttSlashings, err := ds.detectDoubleVotes(ctx, att)
		if err != nil {
			return nil, errors.Wrap(err, "could not detect double votes on attestation")
		}
		newSlashings := append(surroundedAttSlashings, doubleAttSlashings...)
		slashings = append(slashings, newSlashings...)
	}
	return slashings, nil
}

// detectDoubleVote --
// TODO(#4589): Implement.
func (ds *Service) detectDoubleVotes(
	ctx context.Context,
	att *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	return nil, nil
}

// detectSurroundVotes --
func (ds *Service) detectSurroundVotes(
	ctx context.Context,
	validatorIdx uint64,
	incomingAtt *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	if updateErr := ds.minMaxSpanDetector.UpdateSpans(ctx, incomingAtt); updateErr != nil {
		return nil, updateErr
	}
	res, err := ds.minMaxSpanDetector.DetectSlashingForValidator(
		ctx,
		validatorIdx,
		incomingAtt.Data.Source.Epoch,
		incomingAtt.Data.Target.Epoch,
	)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	if res.Kind != attestations.SurroundVote {
		return nil, nil
	}
	if res.SlashableEpoch == 0 {
		return nil, nil
	}
	var slashings []*ethpb.AttesterSlashing
	otherAtts, err := ds.slasherDB.IndexedAttestations(ctx, res.SlashableEpoch)
	if err != nil {
		return nil, err
	}
	for _, att := range otherAtts {
		if att.Data == nil {
			continue
		}
		if isSurrounding(incomingAtt, att) || isSurrounded(incomingAtt, att) {
			slashings = append(slashings, &ethpb.AttesterSlashing{
				Attestation_1: incomingAtt,
				Attestation_2: att,
			})
		}
	}
	return slashings, nil
}

func isSurrounding(att1 *ethpb.IndexedAttestation, att2 *ethpb.IndexedAttestation) bool {
	return att1.Data.Source.Epoch < att2.Data.Source.Epoch && att1.Data.Target.Epoch > att2.Data.Target.Epoch
}

func isSurrounded(att1 *ethpb.IndexedAttestation, att2 *ethpb.IndexedAttestation) bool {
	return att1.Data.Source.Epoch < att2.Data.Source.Epoch && att1.Data.Target.Epoch > att2.Data.Target.Epoch
}
