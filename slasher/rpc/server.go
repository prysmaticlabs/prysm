package rpc

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/p2putils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type Server struct {
	ctx          context.Context
	detector     *detection.Service
	slasherDB    db.Database
	beaconClient *beaconclient.Service
}

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (ss *Server) IsSlashableAttestation(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.AttesterSlashingResponse, error) {
	ctx, span := trace.StartSpan(ctx, "detection.IsSlashableAttestation")
	defer span.End()
	valid, err := ss.verifyIndexedAttestationSig(ctx, req)
	if err != nil {
		log.WithError(err).Error("Failed to verify indexed attestation signature")
		return nil, status.Errorf(codes.Internal, "Could not verify indexed attestation signature: %v: %v", req, err)
	}
	if !valid {
		log.Warning("Indexed attestation signature failed to verify")
		return nil, status.Errorf(codes.Unauthenticated, "Indexed attestation signature failed to verify: %v", req)
	}

	if err := ss.slasherDB.SaveIndexedAttestation(ctx, req); err != nil {
		log.WithError(err).Error("Could not save indexed attestation")
		return nil, status.Errorf(codes.Internal, "Could not save indexed attestation: %v: %v", req, err)
	}
	slashings, err := ss.detector.DetectAttesterSlashings(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not detect attester slashings for attestation: %v: %v", req, err)
	}
	if len(slashings) < 1 {
		if err := ss.detector.UpdateSpans(ctx, req); err != nil {
			log.WithError(err).Error("Could not update spans")
		}
	}
	return &slashpb.AttesterSlashingResponse{
		AttesterSlashing: slashings,
	}, nil
}

// IsSlashableBlock returns an proposer slashing if the block submitted
// is a double proposal.
func (ss *Server) IsSlashableBlock(ctx context.Context, req *ethpb.SignedBeaconBlockHeader) (*slashpb.ProposerSlashingResponse, error) {
	return nil, errors.New("unimplemented")
}

func (ss *Server) verifyIndexedAttestationSig(ctx context.Context, indexedAtt *ethpb.IndexedAttestation) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "core.verifyIndexedAttestationSig")
	defer span.End()
	if indexedAtt == nil || indexedAtt.Data == nil || indexedAtt.Data.Target == nil {
		return false, errors.New("nil or missing indexed attestation data")
	}
	indices := indexedAtt.AttestingIndices

	if uint64(len(indices)) > params.BeaconConfig().MaxValidatorsPerCommittee {
		return false, fmt.Errorf("validator indices count exceeds MAX_VALIDATORS_PER_COMMITTEE, %d > %d", len(indices), params.BeaconConfig().MaxValidatorsPerCommittee)
	}

	set := make(map[uint64]bool)
	setIndices := make([]uint64, 0, len(indices))
	for _, i := range indices {
		if ok := set[i]; ok {
			continue
		}
		setIndices = append(setIndices, i)
		set[i] = true
	}
	sort.SliceStable(setIndices, func(i, j int) bool {
		return setIndices[i] < setIndices[j]
	})
	if !reflect.DeepEqual(setIndices, indices) {
		return false, errors.New("attesting indices is not uniquely sorted")
	}
	pkMap, err := ss.beaconClient.FindOrGetPublicKeys(ctx, indices)
	if err != nil {
		return false, err
	}
	gvr, err := ss.beaconClient.GenesisValidatorsRoot(ctx)
	if err != nil {
		return false, err
	}
	fork, err := p2putils.Fork(indexedAtt.Data.Target.Epoch)
	if err != nil {
		return false, err
	}
	domain, err := helpers.Domain(fork, indexedAtt.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester, gvr)
	if err != nil {
		return false, err
	}
	pubkeys := []*bls.PublicKey{}

	for _, pkBytes := range pkMap {
		pk, err := bls.PublicKeyFromBytes(pkBytes[:])
		if err != nil {
			return false, errors.Wrap(err, "could not deserialize validator public key")
		}
		pubkeys = append(pubkeys, pk)
	}

	messageHash, err := helpers.ComputeSigningRoot(indexedAtt.Data, domain)
	if err != nil {
		return false, errors.Wrap(err, "could not get signing root of object")
	}

	sig, err := bls.SignatureFromBytes(indexedAtt.Signature)
	if err != nil {
		return false, errors.Wrap(err, "could not convert bytes to signature")
	}

	voted := len(indices) > 0
	if voted && !sig.FastAggregateVerify(pubkeys, messageHash) {
		return false, helpers.ErrSigFailedToVerify
	}
	return true, nil
}
