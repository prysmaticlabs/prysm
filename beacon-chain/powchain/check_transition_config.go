package powchain

import (
	"context"
	"errors"
	"math/big"
	"time"

	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	"github.com/prysmaticlabs/prysm/config/params"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
)

const (
	checkTransitionPollingInterval = time.Minute
)

var configMismatchLog = "Configuration mismatch between your execution client and Prysm. Node is shutting down as it" +
	" will not be able to complete the proof-of-stake transition"

// Checks the transition configuration between Prysm and the connected execution node to ensure
// there are no differences. If there are any discrepancies, Prysm must halt its beacon node as
// it will not be ready for the merge transition.
func (s *Service) checkTransitionConfiguration(ctx context.Context) {
	if s.engineAPIClient == nil {
		return
	}
	cfg := &pb.TransitionConfiguration{
		TerminalTotalDifficulty: params.BeaconConfig().TerminalTotalDifficulty,
		TerminalBlockHash:       params.BeaconConfig().TerminalBlockHash[:],
		TerminalBlockNumber:     big.NewInt(0).Bytes(), // A value of 0 is required in the request.
	}
	err := s.engineAPIClient.ExchangeTransitionConfiguration(ctx, cfg)
	handleExchangeConfigurationError(err)

	// We poll the execution client to see if the transition configuration has changed.
	// This serves as a heartbeat to ensure the execution client and Prysm are ready for the
	// Bellatrix hard-fork transition.
	ticker := time.NewTicker(checkTransitionPollingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err = s.engineAPIClient.ExchangeTransitionConfiguration(ctx, cfg)
			handleExchangeConfigurationError(err)
		}
	}
}

// We check if there is a configuration mismatch error between the execution client
// and the Prysm beacon node. If so, we need to halt the node as it cannot successfully
// complete the merge transition for the Bellatrix hard fork.
func handleExchangeConfigurationError(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, v1.ErrConfigMismatch) {
		log.WithError(err).Fatal(configMismatchLog)
	}
	log.WithError(err).Error("Could not check configuration values between execution and consensus client")
	return
}
