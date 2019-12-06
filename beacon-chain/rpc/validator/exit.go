package validator

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/statefeed"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RequestExit requests an exit for a validator.
func (vs *Server) RequestExit(ctx context.Context, req *ethpb.VoluntaryExit) (*ptypes.Empty, error) {
	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Confirm the validator is eligible to exit with the parameters provided
	if req.ValidatorIndex >= uint64(len(s.Validators)) {
		return nil, status.Errorf(codes.InvalidArgument, "Unknown validator index %d", req.ValidatorIndex)
	}
	validator := s.Validators[req.ValidatorIndex]
	if !helpers.IsActiveValidator(validator, req.Epoch) {
		return nil, status.Errorf(codes.InvalidArgument, "Validator %d not active at epoch %d", req.ValidatorIndex, req.Epoch)
	}
	if validator.ExitEpoch != params.BeaconConfig().FarFutureEpoch {
		return nil, status.Errorf(codes.InvalidArgument, "Validator %d already exiting", req.ValidatorIndex)
	}

	secondsPerEpoch := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch
	genesisTime := vs.GenesisTimeFetcher.GenesisTime().Unix()
	currentEpoch := uint64(roughtime.Now().Unix()-genesisTime) / secondsPerEpoch
	earliestRequestedExitEpoch := mathutil.Max(req.Epoch, currentEpoch)
	earliestExitEpoch := validator.ActivationEpoch + params.BeaconConfig().PersistentCommitteePeriod
	if earliestRequestedExitEpoch < earliestExitEpoch {
		return nil, status.Errorf(codes.InvalidArgument, "Validator %d cannot exit before epoch %d", req.ValidatorIndex, earliestExitEpoch)
	}

	// Confirm signature is valid
	root, err := ssz.SigningRoot(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Malformed request")
	}
	sig, err := bls.SignatureFromBytes(req.Signature)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Malformed signature: %v", err)
	}
	validatorPubKey, err := bls.PublicKeyFromBytes(validator.PublicKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Invalid validator public key: %v", err)
	}
	domain := bls.ComputeDomain(params.BeaconConfig().DomainVoluntaryExit)
	verified := sig.Verify(root[:], validatorPubKey, domain)
	if !verified {
		return nil, status.Error(codes.InvalidArgument, "Incorrect signature")
	}

	// Send the voluntary exit to the state feed.
	vs.StateNotifier.StateFeed().Send(&statefeed.Event{
		Type: statefeed.VoluntaryExitReceived,
		Data: &statefeed.VoluntaryExitReceivedData{
			VoluntaryExit: req,
		},
	})

	return nil, nil
}
