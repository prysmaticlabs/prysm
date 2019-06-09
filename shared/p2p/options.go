package p2p

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	filter "github.com/libp2p/go-maddr-filter"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/shared/iputils"
)

// buildOptions for the libp2p host.
// TODO(287): Expand on these options and provide the option configuration via flags.
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
		privKey(cfg.PrvKey),
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

// Adds a private key to the libp2p option if the option was provided.
// If the private key file is missing or cannot be read, or if the
// private key contents cannot be marshaled, an exception is thrown.
func privKey(prvKey string) libp2p.Option {
	if prvKey == "" {
		return func(_ *libp2p.Config) error {
			return nil
		}
	}

	return func(cfg *libp2p.Config) error {
		if _, err := os.Stat(prvKey); os.IsNotExist(err) {
			log.WithField("private key file", prvKey).Warn("Could not read private key, file is missing or unreadable")
			return err
		}
		bytes, err := ioutil.ReadFile(prvKey)
		if err != nil {
			log.WithError(err).Error("Error reading private key from file")
			return err
		}
		keyBytes, err := crypto.ConfigDecodeKey(string(bytes))
		if err != nil {
			log.WithError(err).Error("Error decoding private key")
			return err
		}
		key, err := crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling private key")
			return err
		}
		pubKey, err := peer.IDFromPrivateKey(key)

		if err != nil {
			log.Errorf("Could not print public key: %v", err)
			return err
		}
		log.WithField("public key", pubKey.Pretty()).Info("Private key loaded. Announcing public key.")

		return cfg.Apply(libp2p.Identity(key))
	}
}
