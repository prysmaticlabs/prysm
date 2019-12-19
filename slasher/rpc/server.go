package rpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashpb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type Server struct {
	SlasherDB *db.Store
	ctx       context.Context
}

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (ss *Server) IsSlashableAttestation(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.AttesterSlashingResponse, error) {
	//TODO(#3133): add signature validation
	if err := ss.SlasherDB.SaveIndexedAttestation(req); err != nil {
		return nil, err
	}
	tEpoch := req.Data.Target.Epoch
	indices := req.AttestingIndices
	root, err := ssz.HashTreeRoot(req.Data)
	if err != nil {
		return nil, err
	}
	atsSlashinngRes := &slashpb.AttesterSlashingResponse{}
	at := make(chan []*ethpb.AttesterSlashing, len(indices))
	er := make(chan error, len(indices))
	var wg sync.WaitGroup
	lastIdx := int64(-1)
	for _, idx := range indices {
		if int64(idx) <= lastIdx {
			return nil, fmt.Errorf("indexed attestation contains repeated or non sorted ids")
		}
		wg.Add(1)
		go func(idx uint64) {
			atts, err := ss.SlasherDB.DoubleVotes(tEpoch, idx, root[:], req)
			if err != nil {
				er <- err
				wg.Done()
				return
			}
			if atts != nil && len(atts) > 0 {
				at <- atts
			}
			atts, err = ss.DetectSurroundVotes(ctx, idx, req)
			if err != nil {
				er <- err
				wg.Done()
				return
			}
			if atts != nil && len(atts) > 0 {
				at <- atts
			}
			wg.Done()
			return
		}(idx)
	}
	wg.Wait()
	close(er)
	close(at)
	for e := range er {
		err = fmt.Errorf(err.Error() + " : " + e.Error())
	}
	for atts := range at {
		atsSlashinngRes.AttesterSlashing = append(atsSlashinngRes.AttesterSlashing, atts...)
	}
	return atsSlashinngRes, err
}

// IsSlashableBlock returns a proposer slashing if the block header submitted is
// a slashable proposal.
func (ss *Server) IsSlashableBlock(ctx context.Context, psr *slashpb.ProposerSlashingRequest) (*slashpb.ProposerSlashingResponse, error) {
	//TODO(#3133): add signature validation
	epoch := helpers.SlotToEpoch(psr.BlockHeader.Slot)
	blockHeaders, err := ss.SlasherDB.BlockHeader(epoch, psr.ValidatorIndex)
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

// SlashableProposals is a subscription to receive all slashable proposer slashing events found by the watchtower.
func (ss *Server) SlashableProposals(req *types.Empty, server slashpb.Slasher_SlashableProposalsServer) error {
	//TODO(3133): implement stream provider for newly discovered listening to slashable proposals.
	return status.Error(codes.Unimplemented, "not implemented")
}

// SlashableAttestations is a subscription to receive all slashable attester slashing events found by the watchtower.
func (ss *Server) SlashableAttestations(req *types.Empty, server slashpb.Slasher_SlashableAttestationsServer) error {
	//TODO(3133): implement stream provider for newly discovered listening to slashable attestation.
	return status.Error(codes.Unimplemented, "not implemented")
}

// DetectSurroundVotes is a method used to return the attestation that were detected
// by min max surround detection method.
func (ss *Server) DetectSurroundVotes(ctx context.Context, validatorIdx uint64, req *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error) {
	spanMap, err := ss.SlasherDB.ValidatorSpansMap(validatorIdx)
	if err != nil {
		return nil, err
	}
	minTargetEpoch, spanMap, err := ss.DetectAndUpdateMinEpochSpan(ctx, req.Data.Source.Epoch, req.Data.Target.Epoch, validatorIdx, spanMap)
	if err != nil {
		return nil, err
	}
	maxTargetEpoch, spanMap, err := ss.DetectAndUpdateMaxEpochSpan(ctx, req.Data.Source.Epoch, req.Data.Target.Epoch, validatorIdx, spanMap)
	if err != nil {
		return nil, err
	}
	if err := ss.SlasherDB.SaveValidatorSpansMap(validatorIdx, spanMap); err != nil {
		return nil, err
	}
	var as []*ethpb.AttesterSlashing
	if minTargetEpoch > 0 {
		attestations, err := ss.SlasherDB.IndexedAttestation(minTargetEpoch, validatorIdx)
		if err != nil {
			return nil, err
		}
		for _, ia := range attestations {
			if ia.Data.Source.Epoch > req.Data.Source.Epoch && ia.Data.Target.Epoch < req.Data.Target.Epoch {
				as = append(as, &ethpb.AttesterSlashing{
					Attestation_1: req,
					Attestation_2: ia,
				})
			}
		}
	}
	if maxTargetEpoch > 0 {
		attestations, err := ss.SlasherDB.IndexedAttestation(maxTargetEpoch, validatorIdx)
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
