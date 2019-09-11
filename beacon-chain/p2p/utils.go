package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"io/ioutil"
	"net"

	"github.com/btcsuite/btcd/btcec"
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
	privateKey, err := ioutil.ReadFile(cfg.PrivateKey)
	if err != nil {
		log.WithError(err).Error("Error reading private key from file")
		return nil, err
	}
	b, err := crypto.ConfigDecodeKey(string(privateKey))
	if err != nil {
		panic(err)
	}
	unmarshalledKey, err := crypto.UnmarshalPrivateKey(b)
	if err != nil {
		panic(err)
	}
	priv := (*ecdsa.PrivateKey)((*btcec.PrivateKey)(unmarshalledKey.(*crypto.Secp256k1PrivateKey)))
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
