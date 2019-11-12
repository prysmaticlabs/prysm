package attester

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log logrus.FieldLogger

func init() {
	log = logrus.WithField("prefix", "rpc/attester")
}

// Server defines a server implementation of the gRPC Attester service,
// providing RPC methods for validators acting as attesters to broadcast votes on beacon blocks.
type Server struct {
	P2p               p2p.Broadcaster
	BeaconDB          db.Database
	OperationsHandler operations.Handler
	AttReceiver       blockchain.AttestationReceiver
	HeadFetcher       blockchain.HeadFetcher
	AttestationCache  *cache.AttestationCache
	SyncChecker       sync.Checker
}

// SubmitAttestation is a function called by an attester in a sharding validator to vote
// on a block via an attestation object as defined in the Ethereum Serenity specification.
func (as *Server) SubmitAttestation(ctx context.Context, att *ethpb.Attestation) (*pb.AttestResponse, error) {
	root, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to hash tree root attestation")
	}

	go func() {
		ctx = trace.NewContext(context.Background(), trace.FromContext(ctx))
		attCopy := proto.Clone(att).(*ethpb.Attestation)
		if err := as.AttReceiver.ReceiveAttestation(ctx, att); err != nil {
			log.WithError(err).Error("could not receive attestation in chain service")
			return
		}
		if err := as.OperationsHandler.HandleAttestation(ctx, attCopy); err != nil {
			log.WithError(err).Error("could not handle attestation in operations service")
			return
		}
	}()

	return &pb.AttestResponse{Root: root[:]}, nil
}

// RequestAttestation requests that the beacon node produce an IndexedAttestation,
// with a blank signature field, which the validator will then sign.
func (as *Server) RequestAttestation(ctx context.Context, req *pb.AttestationRequest) (*ethpb.AttestationData, error) {
	ctx, span := trace.StartSpan(ctx, "AttesterServer.RequestAttestation")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("slot", int64(req.Slot)),
		trace.Int64Attribute("committeeIndex", int64(req.CommitteeIndex)),
	)

	if as.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	res, err := as.AttestationCache.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	if res != nil {
		return res, nil
	}

	if err := as.AttestationCache.MarkInProgress(req); err != nil {
		if err == cache.ErrAlreadyInProgress {
			res, err := as.AttestationCache.Get(ctx, req)
			if err != nil {
				return nil, err
			}

			if res == nil {
				return nil, errors.New("a request was in progress and resolved to nil")
			}
			return res, nil
		}
		return nil, err
	}
	defer func() {
		if err := as.AttestationCache.MarkNotInProgress(req); err != nil {
			log.WithError(err).Error("Failed to mark cache not in progress")
		}
	}()

	headState := as.HeadFetcher.HeadState()
	headRoot := as.HeadFetcher.HeadRoot()

	// Safe guard against head state is nil in chain service. This should not happen.
	if headState == nil {
		headState, err = as.BeaconDB.HeadState(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Safe guard against head state is nil in chain service. This should not happen.
	if headState == nil {
		headState, err = as.BeaconDB.HeadState(ctx)
		if err != nil {
			return nil, err
		}
	}

	headState, err = state.ProcessSlots(ctx, headState, req.Slot)
	if err != nil {
		return nil, errors.Wrapf(err, "could not process slots up to %d", req.Slot)
	}

	targetEpoch := helpers.CurrentEpoch(headState)
	epochStartSlot := helpers.StartSlot(targetEpoch)
	targetRoot := make([]byte, 32)
	if epochStartSlot == headState.Slot {
		targetRoot = headRoot[:]
	} else {
		targetRoot, err = helpers.BlockRootAtSlot(headState, epochStartSlot)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get target block for slot %d", epochStartSlot)
		}
	}

	res = &ethpb.AttestationData{
		Slot:            req.Slot,
		Index:           req.CommitteeIndex,
		BeaconBlockRoot: headRoot[:],
		Source:          headState.CurrentJustifiedCheckpoint,
		Target: &ethpb.Checkpoint{
			Epoch: targetEpoch,
			Root:  targetRoot,
		},
	}

	if err := as.AttestationCache.Put(ctx, req, res); err != nil {
		return nil, err
	}

	return res, nil
}
