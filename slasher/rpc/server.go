package rpc

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection"
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

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (ss *Server) IsSlashableAttestation(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.AttesterSlashingResponse, error) {
	ctx, span := trace.StartSpan(ctx, "detection.IsSlashableAttestation")
	defer span.End()
	ss.attestationLock.Lock()
	defer ss.attestationLock.Unlock()

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
	gvr, err := ss.beaconClient.GenesisValidatorsRoot(ctx)
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
	indices := req.AttestingIndices
	pkMap, err := ss.beaconClient.FindOrGetPublicKeys(ctx, indices)
	if err != nil {
		return nil, err
	}
	pubkeys := []bls.PublicKey{}
	for _, pkBytes := range pkMap {
		pk, err := bls.PublicKeyFromBytes(pkBytes[:])
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
	slashings, err := ss.detector.DetectAttesterSlashings(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not detect attester slashings for attestation: %v: %v", req, err)
	}
	if len(slashings) < 1 {
		if err := ss.slasherDB.SaveIndexedAttestation(ctx, req); err != nil {
			log.WithError(err).Error("Could not save indexed attestation")
			return nil, status.Errorf(codes.Internal, "could not save indexed attestation: %v: %v", req, err)
		}
		if err := ss.detector.UpdateSpans(ctx, req); err != nil {
			log.WithError(err).Error("could not update spans")
		}
	}
	return &slashpb.AttesterSlashingResponse{
		AttesterSlashing: slashings,
	}, nil
}

// IsSlashableBlock returns an proposer slashing if the block submitted
// is a double proposal.
func (ss *Server) IsSlashableBlock(ctx context.Context, req *ethpb.SignedBeaconBlockHeader) (*slashpb.ProposerSlashingResponse, error) {
	ctx, span := trace.StartSpan(ctx, "detection.IsSlashableBlock")
	defer span.End()
	ss.proposeLock.Lock()
	defer ss.proposeLock.Unlock()

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request provided")
	}
	if req.Header == nil {
		return nil, status.Error(codes.InvalidArgument, "nil header provided")

	}
	if req.Signature == nil {
		return nil, status.Error(codes.InvalidArgument, "nil signature provided")
	}
	gvr, err := ss.beaconClient.GenesisValidatorsRoot(ctx)
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
	pkMap, err := ss.beaconClient.FindOrGetPublicKeys(ctx, []uint64{req.Header.ProposerIndex})
	if err := helpers.VerifyBlockHeaderSigningRoot(
		req.Header, pkMap[req.Header.ProposerIndex], req.Signature, domain); err != nil {
		return nil, err
	}
	slashing, err := ss.detector.DetectDoubleProposals(ctx, req)
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
func (ss *Server) IsSlashableAttestationNoUpdate(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.Slashable, error) {
	sl := &slashpb.Slashable{}
	slashings, err := ss.detector.DetectAttesterSlashings(ctx, req)
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
func (ss *Server) IsSlashableBlockNoUpdate(ctx context.Context, req *ethpb.BeaconBlockHeader) (*slashpb.Slashable, error) {
	sl := &slashpb.Slashable{}
	slash, err := ss.detector.DetectDoubleProposeNoUpdate(ctx, req)
	if err != nil {
		return sl, status.Errorf(codes.Internal, "could not detect proposer slashing for block: %v: %v", req, err)
	}
	sl.Slashable = slash
	return sl, nil
}
