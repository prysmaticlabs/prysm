package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"

	"github.com/btcsuite/btcd/btcec"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/iputils"
	"github.com/sirupsen/logrus"
)

const keyPath = "network-keys"
const metaDataPath = "metaData"

func convertFromInterfacePrivKey(privkey crypto.PrivKey) *ecdsa.PrivateKey {
	typeAssertedKey := (*ecdsa.PrivateKey)((*btcec.PrivateKey)(privkey.(*crypto.Secp256k1PrivateKey)))
	return typeAssertedKey
}

func convertToInterfacePrivkey(privkey *ecdsa.PrivateKey) crypto.PrivKey {
	typeAssertedKey := crypto.PrivKey((*crypto.Secp256k1PrivateKey)((*btcec.PrivateKey)(privkey)))
	return typeAssertedKey
}

func convertToInterfacePubkey(pubkey *ecdsa.PublicKey) crypto.PubKey {
	typeAssertedKey := crypto.PubKey((*crypto.Secp256k1PublicKey)((*btcec.PublicKey)(pubkey)))
	return typeAssertedKey
}

func privKey(cfg *Config) (*ecdsa.PrivateKey, error) {
	defaultKeyPath := path.Join(cfg.DataDir, keyPath)
	privateKeyPath := cfg.PrivateKey

	_, err := os.Stat(defaultKeyPath)
	defaultKeysExist := !os.IsNotExist(err)
	if err != nil && defaultKeysExist {
		return nil, err
	}

	if privateKeyPath == "" && !defaultKeysExist {
		priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			return nil, err
		}
		rawbytes, err := priv.Raw()
		if err != nil {
			return nil, err
		}
		dst := make([]byte, hex.EncodedLen(len(rawbytes)))
		hex.Encode(dst, rawbytes)
		if err = ioutil.WriteFile(defaultKeyPath, dst, 0600); err != nil {
			return nil, err
		}
		convertedKey := convertFromInterfacePrivKey(priv)
		return convertedKey, nil
	}
	if defaultKeysExist && privateKeyPath == "" {
		privateKeyPath = defaultKeyPath
	}
	return retrievePrivKeyFromFile(privateKeyPath)
}

func retrievePrivKeyFromFile(path string) (*ecdsa.PrivateKey, error) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		log.WithError(err).Error("Error reading private key from file")
		return nil, err
	}
	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err = hex.Decode(dst, src)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode hex string")
	}
	unmarshalledKey, err := crypto.UnmarshalSecp256k1PrivateKey(dst)
	if err != nil {
		return nil, err
	}
	return convertFromInterfacePrivKey(unmarshalledKey), nil
}

func metaDataFromConfig(cfg *Config) (*pbp2p.MetaData, error) {
	defaultKeyPath := path.Join(cfg.DataDir, metaDataPath)
	metaDataPath := cfg.MetaDataDir

	_, err := os.Stat(defaultKeyPath)
	defaultMetadataExist := !os.IsNotExist(err)
	if err != nil && defaultMetadataExist {
		return nil, err
	}
	if metaDataPath == "" && !defaultMetadataExist {
		metaData := &pbp2p.MetaData{
			SeqNumber: 0,
			Attnets:   bitfield.NewBitvector64(),
		}
		dst, err := metaData.Marshal()
		if err != nil {
			return nil, err
		}
		if err = ioutil.WriteFile(defaultKeyPath, dst, 0600); err != nil {
			return nil, err
		}
		return metaData, nil
	}
	if defaultMetadataExist && metaDataPath == "" {
		metaDataPath = defaultKeyPath
	}
	src, err := ioutil.ReadFile(metaDataPath)
	if err != nil {
		log.WithError(err).Error("Error reading metadata from file")
		return nil, err
	}
	metaData := &pbp2p.MetaData{}
	if err := metaData.Unmarshal(src); err != nil {
		return nil, err
	}
	return metaData, nil
}

func ipAddr() net.IP {
	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Fatalf("Could not get IPv4 address: %v", err)
	}
	return net.ParseIP(ip)
}

// Attempt to dial an address to verify its connectivity for a protocol{"udp", "tcp"}
func verifyConnectivity(addr string, port uint, protocol string) {
	if addr != "" {
		a := fmt.Sprintf("%s:%d", addr, port)
		fields := logrus.Fields{
			"protocol": protocol,
			"address":  a,
		}
		switch protocol {
		case "tcp":
			conn, err := net.DialTimeout(protocol, a, dialTimeout)
			if err != nil {
				log.WithFields(fields).Warn("IP address is not accessible")
			} else {
				defer conn.Close()
			}
		case "udp":
			addr, err := net.ResolveUDPAddr(protocol, a)
			if err != nil {
				log.Errorf("Could not resolve UDP address: %v", err)
			}
			_, err = net.ListenUDP("udp", addr)
			if err != nil {
				log.WithFields(fields).Warn("IP Address is not accessible")
			}
		default:
			log.Errorf("Error verifying address connectivy. Protocol not supported: %v", protocol)
			return

		}
	}
}
