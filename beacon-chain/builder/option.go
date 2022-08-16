package builder

import (
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v3/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v3/network"
	"github.com/prysmaticlabs/prysm/v3/network/authorization"
	"github.com/urfave/cli/v2"
)

type Option func(s *Service) error

// FlagOptions for builder service flag configurations.
func FlagOptions(c *cli.Context) ([]Option, error) {
	endpoint := c.String(flags.MevRelayEndpoint.Name)
	opts := []Option{
		WithBuilderEndpoints(endpoint),
	}
	return opts, nil
}

// WithBuilderEndpoints sets the endpoint for the beacon chain builder service.
func WithBuilderEndpoints(endpoint string) Option {
	return func(s *Service) error {
		s.cfg.builderEndpoint = covertEndPoint(endpoint)
		return nil
	}
}

// WithHeadFetcher gets the head info from chain service.
func WithHeadFetcher(svc *blockchain.Service) Option {
	return func(s *Service) error {
		s.cfg.headFetcher = svc
		return nil
	}
}

// WithDatabase for head access.
func WithDatabase(beaconDB db.HeadAccessDatabase) Option {
	return func(s *Service) error {
		s.cfg.beaconDB = beaconDB
		return nil
	}
}

func covertEndPoint(ep string) network.Endpoint {
	return network.Endpoint{
		Url: ep,
		Auth: network.AuthorizationData{ // Auth is not used for builder.
			Method: authorization.None,
			Value:  "",
		}}
}
