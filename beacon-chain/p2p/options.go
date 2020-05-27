package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-libp2p"
	noise "github.com/libp2p/go-libp2p-noise"
	filter "github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/connmgr"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
)

// buildOptions for the libp2p host.
func buildOptions(cfg *Config, ip net.IP, priKey *ecdsa.PrivateKey) []libp2p.Option {
	listen, err := multiAddressBuilder(ip.String(), cfg.TCPPort)
	if err != nil {
		log.Fatalf("Failed to p2p listen: %v", err)
	}
	options := []libp2p.Option{
		privKeyOption(priKey),
		libp2p.EnableRelay(),
		libp2p.ListenAddrs(listen),
		whitelistSubnet(cfg.WhitelistCIDR),
		blacklistSubnets(cfg.BlacklistCIDR),
		// Add one for the boot node and another for the relay, otherwise when we are close to maxPeers we will be above the high
		// water mark and continually trigger pruning.
		libp2p.ConnectionManager(connmgr.NewConnManager(int(cfg.MaxPeers+2), int(cfg.MaxPeers+2), 1*time.Second)),
	}
	if featureconfig.Get().EnableNoise {
		// Enable NOISE for the beacon node
		options = append(options, libp2p.Security(noise.ID, noise.New))
	}
	if cfg.EnableUPnP {
		options = append(options, libp2p.NATPortMap()) //Allow to use UPnP
	}
	if cfg.RelayNodeAddr != "" {
		options = append(options, libp2p.AddrsFactory(withRelayAddrs(cfg.RelayNodeAddr)))
	}
	if cfg.HostAddress != "" {
		options = append(options, libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
			external, err := multiAddressBuilder(cfg.HostAddress, cfg.TCPPort)
			if err != nil {
				log.WithError(err).Error("Unable to create external multiaddress")
			} else {
				addrs = append(addrs, external)
			}
			return addrs
		}))
	}
	if cfg.HostDNS != "" {
		options = append(options, libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
			external, err := ma.NewMultiaddr(fmt.Sprintf("/dns4/%s/tcp/%d", cfg.HostDNS, cfg.TCPPort))
			if err != nil {
				log.WithError(err).Error("Unable to create external multiaddress")
			} else {
				addrs = append(addrs, external)
			}
			return addrs
		}))
	}
	if cfg.LocalIP != "" {
		if net.ParseIP(cfg.LocalIP) == nil {
			log.Errorf("Invalid local ip provided: %s", cfg.LocalIP)
			return options
		}
		listen, err = multiAddressBuilder(cfg.LocalIP, cfg.TCPPort)
		if err != nil {
			log.Fatalf("Failed to p2p listen: %v", err)
		}
		options = append(options, libp2p.ListenAddrs(listen))
	}
	return options
}

func multiAddressBuilder(ipAddr string, port uint) (ma.Multiaddr, error) {
	parsedIP := net.ParseIP(ipAddr)
	if parsedIP.To4() == nil && parsedIP.To16() == nil {
		return nil, errors.Errorf("invalid ip address provided: %s", ipAddr)
	}
	if parsedIP.To4() != nil {
		return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, port))
	}
	return ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/%d", ipAddr, port))
}

// Adds a private key to the libp2p option if the option was provided.
// If the private key file is missing or cannot be read, or if the
// private key contents cannot be marshaled, an exception is thrown.
func privKeyOption(privkey *ecdsa.PrivateKey) libp2p.Option {
	return func(cfg *libp2p.Config) error {
		log.Debug("ECDSA private key generated")
		return cfg.Apply(libp2p.Identity(convertToInterfacePrivkey(privkey)))
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
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return func(_ *libp2p.Config) error {
			return err
		}
	}
	filters := filter.NewFilters()
	filters.AddFilter(*ipnet, filter.ActionAccept)

	return libp2p.Filters(filters)
}

// blacklistSubnet adds a blacklist multiaddress filter for multiple given CIDR subnets.
// Example: 192.168.0.0/16 may be used to deny connections from your local
// network.
func blacklistSubnets(mulCidrs []string) libp2p.Option {
	if len(mulCidrs) == 0 {
		return func(_ *libp2p.Config) error {
			return nil
		}
	}
	ipNets := []*net.IPNet{}
	for _, cidr := range mulCidrs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return func(_ *libp2p.Config) error {
				return err
			}
		}
		ipNets = append(ipNets, ipnet)
	}
	return libp2p.FilterAddresses(ipNets...)
}
