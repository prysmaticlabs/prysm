package builder

import (
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/network"
	"github.com/prysmaticlabs/prysm/network/authorization"
	"github.com/urfave/cli/v2"
)

type Option func(s *Service) error

// FlagOptions for builder service flag configurations.
func FlagOptions(c *cli.Context) ([]Option, error) {
	endpoints := parseBuilderEndpoints(c)
	opts := []Option{
		WithBuilderEndpoints(endpoints),
	}
	return opts, nil
}

func WithBuilderEndpoints(endpoints []string) Option {
	return func(s *Service) error {
		stringEndpoints := dedupEndpoints(endpoints)
		endpoints := make([]network.Endpoint, len(stringEndpoints))
		for i, e := range stringEndpoints {
			endpoints[i] = covertEndPoint(e)
		}
		s.cfg.builderEndpoint = endpoints[0] // Use the first one as the default.
		return nil
	}
}

func covertEndPoint(ep string) network.Endpoint {
	return network.Endpoint{
		Url: ep,
		Auth: network.AuthorizationData{ // Not sure about authorization for now.
			Method: authorization.None,
			Value:  "",
		}}
}

func parseBuilderEndpoints(c *cli.Context) []string {
	// Goal is to support multiple end points later.
	return []string{c.String(flags.MevBuilderFlag.Name)}
}

func dedupEndpoints(endpoints []string) []string {
	selectionMap := make(map[string]bool)
	newEndpoints := make([]string, 0, len(endpoints))
	for _, point := range endpoints {
		if selectionMap[point] {
			continue
		}
		newEndpoints = append(newEndpoints, point)
		selectionMap[point] = true
	}
	return newEndpoints
}
