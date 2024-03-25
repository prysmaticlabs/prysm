package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-libp2p"
	mplex "github.com/libp2p/go-libp2p-mplex"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	gomplex "github.com/libp2p/go-mplex"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	ecdsaprysm "github.com/prysmaticlabs/prysm/v5/crypto/ecdsa"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// MultiAddressBuilder takes in an ip address string and port to produce a go multiaddr format.
func MultiAddressBuilder(ipAddr string, port uint) (ma.Multiaddr, error) {
	parsedIP := net.ParseIP(ipAddr)
	if parsedIP.To4() == nil && parsedIP.To16() == nil {
		return nil, errors.Errorf("invalid ip address provided: %s", ipAddr)
	}
	if parsedIP.To4() != nil {
		return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ipAddr, port))
	}
	return ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/%d", ipAddr, port))
}

// buildOptions for the libp2p host.
func (s *Service) buildOptions(ip net.IP, priKey *ecdsa.PrivateKey) ([]libp2p.Option, error) {
	cfg := s.cfg
	listen, err := MultiAddressBuilder(ip.String(), cfg.TCPPort)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot produce multiaddr format from %s:%d", ip.String(), cfg.TCPPort)
	}
	if cfg.LocalIP != "" {
		if net.ParseIP(cfg.LocalIP) == nil {
			return nil, errors.Wrapf(err, "invalid local ip provided: %s:%d", cfg.LocalIP, cfg.TCPPort)
		}

		listen, err = MultiAddressBuilder(cfg.LocalIP, cfg.TCPPort)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot produce multiaddr format from %s:%d", cfg.LocalIP, cfg.TCPPort)
		}
	}
	ifaceKey, err := ecdsaprysm.ConvertToInterfacePrivkey(priKey)
	if err != nil {
		return nil, errors.Wrap(err, "cannot convert private key to interface private key. (Private key not displayed in logs for security reasons)")
	}
	id, err := peer.IDFromPublicKey(ifaceKey.GetPublic())
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get ID from public key: %s", ifaceKey.GetPublic().Type().String())
	}

	log.Infof("Running node with peer id of %s ", id.String())

	options := []libp2p.Option{
		privKeyOption(priKey),
		libp2p.ListenAddrs(listen),
		libp2p.UserAgent(version.BuildData()),
		libp2p.ConnectionGater(s),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.DefaultMuxers,
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Ping(false), // Disable Ping Service.
	}

	if cfg.EnableUPnP {
		options = append(options, libp2p.NATPortMap()) // Allow to use UPnP
	}
	if cfg.RelayNodeAddr != "" {
		options = append(options, libp2p.AddrsFactory(withRelayAddrs(cfg.RelayNodeAddr)))
	} else {
		// Disable relay if it has not been set.
		options = append(options, libp2p.DisableRelay())
	}
	if cfg.HostAddress != "" {
		options = append(options, libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
			external, err := MultiAddressBuilder(cfg.HostAddress, cfg.TCPPort)
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

	if features.Get().DisableResourceManager {
		options = append(options, libp2p.ResourceManager(&network.NullResourceManager{}))
	}
	return options, nil
}

func multiAddressBuilderWithID(ipAddr, protocol string, port uint, id peer.ID) (ma.Multiaddr, error) {
	parsedIP := net.ParseIP(ipAddr)
	if parsedIP.To4() == nil && parsedIP.To16() == nil {
		return nil, errors.Errorf("invalid ip address provided: %s", ipAddr)
	}
	if id.String() == "" {
		return nil, errors.New("empty peer id given")
	}
	if parsedIP.To4() != nil {
		return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/%s/%d/p2p/%s", ipAddr, protocol, port, id.String()))
	}
	return ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/%s/%d/p2p/%s", ipAddr, protocol, port, id.String()))
}

// Adds a private key to the libp2p option if the option was provided.
// If the private key file is missing or cannot be read, or if the
// private key contents cannot be marshaled, an exception is thrown.
func privKeyOption(privkey *ecdsa.PrivateKey) libp2p.Option {
	return func(cfg *libp2p.Config) error {
		ifaceKey, err := ecdsaprysm.ConvertToInterfacePrivkey(privkey)
		if err != nil {
			return err
		}
		log.Debug("ECDSA private key generated")
		return cfg.Apply(libp2p.Identity(ifaceKey))
	}
}

// Configures stream timeouts on mplex.
func configureMplex() {
	gomplex.ResetStreamTimeout = 5 * time.Second
}
