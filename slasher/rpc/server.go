package rpc

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/slasher/db"
)

// Server defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type Server struct {
	SlasherDB *db.Store
	ctx       context.Context
}

// IsSlashableBlock returns a proposer slashing if the block header submitted is
// a slashable proposal.
func (ss *Server) IsSlashableBlock(ctx context.Context, psr *slashpb.ProposerSlashingRequest) (*slashpb.ProposerSlashingResponse, error) {
	//TODO(#3133): add signature validation
	epoch := helpers.SlotToEpoch(psr.BlockHeader.Header.Slot)
	blockHeaders, err := ss.SlasherDB.BlockHeaders(epoch, psr.ValidatorIndex)
	if err != nil {
		return nil, errors.Wrap(err, "slasher service error while trying to retrieve blocks")
	}
	pSlashingsResponse := &slashpb.ProposerSlashingResponse{}
	presentInDb := false
	for _, bh := range blockHeaders {
		if proto.Equal(bh, psr.BlockHeader) {
			presentInDb = true
			continue
		}
		pSlashingsResponse.ProposerSlashing = append(pSlashingsResponse.ProposerSlashing, &ethpb.ProposerSlashing{ProposerIndex: psr.ValidatorIndex, Header_1: psr.BlockHeader, Header_2: bh})
	}
	if len(pSlashingsResponse.ProposerSlashing) == 0 && !presentInDb {
		err = ss.SlasherDB.SaveBlockHeader(epoch, psr.ValidatorIndex, psr.BlockHeader)
		if err != nil {
			return nil, err
		}
	}
	return pSlashingsResponse, nil
}

// ProposerSlashings returns proposer slashings if slashing with the requested status are found in the db.
func (ss *Server) ProposerSlashings(ctx context.Context, st *slashpb.SlashingStatusRequest) (*slashpb.ProposerSlashingResponse, error) {
	pSlashingsResponse := &slashpb.ProposerSlashingResponse{}
	var err error
	pSlashingsResponse.ProposerSlashing, err = ss.SlasherDB.ProposalSlashingsByStatus(db.SlashingStatus(st.Status))
	if err != nil {
		return nil, err
	}
	return pSlashingsResponse, nil
}

// AttesterSlashings returns attester slashings if slashing with the requested status are found in the db.
func (ss *Server) AttesterSlashings(ctx context.Context, st *slashpb.SlashingStatusRequest) (*slashpb.AttesterSlashingResponse, error) {
	aSlashingsResponse := &slashpb.AttesterSlashingResponse{}
	var err error
	aSlashingsResponse.AttesterSlashing, err = ss.SlasherDB.AttesterSlashings(db.SlashingStatus(st.Status))
	if err != nil {
		return nil, err
	}
	return aSlashingsResponse, nil
}

// DetectSurroundVotes is a method used to return the attestation that were detected
// by min max surround detection method.
func (ss *Server) DetectSurroundVotes(ctx context.Context, validatorIdx uint64, req *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error) {
	spanMap, err := ss.SlasherDB.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validator spans map")
	}
	minTargetEpoch, spanMap, err := ss.DetectAndUpdateMinEpochSpan(ctx, req.Data.Source.Epoch, req.Data.Target.Epoch, validatorIdx, spanMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update min spans")
	}
	maxTargetEpoch, spanMap, err := ss.DetectAndUpdateMaxEpochSpan(ctx, req.Data.Source.Epoch, req.Data.Target.Epoch, validatorIdx, spanMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update max spans")
	}
	if err := ss.SlasherDB.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		return nil, errors.Wrap(err, "failed to save validator spans map")
	}

	var as []*ethpb.AttesterSlashing
	if minTargetEpoch > 0 {
		attestations, err := ss.SlasherDB.IdxAttsForTargetFromID(minTargetEpoch, validatorIdx)
		if err != nil {
			return nil, err
		}
		for _, ia := range attestations {
			if ia.Data == nil {
				continue
			}
			if ia.Data.Source.Epoch > req.Data.Source.Epoch && ia.Data.Target.Epoch < req.Data.Target.Epoch {
				as = append(as, &ethpb.AttesterSlashing{
					Attestation_1: req,
					Attestation_2: ia,
				})
			}
		}
	}
	if maxTargetEpoch > 0 {
		attestations, err := ss.SlasherDB.IdxAttsForTargetFromID(maxTargetEpoch, validatorIdx)
		if err != nil {
			return nil, err
		}
		for _, ia := range attestations {
			if ia.Data.Source.Epoch < req.Data.Source.Epoch && ia.Data.Target.Epoch > req.Data.Target.Epoch {
				as = append(as, &ethpb.AttesterSlashing{
					Attestation_1: req,
					Attestation_2: ia,
				})
			}
		}
	}
	return as, nil
}
