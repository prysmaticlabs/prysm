package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-libp2p"
	filter "github.com/libp2p/go-maddr-filter"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/connmgr"
)

// buildOptions for the libp2p host.
func buildOptions(cfg *Config, ip net.IP, priKey *ecdsa.PrivateKey) []libp2p.Option {
	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, cfg.TCPPort))
	if err != nil {
		log.Fatalf("Failed to p2p listen: %v", err)
	}
	options := []libp2p.Option{
		privKeyOption(priKey),
		libp2p.EnableRelay(),
		libp2p.ListenAddrs(listen),
		whitelistSubnet(cfg.WhitelistCIDR),
		// Add one for the boot node and another for the relay, otherwise when we are close to maxPeers we will be above the high
		// water mark and continually trigger pruning.
		libp2p.ConnectionManager(connmgr.NewConnManager(int(cfg.MaxPeers+2), int(cfg.MaxPeers+2), 1*time.Second)),
	}
	if cfg.EnableUPnP {
		options = append(options, libp2p.NATPortMap()) //Allow to use UPnP
	}
	if cfg.RelayNodeAddr != "" {
		options = append(options, libp2p.AddrsFactory(withRelayAddrs(cfg.RelayNodeAddr)))
	}
	if cfg.HostAddress != "" {
		options = append(options, libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			external, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.HostAddress, cfg.TCPPort))
			if err != nil {
				log.WithError(err).Error("Unable to create external multiaddress")
			} else {
				addrs = append(addrs, external)
			}
			return addrs
		}))
	}
	if cfg.HostDNS != "" {
		options = append(options, libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			external, err := multiaddr.NewMultiaddr(fmt.Sprintf("/dns4/%s/tcp/%d", cfg.HostDNS, cfg.TCPPort))
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
		listen, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.LocalIP, cfg.TCPPort))
		if err != nil {
			log.Fatalf("Failed to p2p listen: %v", err)
		}
		options = append(options, libp2p.ListenAddrs(listen))
	}
	return options
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
