package slasher

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"

	st "github.com/prysmaticlabs/prysm/beacon-chain/state"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/sirupsen/logrus"
)

var log logrus.FieldLogger

func init() {
	log = logrus.WithField("prefix", "rpc/aggregator")
}

// Client defines a client implementation of the gRPC slasher service.
type Client struct {
	HeadFetcher   blockchain.HeadFetcher
	SlashingPool  *slashings.Pool
	SlasherClient slashpb.SlasherClient
	//P2p Will later be used to send slashing on pub sub
	P2p             p2p.Broadcaster
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
				return err
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
		s.updateSlashingPool(ctx, state)
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
		s.SlashingPool.InsertProposerSlashing(ctx, state, ps)
	}
	for _, as := range asr.AttesterSlashing {
		s.SlashingPool.InsertAttesterSlashing(ctx, state, as)
	}
	return nil
}
