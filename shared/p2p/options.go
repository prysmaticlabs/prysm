package p2p

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"

	peer "github.com/libp2p/go-libp2p-peer"

	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/prysmaticlabs/prysm/shared/iputils"

	filter "github.com/libp2p/go-maddr-filter"
	ma "github.com/multiformats/go-multiaddr"
)

const defaultPrivateKeyFile = "identity.key"

// buildOptions for the libp2p host.
// TODO(287): Expand on these options and provide the option configuration via flags.
func buildOptions(cfg *ServerConfig) ([]libp2p.Option, error) {
	if cfg.PrvKey == "" {
		cfg.PrvKey = defaultPrivateKeyFile
	}
	var prvKey crypto.PrivKey
	if _, err := os.Stat(cfg.PrvKey); os.IsNotExist(err) {
		log.Warn("Could not read private key, file is missing. Generating a new private key.", cfg.PrvKey)
		prvKey, _, err = crypto.GenerateKeyPair(crypto.RSA, 2048)
		if err != nil {
			log.WithError(err).Error("Unable to generate a private key")
			return nil, err
		}
		bytes, err := crypto.MarshalPrivateKey(prvKey)
		if err != nil {
			log.WithError(err).Error("Unable to marshall private key")
			return nil, err
		}
		err = ioutil.WriteFile(cfg.PrvKey, bytes, 0600)
		if err != nil {
			log.WithError(err).Error("Error writing private key to file")
			return nil, err
		}

	} else {
		bytes, err := ioutil.ReadFile(cfg.PrvKey)
		if err != nil {
			log.WithError(err).Error("Error reading private key from file")
			return nil, err
		}
		prvKey, err = crypto.UnmarshalPrivateKey(bytes)
		if err != nil {
			log.WithError(err).Error("Error unmarshalling private key")
			return nil, err
		}
	}
	pubKey, err := peer.IDFromPrivateKey(prvKey)
	if err != nil {
		log.Errorf("Could not print public key: %v", err)
		return nil, err
	}
	log.WithField("public key", pubKey.Pretty()).Info("Private key loaded. Announcing public key.")
	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Errorf("Could not get IPv4 address: %v", err)
		return nil, err
	}

	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, cfg.Port))
	if err != nil {
		log.Errorf("Failed to p2p listen: %v", err)
		return nil, err
	}

	return []libp2p.Option{
		libp2p.Identity(prvKey),
		libp2p.ListenAddrs(listen),
		libp2p.EnableRelay(), // Allows dialing to peers via relay.
		optionConnectionManager(cfg.MaxPeers),
		whitelistSubnet(cfg.WhitelistCIDR),
	}, nil
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
