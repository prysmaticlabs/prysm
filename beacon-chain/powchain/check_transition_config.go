package powchain

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/holiman/uint256"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	"github.com/prysmaticlabs/prysm/config/params"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
)

var (
	checkTransitionPollingInterval = time.Second * 10
	configMismatchLog              = "Configuration mismatch between your execution client and Prysm. " +
		"Please check your execution client and restart it with the proper configuration. If this is not done, " +
		"your node will not be able to complete the proof-of-stake transition"
)

// Checks the transition configuration between Prysm and the connected execution node to ensure
// there are no differences in terminal block difficulty and block hash.
// If there are any discrepancies, we must log errors to ensure users can resolve
//the problem and be ready for the merge transition.
func (s *Service) checkTransitionConfiguration(ctx context.Context) {
	if s.engineAPIClient == nil {
		return
	}
	i := new(big.Int)
	i.SetString(params.BeaconConfig().TerminalTotalDifficulty, 10)
	ttd := new(uint256.Int)
	ttd.SetFromBig(i)
	cfg := &pb.TransitionConfiguration{
		TerminalTotalDifficulty: ttd.Hex(),
		TerminalBlockHash:       params.BeaconConfig().TerminalBlockHash[:],
		TerminalBlockNumber:     big.NewInt(0).Bytes(), // A value of 0 is recommended in the request.
	}
	err := s.engineAPIClient.ExchangeTransitionConfiguration(ctx, cfg)
	if err != nil {
		if errors.Is(err, v1.ErrConfigMismatch) {
			log.WithError(err).Fatal(configMismatchLog)
		}
		log.WithError(err).Error("Could not check configuration values between execution and consensus client")
	}

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
			s.handleExchangeConfigurationError(err)
		}
	}
}

// We check if there is a configuration mismatch error between the execution client
// and the Prysm beacon node. If so, we need to log errors in the node as it cannot successfully
// complete the merge transition for the Bellatrix hard fork.
func (s *Service) handleExchangeConfigurationError(err error) {
	if err == nil {
		// If there is no error in checking the exchange configuration error, we clear
		// the run error of the service if we had previously set it to ErrConfigMismatch.
		if errors.Is(s.runError, v1.ErrConfigMismatch) {
			s.runError = nil
		}
		return
	}
	// If the error is a configuration mismatch, we set a runtime error in the service.
	if errors.Is(err, v1.ErrConfigMismatch) {
		s.runError = err
		log.WithError(err).Error(configMismatchLog)
		return
	}
	log.WithError(err).Error("Could not check configuration values between execution and consensus client")
}
