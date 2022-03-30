package powchain

import (
	"context"
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/powchain/engine-api-client/v1"
	"github.com/prysmaticlabs/prysm/config/params"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/sirupsen/logrus"
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
func (s *Service) checkTransitionConfiguration(
	ctx context.Context, blockNotifications chan *statefeed.BlockProcessedData,
) {
	// If Bellatrix fork epoch is not set, we do not run this check.
	if params.BeaconConfig().BellatrixForkEpoch == math.MaxUint64 {
		return
	}
	// If no engine API, then also avoid running this check.
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
	hasTtdReached := false
	defer ticker.Stop()
	sub := s.cfg.stateNotifier.StateFeed().Subscribe(blockNotifications)
	defer sub.Unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case <-sub.Err():
			return
		case ev := <-blockNotifications:
			isExecutionBlock, err := blocks.IsExecutionBlock(ev.SignedBlock.Block().Body())
			if err != nil {
				log.WithError(err).Debug("Could not check whether signed block is execution block")
				continue
			}
			if isExecutionBlock {
				log.Debug("PoS transition is complete, no longer checking for configuration changes")
				return
			}
		case <-ticker.C:
			err = s.engineAPIClient.ExchangeTransitionConfiguration(ctx, cfg)
			s.handleExchangeConfigurationError(err)
			if !hasTtdReached {
				hasTtdReached, err = s.logTtdStatus(ctx, ttd)
				if err != nil {
					log.WithError(err).Error("Could not log ttd status")
				}
			}
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

// Logs the terminal total difficulty status.
func (s *Service) logTtdStatus(ctx context.Context, ttd *uint256.Int) (bool, error) {
	latest, err := s.engineAPIClient.LatestExecutionBlock(ctx)
	if err != nil {
		return false, err
	}
	latestTtd, err := hexutil.DecodeBig(latest.TotalDifficulty)
	if err != nil {
		return false, err
	}
	if latestTtd.Cmp(ttd.ToBig()) >= 0 {
		return true, nil
	}
	log.WithFields(logrus.Fields{
		"latestTTD":     latestTtd.String(),
		"transitionTTD": ttd.ToBig().String(),
	}).Info("Total terminal difficulty has not been reached yet")
	return false, nil
}
