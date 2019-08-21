package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"net"
	"os"

	"github.com/btcsuite/btcd/btcec"
	curve "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prysmaticlabs/prysm/shared/iputils"
)

func convertFromInterfacePrivKey(privkey crypto.PrivKey) *ecdsa.PrivateKey {
	typeAssertedKey := (*ecdsa.PrivateKey)((*btcec.PrivateKey)(privkey.(*crypto.Secp256k1PrivateKey)))
	return typeAssertedKey
}

func convertToInterfacePrivkey(privkey *ecdsa.PrivateKey) crypto.PrivKey {
	typeAssertedKey := crypto.PrivKey((*crypto.Secp256k1PrivateKey)((*btcec.PrivateKey)(privkey)))
	return typeAssertedKey
}

func convertFromInterfacePubKey(pubkey crypto.PubKey) *ecdsa.PublicKey {
	typeAssertedKey := (*ecdsa.PublicKey)((*btcec.PublicKey)(pubkey.(*crypto.Secp256k1PublicKey)))
	return typeAssertedKey
}

func convertToInterfacePubkey(pubkey *ecdsa.PublicKey) crypto.PubKey {
	typeAssertedKey := crypto.PubKey((*crypto.Secp256k1PublicKey)((*btcec.PublicKey)(pubkey)))
	return typeAssertedKey
}

func privKey(cfg *Config) (*ecdsa.PrivateKey, error) {
	if cfg.PrivateKey == "" {
		priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			return nil, err
		}
		convertedKey := convertFromInterfacePrivKey(priv)
		return convertedKey, nil
	}
	if _, err := os.Stat(cfg.PrivateKey); os.IsNotExist(err) {
		log.WithField("private key file", cfg.PrivateKey).Warn("Could not read private key, file is missing or unreadable")
		return nil, err
	}
	priv, err := curve.LoadECDSA(cfg.PrivateKey)
	if err != nil {
		log.WithError(err).Error("Error reading private key from file")
		return nil, err
	}
	return priv, nil
}

func ipAddr(cfg *Config) net.IP {
	if cfg.HostAddress == "" {
		ip, err := iputils.ExternalIPv4()
		if err != nil {
			log.Fatalf("Could not get IPv4 address: %v", err)
		}
		return net.ParseIP(ip)
	}
	return net.ParseIP(cfg.HostAddress)
}
