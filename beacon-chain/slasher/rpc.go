package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

// IsSlashableBlock comapres the given signed proposal to the slasher database and returns any double block
// proposals that are considered slashable.
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

// IsSlashableAttestation comapres the given indexed attestation to the slasher database and returns any
// slashing proofs if they exist.
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
	attesterSlashings, err := s.CheckSlashableAttestations(ctx, []*slashertypes.IndexedAttestationWrapper{indexedAttWrapper})
	if err != nil {
		return nil, err
	}
	if len(attesterSlashings) == 0 {
		return nil, nil
	}
	return attesterSlashings, nil
}
