package execution

import (
	"context"
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/network"
	pb "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
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
	ctx context.Context, blockNotifications chan *feed.Event,
) {
	// If Bellatrix fork epoch is not set, we do not run this check.
	if params.BeaconConfig().BellatrixForkEpoch == math.MaxUint64 {
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
	err := s.ExchangeTransitionConfiguration(ctx, cfg)
	if err != nil {
		if errors.Is(err, ErrConfigMismatch) {
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
			data, ok := ev.Data.(*statefeed.BlockProcessedData)
			if !ok {
				continue
			}
			isExecutionBlock, err := blocks.IsExecutionBlock(data.SignedBlock.Block().Body())
			if err != nil {
				log.WithError(err).Debug("Could not check whether signed block is execution block")
				continue
			}
			if isExecutionBlock {
				log.Debug("PoS transition is complete, no longer checking for configuration changes")
				return
			}
		case tm := <-ticker.C:
			ctx, cancel := context.WithDeadline(ctx, tm.Add(network.DefaultRPCHTTPTimeout))
			err = s.ExchangeTransitionConfiguration(ctx, cfg)
			s.handleExchangeConfigurationError(err)
			currentEpoch := slots.ToEpoch(slots.CurrentSlot(s.chainStartData.GetGenesisTime()))
			if currentEpoch >= params.BeaconConfig().BellatrixForkEpoch && !hasTtdReached {
				hasTtdReached, err = s.logTtdStatus(ctx, ttd)
				if err != nil {
					log.WithError(err).Error("Could not log ttd status")
				}
			}
			cancel()
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
		if errors.Is(s.runError, ErrConfigMismatch) {
			s.runError = nil
		}
		return
	}
	// If the error is a configuration mismatch, we set a runtime error in the service.
	if errors.Is(err, ErrConfigMismatch) {
		s.runError = err
		log.WithError(err).Error(configMismatchLog)
		return
	}
	log.WithError(err).Error("Could not check configuration values between execution and consensus client")
}

// Logs the terminal total difficulty status.
func (s *Service) logTtdStatus(ctx context.Context, ttd *uint256.Int) (bool, error) {
	latest, err := s.LatestExecutionBlock(ctx)
	switch {
	case errors.Is(err, hexutil.ErrEmptyString):
		return false, nil
	case err != nil:
		return false, err
	case latest == nil:
		return false, errors.New("latest block is nil")
	case latest.TotalDifficulty == "":
		return false, nil
	default:
	}
	latestTtd, err := hexutil.DecodeBig(latest.TotalDifficulty)
	if err != nil {
		return false, err
	}
	if latestTtd.Cmp(ttd.ToBig()) >= 0 {
		return true, nil
	}
	log.WithFields(logrus.Fields{
		"latestDifficulty":   latestTtd.String(),
		"terminalDifficulty": ttd.ToBig().String(),
		"network":            params.BeaconConfig().ConfigName,
	}).Info("Ready for The Merge")

	totalTerminalDifficulty.Set(float64(latestTtd.Uint64()))
	return false, nil
}
