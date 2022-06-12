package builder

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/network"
	"github.com/prysmaticlabs/prysm/network/authorization"
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

// WithDatabase sets the database for the beacon chain builder service.
func WithDatabase(database db.HeadAccessDatabase) Option {
	return func(s *Service) error {
		s.cfg.beaconDB = database
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
