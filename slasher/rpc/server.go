package rpc

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	beaconClient ethpb.BeaconChainClient
}

// IsSlashableAttestation returns an attester slashing if the attestation submitted
// is a slashable vote.
func (ss *Server) IsSlashableAttestation(ctx context.Context, req *ethpb.IndexedAttestation) (*slashpb.AttesterSlashingResponse, error) {
	ctx, span := trace.StartSpan(ctx, "detection.IsSlashableAttestation")
	defer span.End()
	pubMap, err := ss.beaconClient.FindOrGetPublicKeys(ctx, req.AttestingIndices)
	if err != nil {
		log.WithError(err).Error("Failed to retrieve public key for validators")
		return nil, status.Errorf(codes.Internal, "Could not retrieve public keys for validators: %v: %v", req.AttestingIndices, err)
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

func (ss *Server) verifySig(ctx context.Context, indexedAtt *ethpb.IndexedAttestation) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "core.VerifyIndexedAttestation")
	defer span.End()
	if indexedAtt == nil || indexedAtt.Data == nil || indexedAtt.Data.Target == nil {
		return errors.New("nil or missing indexed attestation data")
	}
	indices := indexedAtt.AttestingIndices

	if uint64(len(indices)) > params.BeaconConfig().MaxValidatorsPerCommittee {
		return fmt.Errorf("validator indices count exceeds MAX_VALIDATORS_PER_COMMITTEE, %d > %d", len(indices), params.BeaconConfig().MaxValidatorsPerCommittee)
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
		return errors.New("attesting indices is not uniquely sorted")
	}

	domain, err := helpers.Domain(beaconState.Fork(), indexedAtt.Data.Target.Epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return false, err
	}
	var pubkey *bls.PublicKey
	if len(indices) > 0 {
		pubkeyAtIdx := beaconState.PubkeyAtIndex(indices[0])
		pubkey, err = bls.PublicKeyFromBytes(pubkeyAtIdx[:])
		if err != nil {
			return false, errors.Wrap(err, "could not deserialize validator public key")
		}
		for i := 1; i < len(indices); i++ {
			pubkeyAtIdx = beaconState.PubkeyAtIndex(indices[i])
			pk, err := bls.PublicKeyFromBytes(pubkeyAtIdx[:])
			if err != nil {
				return false, errors.Wrap(err, "could not deserialize validator public key")
			}
			pubkey.Aggregate(pk)
		}
	}

	messageHash, err := ssz.HashTreeRoot(indexedAtt.Data)
	if err != nil {
		return false, errors.Wrap(err, "could not tree hash att data")
	}

	sig, err := bls.SignatureFromBytes(indexedAtt.Signature)
	if err != nil {
		return false, errors.Wrap(err, "could not convert bytes to signature")
	}

	voted := len(indices) > 0
	if voted && !sig.Verify(messageHash[:], pubkey, domain) {
		return false, ErrSigFailedToVerify
	}
}
