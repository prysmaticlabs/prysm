package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/shared/iputils"
)

// buildOptions for the libp2p host.
func buildOptions(cfg *Config) ([]libp2p.Option, net.IP, *ecdsa.PrivateKey) {
	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Fatalf("Could not get IPv4 address: %v", err)
	}
	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, cfg.Port))
	if err != nil {
		log.Fatalf("Failed to p2p listen: %v", err)
	}
	privateKey, err := privKey(cfg.PrivateKey)
	if err != nil {
		log.Fatalf("Could not create private key %v", err)
	}
	peerStore := peerstore.NewPeerstore(
		pstoremem.NewKeyBook(),
		pstoremem.NewAddrBook(),
		pstoremem.NewProtoBook(),
		pstoremem.NewPeerMetadata(),
	)
	options := []libp2p.Option{
		libp2p.ListenAddrs(listen),
		privKeyOption(privateKey),
		libp2p.Peerstore(peerStore),
	}
	if cfg.EnableUPnP {
		options = append(options, libp2p.NATPortMap()) //Allow to use UPnP
	}
	// add discv5 to list of protocols in libp2p.
	if err := addDiscv5protocol(); err != nil {
		log.Fatalf("Could not set add discv5 to libp2p protocols: %v", err)
	}
	return options, net.ParseIP(ip), privateKey
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
