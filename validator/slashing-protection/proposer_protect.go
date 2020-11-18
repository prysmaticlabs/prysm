package slashingprotection

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	ErrSlashableBlock       = errors.New("attempted to sign a double proposal, block rejected by slashing protection")
	ErrRemoteSlashableBlock = errors.New("attempted a double proposal, block rejected by remote slashing protection")
)

func (s *Service) IsSlashableBlock(
	ctx context.Context, header *ethpb.BeaconBlockHeader, pubKey [48]byte,
) error {
	signingRoot, err := s.validatorDB.ProposalHistoryForSlot(ctx, pubKey[:], header.Slot)
	if err != nil {
		return errors.Wrap(err, "failed to get proposal history")
	}
	// If the bit for the current slot is marked, do not propose.
	if !bytes.Equal(signingRoot, params.BeaconConfig().ZeroHash[:]) {
		//		if v.emitAccountMetrics {
		//			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		//		}
		return ErrSlashableBlock
	}

	if s.remoteProtector != nil {
		if !s.remoteProtector.IsSlashableBlock(ctx, header) {
			//if v.emitAccountMetrics {
			//	ValidatorProposeFailVecSlasher.WithLabelValues(fmtKey).Inc()
			//}
			return ErrRemoteSlashableBlock
		}
	}
	return nil
}

func (s *Service) CommitBlock(ctx context.Context, pubKey [48]byte, block *ethpb.SignedBeaconBlock, domain *ethpb.DomainResponse) error {
	signingRoot, err := helpers.ComputeSigningRoot(block.Block, domain.SignatureDomain)
	if err != nil {
		//if v.emitAccountMetrics {
		//	ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		//}
		return errors.Wrap(err, "failed to compute signing root for block")
	}
	if err := s.validatorDB.SaveProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot, signingRoot[:]); err != nil {
		//if v.emitAccountMetrics {
		//	ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		//}
		return errors.Wrap(err, "failed to save updated proposal history")
	}
	if s.remoteProtector != nil {
		sbh, err := blockutil.SignedBeaconBlockHeaderFromBlock(block)
		if err != nil {
			return errors.Wrap(err, "failed to get block header from block")
		}
		valid, err := s.remoteProtector.CommitBlock(ctx, sbh)
		if err != nil {
			return err
		}
		if !valid {
			//if v.emitAccountMetrics {
			//	ValidatorProposeFailVecSlasher.WithLabelValues(fmtKey).Inc()
			//}
			return ErrRemoteSlashableBlock
		}
	}
	return nil
}
