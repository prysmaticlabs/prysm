package rpc

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type Server struct {
	ctx             context.Context
	detector        *detection.Service
	slasherDB       db.Database
	beaconClient    *beaconclient.Service
	attestationLock sync.Mutex
	proposeLock     sync.Mutex
}

// HighestAttestations returns the highest observed attestation source and epoch for a given validator id.
func (s *Server) HighestAttestations(ctx context.Context, req *slashpb.HighestAttestationRequest) (*slashpb.HighestAttestationResponse, error) {
	ctx, span := trace.StartSpan(ctx, "history.HighestAttestations")
	defer span.End()

	ret := make([]*slashpb.HighestAttestation, 0)
	for _, id := range req.ValidatorIds {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		res, err := s.slasherDB.HighestAttestation(ctx, id)
		if err != nil {
			return nil, err
		}
		if res != nil {
			ret = append(ret, &slashpb.HighestAttestation{
				ValidatorId:        res.ValidatorId,
				HighestTargetEpoch: res.HighestTargetEpoch,
				HighestSourceEpoch: res.HighestSourceEpoch,
			})
		}
	}

	return &slashpb.HighestAttestationResponse{
		Attestations: ret,
	}, nil
}

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (s *Server) IsSlashableAttestation(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.AttesterSlashingResponse, error) {
	ctx, span := trace.StartSpan(ctx, "detection.IsSlashableAttestation")
	defer span.End()

	log.WithFields(logrus.Fields{
		"slot":    req.Data.Slot,
		"indices": req.AttestingIndices,
	}).Debug("Received attestation via RPC")
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request provided")
	}
	if req.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request data provided")
	}
	if req.Data.Target == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request data target provided")
	}
	if req.Data.Source == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request data source provided")
	}
	if req.Signature == nil {
		return nil, status.Error(codes.InvalidArgument, "nil signature provided")
	}

	err := attestationutil.IsValidAttestationIndices(ctx, req)
	if err != nil {
		return nil, err
	}
	gvr, err := s.beaconClient.GenesisValidatorsRoot(ctx)
	if err != nil {
		return nil, err
	}
	fork, err := p2putils.Fork(req.Data.Target.Epoch)
	if err != nil {
		return nil, err
	}
	domain, err := helpers.Domain(fork, req.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, gvr)
	if err != nil {
		return nil, err
	}
	indices := make([]types.ValidatorIndex, len(req.AttestingIndices))
	for i, index := range req.AttestingIndices {
		indices[i] = types.ValidatorIndex(index)
	}
	pkMap, err := s.beaconClient.FindOrGetPublicKeys(ctx, indices)
	if err != nil {
		return nil, err
	}
	var pubkeys []bls.PublicKey
	for _, pkBytes := range pkMap {
		pk, err := bls.PublicKeyFromBytes(pkBytes)
		if err != nil {
			return nil, errors.Wrap(err, "could not deserialize validator public key")
		}
		pubkeys = append(pubkeys, pk)
	}

	err = attestationutil.VerifyIndexedAttestationSig(ctx, req, pubkeys, domain)
	if err != nil {
		log.WithError(err).Error("failed to verify indexed attestation signature")
		return nil, status.Errorf(codes.Internal, "could not verify indexed attestation signature: %v: %v", req, err)
	}

	s.attestationLock.Lock()
	defer s.attestationLock.Unlock()

	slashings, err := s.detector.DetectAttesterSlashings(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not detect attester slashings for attestation: %v: %v", req, err)
	}
	if len(slashings) < 1 {
		if err := s.slasherDB.SaveIndexedAttestation(ctx, req); err != nil {
			log.WithError(err).Error("Could not save indexed attestation")
			return nil, status.Errorf(codes.Internal, "could not save indexed attestation: %v: %v", req, err)
		}
		if err := s.detector.UpdateSpans(ctx, req); err != nil {
			log.WithError(err).Error("could not update spans")
			return nil, status.Errorf(codes.Internal, "failed to update spans: %v: %v", req, err)
		}
	}
	return &slashpb.AttesterSlashingResponse{
		AttesterSlashing: slashings,
	}, nil
}

// IsSlashableBlock returns an proposer slashing if the block submitted
// is a double proposal.
func (s *Server) IsSlashableBlock(ctx context.Context, req *ethpb.SignedBeaconBlockHeader) (*slashpb.ProposerSlashingResponse, error) {
	ctx, span := trace.StartSpan(ctx, "detection.IsSlashableBlock")
	defer span.End()

	log.WithFields(logrus.Fields{
		"slot":           req.Header.Slot,
		"proposer_index": req.Header.ProposerIndex,
	}).Info("Received block via RPC")
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request provided")
	}
	if req.Header == nil {
		return nil, status.Error(codes.InvalidArgument, "nil header provided")

	}
	if req.Signature == nil {
		return nil, status.Error(codes.InvalidArgument, "nil signature provided")
	}
	gvr, err := s.beaconClient.GenesisValidatorsRoot(ctx)
	if err != nil {
		return nil, err
	}
	blockEpoch := helpers.SlotToEpoch(req.Header.Slot)
	fork, err := p2putils.Fork(blockEpoch)
	if err != nil {
		return nil, err
	}
	domain, err := helpers.Domain(fork, blockEpoch, params.BeaconConfig().DomainBeaconProposer, gvr)
	if err != nil {
		return nil, err
	}
	pkMap, err := s.beaconClient.FindOrGetPublicKeys(ctx, []types.ValidatorIndex{req.Header.ProposerIndex})
	if err != nil {
		return nil, err
	}
	if err := helpers.VerifyBlockHeaderSigningRoot(
		req.Header,
		pkMap[req.Header.ProposerIndex],
		req.Signature, domain); err != nil {
		return nil, err
	}

	s.proposeLock.Lock()
	defer s.proposeLock.Unlock()

	slashing, err := s.detector.DetectDoubleProposals(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not detect proposer slashing for block: %v: %v", req, err)
	}
	psr := &slashpb.ProposerSlashingResponse{}
	if slashing != nil {
		psr = &slashpb.ProposerSlashingResponse{
			ProposerSlashing: []*ethpb.ProposerSlashing{slashing},
		}
	}
	return psr, nil

}

// IsSlashableAttestationNoUpdate returns true if the attestation submitted
// is a slashable vote (no db update is being done).
func (s *Server) IsSlashableAttestationNoUpdate(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.Slashable, error) {
	sl := &slashpb.Slashable{}
	slashings, err := s.detector.DetectAttesterSlashings(ctx, req)
	if err != nil {
		return sl, status.Errorf(codes.Internal, "could not detect attester slashings for attestation: %v: %v", req, err)
	}
	if len(slashings) < 1 {
		return sl, nil
	}
	sl.Slashable = true
	return sl, nil
}

// IsSlashableBlockNoUpdate returns true if the block submitted
// is slashable (no db update is being done).
func (s *Server) IsSlashableBlockNoUpdate(ctx context.Context, req *ethpb.BeaconBlockHeader) (*slashpb.Slashable, error) {
	sl := &slashpb.Slashable{}
	slash, err := s.detector.DetectDoubleProposeNoUpdate(ctx, req)
	if err != nil {
		return sl, status.Errorf(codes.Internal, "could not detect proposer slashing for block: %v: %v", req, err)
	}
	sl.Slashable = slash
	return sl, nil
}
