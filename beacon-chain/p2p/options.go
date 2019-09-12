package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// buildOptions for the libp2p host.
func buildOptions(cfg *Config, ip net.IP, priKey *ecdsa.PrivateKey) []libp2p.Option {
	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, cfg.TCPPort))
	if err != nil {
		log.Fatalf("Failed to p2p listen: %v", err)
	}
	options := []libp2p.Option{
		privKeyOption(priKey),
		libp2p.ListenAddrs(listen),
	}
	if cfg.EnableUPnP {
		options = append(options, libp2p.NATPortMap()) //Allow to use UPnP
	}
	return options
}

// Adds a private key to the libp2p option if the option was provided.
// If the private key file is missing or cannot be read, or if the
// private key contents cannot be marshaled, an exception is thrown.
func privKeyOption(privkey *ecdsa.PrivateKey) libp2p.Option {
	return func(cfg *libp2p.Config) error {
		convertedKey := convertToInterfacePrivkey(privkey)
		id, err := peer.IDFromPrivateKey(convertedKey)
		if err != nil {
			return err
		}
		log.WithField("peer id", id.Pretty()).Info("Private key generated. Announcing peer id")
		return cfg.Apply(libp2p.Identity(convertedKey))
	}
}
