package powchain

import (
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/network"
)

// DefaultRPCHTTPTimeout for HTTP requests via an RPC connection to an execution node.
const DefaultRPCHTTPTimeout = time.Second * 6

type Option func(s *Service) error

// WithHttpEndpoints deduplicates and parses http endpoints for the powchain service to use,
// and sets the "current" endpoint that will be used first.
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

// WithJWTSecret for authenticating the execution node JSON-RPC endpoint.
func WithJWTSecret(secret []byte) Option {
	return func(c *Service) error {
		if len(secret) == 0 {
			return nil
		}
		authTransport := &jwtTransport{
			underlyingTransport: http.DefaultTransport,
			jwtSecret:           secret,
		}
		c.cfg.httpRPCClient = &http.Client{
			Timeout:   DefaultRPCHTTPTimeout,
			Transport: authTransport,
		}
		return nil
	}
}

// WithDepositContractAddress for the deposit contract.
func WithDepositContractAddress(addr common.Address) Option {
	return func(s *Service) error {
		s.cfg.depositContractAddr = addr
		return nil
	}
}

// WithDatabase for the beacon chain database.
func WithDatabase(database db.HeadAccessDatabase) Option {
	return func(s *Service) error {
		s.cfg.beaconDB = database
		return nil
	}
}

// WithDepositCache for caching deposits.
func WithDepositCache(cache *depositcache.DepositCache) Option {
	return func(s *Service) error {
		s.cfg.depositCache = cache
		return nil
	}
}

// WithStateNotifier for subscribing to state changes.
func WithStateNotifier(notifier statefeed.Notifier) Option {
	return func(s *Service) error {
		s.cfg.stateNotifier = notifier
		return nil
	}
}

// WithStateGen to regenerate beacon states from checkpoints.
func WithStateGen(gen *stategen.State) Option {
	return func(s *Service) error {
		s.cfg.stateGen = gen
		return nil
	}
}

// WithEth1HeaderRequestLimit to set the upper limit of eth1 header requests.
func WithEth1HeaderRequestLimit(limit uint64) Option {
	return func(s *Service) error {
		s.cfg.eth1HeaderReqLimit = limit
		return nil
	}
}

// WithBeaconNodeStatsUpdater to set the beacon node stats updater.
func WithBeaconNodeStatsUpdater(updater BeaconNodeStatsUpdater) Option {
	return func(s *Service) error {
		s.cfg.beaconNodeStatsUpdater = updater
		return nil
	}
}

// WithFinalizedStateAtStartup to set the beacon node's finalized state at startup.
func WithFinalizedStateAtStartup(st state.BeaconState) Option {
	return func(s *Service) error {
		s.cfg.finalizedStateAtStartup = st
		return nil
	}
}
