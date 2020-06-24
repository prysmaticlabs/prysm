package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/iputils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

const keyPath = "network-keys"
const metaDataPath = "metaData"

const dialTimeout = 1 * time.Second

// SerializeENR takes the enr record in its key-value form and serializes it.
func SerializeENR(record *enr.Record) (string, error) {
	buf := bytes.NewBuffer([]byte{})
	if err := record.EncodeRLP(buf); err != nil {
		return "", errors.Wrap(err, "could not encode ENR record to bytes")
	}
	enrString := base64.URLEncoding.EncodeToString(buf.Bytes())
	return enrString, nil
}

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

// Determines a private key for p2p networking from the p2p service's
// configuration struct. If no key is found, it generates a new one.
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
		if err = ioutil.WriteFile(defaultKeyPath, dst, params.BeaconIoConfig().FilePermission); err != nil {
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

// Retrieves a p2p networking private key from a file path.
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

// Retrieves node p2p metadata from a set of configuration values
// from the p2p service.
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
		if err = ioutil.WriteFile(defaultKeyPath, dst, params.BeaconIoConfig().FilePermission); err != nil {
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

// Retrieves an external ipv4 address and converts into a libp2p formatted value.
func ipAddr() net.IP {
	ip, err := iputils.ExternalIPv4()
	if err != nil {
		log.Fatalf("Could not get IPv4 address: %v", err)
	}
	return net.ParseIP(ip)
}

// Attempt to dial an address to verify its connectivity
func verifyConnectivity(addr string, port uint, protocol string) {
	if addr != "" {
		a := fmt.Sprintf("%s:%d", addr, port)
		fields := logrus.Fields{
			"protocol": protocol,
			"address":  a,
		}
		conn, err := net.DialTimeout(protocol, a, dialTimeout)
		if err != nil {
			log.WithError(err).WithFields(fields).Warn("IP address is not accessible")
			return
		}
		if err := conn.Close(); err != nil {
			log.WithError(err).Debug("Could not close connection")
		}
	}
}
