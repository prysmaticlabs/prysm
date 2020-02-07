package slasher

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	st "github.com/prysmaticlabs/prysm/beacon-chain/state"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log logrus.FieldLogger

func init() {
	log = logrus.WithField("prefix", "rpc/aggregator")
}

// Client defines a client implementation of the gRPC slasher service.
type Client struct {
	HeadFetcher     blockchain.HeadFetcher
	SlashingPool    *slashings.Pool
	SlasherClient   slashpb.SlasherClient
	P2p             p2p.Broadcaster //P2p Will later be used to send slashing on pub sub.
	ShouldBroadcast bool
}

// SlashingPoolFeeder this is a stub for the coming PRs #3133
// Store validator index to public key map Validate attestation signature.
func (s *Client) SlashingPoolFeeder(ctx context.Context) error {
	secondsPerEpoch := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch
	d := time.Duration(secondsPerEpoch) * time.Second
	tick := time.Tick(d)
	for {
		select {
		case <-tick:
			if err := s.updatePool(ctx); err != nil {
				return errors.Wrap(err, "failed to update slashing pool")
			}
		case <-ctx.Done():
			err := status.Error(codes.Canceled, "Stream context canceled")
			log.WithError(err)
			return err
		}
	}
}

func (s *Client) updatePool(ctx context.Context) error {
	state, err := s.HeadFetcher.HeadState(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	if s.SlasherClient != nil {
		if err := s.updateSlashingPool(ctx, state); err != nil {
			log.WithError(err)
			return err
		}
	} else {
		err := status.Error(codes.Internal, "Slasher server has not been started")
		log.WithError(err)
		return err
	}
	return nil
}

func (s *Client) updateSlashingPool(ctx context.Context, state *st.BeaconState) error {
	psr, err := s.SlasherClient.ProposerSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active})
	if err != nil {
		return status.Errorf(codes.Internal, "Could not retrieve proposer slashings: %v", err)
	}
	asr, err := s.SlasherClient.AttesterSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active})
	if err != nil {
		return status.Errorf(codes.Internal, "Could not retrieve attester slashings: %v", err)
	}
	for _, ps := range psr.ProposerSlashing {
		if err := s.SlashingPool.InsertProposerSlashing(state, ps); err != nil {
			log.WithError(errors.Wrap(err, "could not insert proposer slashing into pool"))
			continue
		}
	}
	for _, as := range asr.AttesterSlashing {
		if err := s.SlashingPool.InsertAttesterSlashing(state, as); err != nil {
			log.WithError(errors.Wrap(err, "could not insert attester slashing into pool"))
			continue
		}
	}
	log.Infof("updating slashing pool with %d attestations", len(asr.AttesterSlashing))

	return nil
}
