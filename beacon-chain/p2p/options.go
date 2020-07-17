package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	noise "github.com/libp2p/go-libp2p-noise"
	secio "github.com/libp2p/go-libp2p-secio"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/version"
)

// buildOptions for the libp2p host.
func (s *Service) buildOptions(ip net.IP, priKey *ecdsa.PrivateKey) []libp2p.Option {
	cfg := s.cfg
	listen, err := multiAddressBuilder(ip.String(), cfg.TCPPort)
	if err != nil {
		log.Fatalf("Failed to p2p listen: %v", err)
	}
	options := []libp2p.Option{
		privKeyOption(priKey),
		libp2p.ListenAddrs(listen),
		libp2p.UserAgent(version.GetBuildData()),
		libp2p.ConnectionGater(s),
	}
	if featureconfig.Get().EnableNoise {
		// Enable NOISE for the beacon node with secio as a fallback.
		options = append(options, libp2p.Security(noise.ID, noise.New), libp2p.Security(secio.ID, secio.New))
	} else {
		options = append(options, libp2p.Security(secio.ID, secio.New))
	}
	if cfg.EnableUPnP {
		options = append(options, libp2p.NATPortMap()) //Allow to use UPnP
	}
	if cfg.RelayNodeAddr != "" {
		options = append(options, libp2p.AddrsFactory(withRelayAddrs(cfg.RelayNodeAddr)))
		options = append(options, libp2p.EnableRelay())
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

func multiAddressBuilderWithID(ipAddr string, port uint, id peer.ID) (ma.Multiaddr, error) {
	parsedIP := net.ParseIP(ipAddr)
	if parsedIP.To4() == nil && parsedIP.To16() == nil {
		return nil, errors.Errorf("invalid ip address provided: %s", ipAddr)
	}
	if id.String() == "" {
		return nil, errors.New("empty peer id given")
	}
	if parsedIP.To4() != nil {
		return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ipAddr, port, id.String()))
	}
	return ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/%d/p2p/%s", ipAddr, port, id.String()))
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
