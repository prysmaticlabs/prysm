package rpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations"
	log "github.com/sirupsen/logrus"
)

// Server defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type Server struct {
	SlasherDB *kv.Store
	ctx       context.Context
}

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (ss *Server) IsSlashableAttestation(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.AttesterSlashingResponse, error) {
	//TODO(#3133): add signature validation
	if req.Data == nil {
		return nil, fmt.Errorf("cant hash nil data in indexed attestation")
	}
	if err := ss.SlasherDB.SaveIndexedAttestation(ctx, req); err != nil {
		return nil, err
	}
	indices := req.AttestingIndices
	root, err := hashutil.HashProto(req.Data)
	if err != nil {
		return nil, err
	}
	attSlashingResp := &slashpb.AttesterSlashingResponse{}
	attSlashings := make(chan []*ethpb.AttesterSlashing, len(indices))
	errorChans := make(chan error, len(indices))
	var wg sync.WaitGroup
	lastIdx := int64(-1)
	for _, idx := range indices {
		if int64(idx) <= lastIdx {
			return nil, fmt.Errorf("indexed attestation contains repeated or non sorted ids")
		}
		wg.Add(1)
		go func(idx uint64, root [32]byte, req *ethpb.IndexedAttestation) {
			atts, err := ss.SlasherDB.DoubleVotes(ctx, idx, root[:], req)
			if err != nil {
				errorChans <- err
				wg.Done()
				return
			}
			if atts != nil && len(atts) > 0 {
				attSlashings <- atts
			}
			atts, err = ss.DetectSurroundVotes(ctx, idx, req)
			if err != nil {
				errorChans <- err
				wg.Done()
				return
			}
			if atts != nil && len(atts) > 0 {
				attSlashings <- atts
			}
			wg.Done()
			return
		}(idx, root, req)
	}
	wg.Wait()
	close(errorChans)
	close(attSlashings)
	for e := range errorChans {
		if err != nil {
			err = fmt.Errorf(err.Error() + " : " + e.Error())
			continue
		}
		err = e
	}
	for atts := range attSlashings {
		attSlashingResp.AttesterSlashing = append(attSlashingResp.AttesterSlashing, atts...)
	}
	return attSlashingResp, err
}

// UpdateSpanMaps updates and load all span maps from db.
func (ss *Server) UpdateSpanMaps(ctx context.Context, req *ethpb.IndexedAttestation) error {
	indices := req.AttestingIndices
	lastIdx := int64(-1)
	var wg sync.WaitGroup
	er := make(chan error, len(indices))
	for _, idx := range indices {
		if int64(idx) <= lastIdx {
			er <- fmt.Errorf("indexed attestation contains repeated or non sorted ids")
		}
		wg.Add(1)
		go func(i uint64) {
			spanMap, err := ss.SlasherDB.ValidatorSpansMap(ctx, i)
			if err != nil {
				er <- err
				wg.Done()
				return
			}
			if req.Data == nil {
				log.Trace("Got indexed attestation with no data")
				wg.Done()
				return
			}
			spanMap, _, err = attestations.DetectAndUpdateSpans(ctx, spanMap, req)
			if err != nil {
				er <- err
				wg.Done()
				return
			}
			if err := ss.SlasherDB.SaveValidatorSpansMap(ctx, i, spanMap); err != nil {
				er <- err
				wg.Done()
				return
			}
		}(idx)
		wg.Wait()
	}
	close(er)
	for e := range er {
		log.Errorf("Got error while trying to update span maps: %v", e)
	}
	return nil
}

// IsSlashableBlock returns a proposer slashing if the block header submitted is
// a slashable proposal.
func (ss *Server) IsSlashableBlock(ctx context.Context, psr *slashpb.ProposerSlashingRequest) (*slashpb.ProposerSlashingResponse, error) {
	//TODO(#3133): add signature validation
	epoch := helpers.SlotToEpoch(psr.BlockHeader.Header.Slot)
	blockHeaders, err := ss.SlasherDB.BlockHeaders(ctx, epoch, psr.ValidatorIndex)
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
		err = ss.SlasherDB.SaveBlockHeader(ctx, epoch, psr.ValidatorIndex, psr.BlockHeader)
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
	pSlashingsResponse.ProposerSlashing, err = ss.SlasherDB.ProposalSlashingsByStatus(ctx, types.SlashingStatus(st.Status))
	if err != nil {
		return nil, err
	}
	return pSlashingsResponse, nil
}

// AttesterSlashings returns attester slashings if slashing with the requested status are found in the db.
func (ss *Server) AttesterSlashings(ctx context.Context, st *slashpb.SlashingStatusRequest) (*slashpb.AttesterSlashingResponse, error) {
	aSlashingsResponse := &slashpb.AttesterSlashingResponse{}
	var err error
	aSlashingsResponse.AttesterSlashing, err = ss.SlasherDB.AttesterSlashings(ctx, types.SlashingStatus(st.Status))
	if err != nil {
		return nil, err
	}
	return aSlashingsResponse, nil
}

// DetectSurroundVotes is a method used to return the attestation that were detected
// by min max surround detection method.
func (ss *Server) DetectSurroundVotes(ctx context.Context, validatorIdx uint64, req *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error) {
	spanMap, err := ss.SlasherDB.ValidatorSpansMap(ctx, validatorIdx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get validator spans map")
	}
	spanMap, slashableEpoch, err := attestations.DetectAndUpdateSpans(ctx, spanMap, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update spans")
	}
	if err := ss.SlasherDB.SaveValidatorSpansMap(ctx, validatorIdx, spanMap); err != nil {
		return nil, errors.Wrap(err, "failed to save validator spans map")
	}

	var as []*ethpb.AttesterSlashing
	if slashableEpoch > 0 {
		atts, err := ss.SlasherDB.IdxAttsForTargetFromID(ctx, slashableEpoch, validatorIdx)
		if err != nil {
			return nil, err
		}
		for _, ia := range atts {
			if ia.Data == nil {
				continue
			}
			surrounding := ia.Data.Source.Epoch < req.Data.Source.Epoch && ia.Data.Target.Epoch > req.Data.Target.Epoch
			surrounded := ia.Data.Source.Epoch > req.Data.Source.Epoch && ia.Data.Target.Epoch < req.Data.Target.Epoch
			if surrounding || surrounded {
				as = append(as, &ethpb.AttesterSlashing{
					Attestation_1: req,
					Attestation_2: ia,
				})
			}
		}
	}
	return as, nil
}
