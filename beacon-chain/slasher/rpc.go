package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

// IsSlashableBlock checks if an input block header is slashable
// with respect to historical block proposal data.
func (s *Service) IsSlashableBlock(
	ctx context.Context, block *ethpb.SignedBeaconBlockHeader,
) (*ethpb.ProposerSlashing, error) {
	dataRoot, err := block.Header.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	signedBlockWrapper := &slashertypes.SignedBlockHeaderWrapper{
		SignedBeaconBlockHeader: block,
		SigningRoot:             dataRoot,
	}
	proposerSlashings, err := s.detectProposerSlashings(ctx, []*slashertypes.SignedBlockHeaderWrapper{signedBlockWrapper})
	if err != nil {
		return nil, err
	}
	if len(proposerSlashings) == 0 {
		return nil, nil
	}
	return proposerSlashings[0], nil
}

// IsSlashableAttestation checks if an input indexed attestation is slashable
// with respect to historical attestation data.
func (s *Service) IsSlashableAttestation(
	ctx context.Context, attestation *ethpb.IndexedAttestation,
) ([]*ethpb.AttesterSlashing, error) {
	dataRoot, err := attestation.Data.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	indexedAttWrapper := &slashertypes.IndexedAttestationWrapper{
		IndexedAttestation: attestation,
		SigningRoot:        dataRoot,
	}
	attesterSlashings, err := s.checkSlashableAttestations(ctx, []*slashertypes.IndexedAttestationWrapper{indexedAttWrapper})
	if err != nil {
		return nil, err
	}
	if len(attesterSlashings) == 0 {
		return nil, nil
	}
	return attesterSlashings, nil
}
