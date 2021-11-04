package powchain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/network"
)

type Option func(s *Service) error

// WithHttpEndpoints --
func WithHttpEndpoints(endpointStrings []string) Option {
	return func(s *Service) error {
		stringEndpoints := dedupEndpoints(endpointStrings)
		endpoints := make([]network.Endpoint, len(stringEndpoints))
		for i, e := range stringEndpoints {
			endpoints[i] = HttpEndpoint(e)
		}
		// Select first http endpoint in the provided list.
		var currEndpoint network.Endpoint
		if len(endpointStrings) > 0 {
			currEndpoint = endpoints[0]
		}
		s.cfg.httpEndpoints = endpoints
		s.cfg.currHttpEndpoint = currEndpoint
		return nil
	}
}

// WithHttpEndpoints --
func WithDepositContractAddress(addr common.Address) Option {
	return func(s *Service) error {
		s.cfg.depositContractAddr = addr
		return nil
	}
}

// WithHttpEndpoints --
func WithDatabase(database db.HeadAccessDatabase) Option {
	return func(s *Service) error {
		s.cfg.beaconDB = database
		return nil
	}
}

// WithHttpEndpoints --
func WithDepositCache(cache *depositcache.DepositCache) Option {
	return func(s *Service) error {
		s.cfg.depositCache = cache
		return nil
	}
}

// WithHttpEndpoints --
func WithStateNotifier(notifier statefeed.Notifier) Option {
	return func(s *Service) error {
		s.cfg.stateNotifier = notifier
		return nil
	}
}

// WithHttpEndpoints --
func WithStateGen(gen *stategen.State) Option {
	return func(s *Service) error {
		s.cfg.stateGen = gen
		return nil
	}
}

// WithHttpEndpoints --
func WithEth1HeaderRequestLimit(limit uint64) Option {
	return func(s *Service) error {
		s.cfg.eth1HeaderReqLimit = limit
		return nil
	}
}

// WithHttpEndpoints --
func WithBeaconNodeStatsUpdater(updater BeaconNodeStatsUpdater) Option {
	return func(s *Service) error {
		s.cfg.beaconNodeStatsUpdater = updater
		return nil
	}
}
