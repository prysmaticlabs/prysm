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

	gcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/network"
	pb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/metadata"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const keyPath = "network-keys"
const metaDataPath = "metaData"

const dialTimeout = 1 * time.Second

// SerializeENR takes the enr record in its key-value form and serializes it.
func SerializeENR(record *enr.Record) (string, error) {
	if record == nil {
		return "", errors.New("could not serialize nil record")
	}
	buf := bytes.NewBuffer([]byte{})
	if err := record.EncodeRLP(buf); err != nil {
		return "", errors.Wrap(err, "could not encode ENR record to bytes")
	}
	enrString := base64.URLEncoding.EncodeToString(buf.Bytes())
	return enrString, nil
}

func convertFromInterfacePrivKey(privkey crypto.PrivKey) *ecdsa.PrivateKey {
	typeAssertedKey := (*ecdsa.PrivateKey)(privkey.(*crypto.Secp256k1PrivateKey))
	typeAssertedKey.Curve = gcrypto.S256() // Temporary hack, so libp2p Secp256k1 is recognized as geth Secp256k1 in disc v5.1.
	return typeAssertedKey
}

func convertToInterfacePrivkey(privkey *ecdsa.PrivateKey) crypto.PrivKey {
	typeAssertedKey := crypto.PrivKey((*crypto.Secp256k1PrivateKey)(privkey))
	return typeAssertedKey
}

func convertToInterfacePubkey(pubkey *ecdsa.PublicKey) crypto.PubKey {
	typeAssertedKey := crypto.PubKey((*crypto.Secp256k1PublicKey)(pubkey))
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
		convertedKey := convertFromInterfacePrivKey(priv)
		return convertedKey, nil
	}
	if defaultKeysExist && privateKeyPath == "" {
		privateKeyPath = defaultKeyPath
	}
	return privKeyFromFile(privateKeyPath)
}

// Retrieves a p2p networking private key from a file path.
func privKeyFromFile(path string) (*ecdsa.PrivateKey, error) {
	src, err := ioutil.ReadFile(path) // #nosec G304
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
// TODO: Figure out how to do a v1/v2 check.
func metaDataFromConfig(cfg *Config) (metadata.Metadata, error) {
	defaultKeyPath := path.Join(cfg.DataDir, metaDataPath)
	metaDataPath := cfg.MetaDataDir

	_, err := os.Stat(defaultKeyPath)
	defaultMetadataExist := !os.IsNotExist(err)
	if err != nil && defaultMetadataExist {
		return nil, err
	}
	if metaDataPath == "" && !defaultMetadataExist {
		metaData := &pb.MetaDataV0{
			SeqNumber: 0,
			Attnets:   bitfield.NewBitvector64(),
		}
		dst, err := proto.Marshal(metaData)
		if err != nil {
			return nil, err
		}
		if err := file.WriteFile(defaultKeyPath, dst); err != nil {
			return nil, err
		}
		return wrapper.WrappedMetadataV0(metaData), nil
	}
	if defaultMetadataExist && metaDataPath == "" {
		metaDataPath = defaultKeyPath
	}
	src, err := ioutil.ReadFile(metaDataPath) // #nosec G304
	if err != nil {
		log.WithError(err).Error("Error reading metadata from file")
		return nil, err
	}
	metaData := &pb.MetaDataV0{}
	if err := proto.Unmarshal(src, metaData); err != nil {
		return nil, err
	}
	return wrapper.WrappedMetadataV0(metaData), nil
}

// Retrieves an external ipv4 address and converts into a libp2p formatted value.
func ipAddr() net.IP {
	ip, err := network.ExternalIP()
	if err != nil {
		log.Fatalf("Could not get IPv4 address: %v", err)
	}
	return net.ParseIP(ip)
}

// Attempt to dial an address to verify its connectivity
func verifyConnectivity(addr string, port uint, protocol string) {
	if addr != "" {
		a := net.JoinHostPort(addr, fmt.Sprintf("%d", port))
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
