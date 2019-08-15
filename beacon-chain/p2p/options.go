package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
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
func buildOptions(cfg *Config) ([]libp2p.Option, string, *ecdsa.PrivateKey) {

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

	options := []libp2p.Option{
		libp2p.ListenAddrs(listen),
		privKeyOption(privateKey),
	}

	if cfg.EnableUPnP {
		options = append(options, libp2p.NATPortMap()) //Allow to use UPnP
	}

	return options, ip, privateKey
}

func privKey(prvKey string) (*ecdsa.PrivateKey, error) {
	if prvKey == "" {
		priv, err := ecdsa.GenerateKey(curve.S256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		return priv, nil
	}

	if _, err := os.Stat(prvKey); os.IsNotExist(err) {
		log.WithField("private key file", prvKey).Warn("Could not read private key, file is missing or unreadable")
		return nil, err
	}
	bytes, err := ioutil.ReadFile(prvKey)
	if err != nil {
		log.WithError(err).Error("Error reading private key from file")
		return nil, err
	}
	keyBytes, err := crypto.ConfigDecodeKey(string(bytes))
	if err != nil {
		log.WithError(err).Error("Error decoding private key")
		return nil, err
	}
	priv, err := x509.ParseECPrivateKey(keyBytes)
	if err != nil {
		log.WithError(err).Error("Error unmarshalling private key")
		return nil, err
	}

	return priv, nil
}

// Adds a private key to the libp2p option if the option was provided.
// If the private key file is missing or cannot be read, or if the
// private key contents cannot be marshaled, an exception is thrown.
func privKeyOption(prvKey *ecdsa.PrivateKey) libp2p.Option {
	return func(cfg *libp2p.Config) error {
		privKey, pubKey, err := crypto.ECDSAKeyPairFromKey(prvKey)
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
