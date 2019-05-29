package p2p

import (
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p"
	filter "github.com/libp2p/go-maddr-filter"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/prysmaticlabs/prysm/shared/iputils"
)

// buildOptions for the libp2p host.
// TODO(287): Expand on these options and provide the option configuration via flags.
// Currently, this is a random port and a (seemingly) consistent private key
// identity.
func buildOptions(cfg *ServerConfig) []libp2p.Option {
	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Errorf("Could not get IPv4 address: %v", err)
	}

	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, cfg.Port))
	if err != nil {
		log.Errorf("Failed to p2p listen: %v", err)
	}

	return []libp2p.Option{
		libp2p.ListenAddrs(listen),
		libp2p.EnableRelay(), // Allows dialing to peers via relay.
		optionConnectionManager(cfg.MaxPeers),
		whitelistSubnet(cfg.WhitelistCIDR),
	}
}

// whitelistSubnet adds a whitelist multiaddress filter for a given CIDR subnet.
// Example: 192.168.0.0/16 may be used to accept only connections on your local
// network.
func whitelistSubnet(cidr string) libp2p.Option {
	if cidr == "" {
		return func(_ *libp2p.Config) error {
			return nil
		}
	}

	return func(cfg *libp2p.Config) error {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}

		if cfg.Filters == nil {
			cfg.Filters = filter.NewFilters()
		}
		cfg.Filters.AddFilter(*ipnet, filter.ActionAccept)

		return nil
	}
}
