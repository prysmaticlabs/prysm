package p2p

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"

	curve "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/shared/iputils"
)

// buildOptions for the libp2p host.
func buildOptions(cfg *Config) []libp2p.Option {

	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Errorf("Could not get IPv4 address: %v", err)
	}

	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, cfg.Port))
	if err != nil {
		log.Errorf("Failed to p2p listen: %v", err)
	}

	options := []libp2p.Option{
		libp2p.ListenAddrs(listen),
		privKey(cfg.PrivateKey),
	}

	if cfg.EnableUPnP {
		options = append(options, libp2p.NATPortMap()) //Allow to use UPnP
	}

	return options
}

// Adds a private key to the libp2p option if the option was provided.
// If the private key file is missing or cannot be read, or if the
// private key contents cannot be marshaled, an exception is thrown.
func privKey(prvKey string) libp2p.Option {
	if prvKey == "" {
		return func(cfg *libp2p.Config) error {
			privKey, pubKey, err := crypto.GenerateECDSAKeyPairWithCurve(curve.S256(), rand.Reader)
			if err != nil {
				return err
			}
			id, err := peer.IDFromPublicKey(pubKey)
			if err != nil {
				return err
			}
			log.WithField("peer id", id.Pretty()).Info("Private key generated. Announcing peer id")
			return cfg.Apply(libp2p.Identity(privKey))
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
