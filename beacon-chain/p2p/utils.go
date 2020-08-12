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
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/iputils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

const keyPath = "network-keys"
const metaDataPath = "metaData"
const enrPath = "peer.enr"

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

func deserializeENR(raw []byte) (*enr.Record, error) {
	decodedBytes, err := base64.URLEncoding.DecodeString(string(raw))
	if err != nil {
		return nil, err
	}
	record := &enr.Record{}
	rlpStream := rlp.NewStream(bytes.NewBuffer(decodedBytes), enr.SizeLimit)
	err = record.DecodeRLP(rlpStream)
	return record, err
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
		if err = ioutil.WriteFile(defaultKeyPath, dst, params.BeaconIoConfig().ReadWritePermissions); err != nil {
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
	mdPath := cfg.MetaDataDir

	_, err := os.Stat(defaultKeyPath)
	defaultMetadataExist := !os.IsNotExist(err)
	if err != nil && defaultMetadataExist {
		return nil, err
	}
	if mdPath == "" && !defaultMetadataExist {
		metaData := &pbp2p.MetaData{
			SeqNumber: 0,
			Attnets:   bitfield.NewBitvector64(),
		}
		dst, err := metaData.Marshal()
		if err != nil {
			return nil, err
		}
		if err = ioutil.WriteFile(defaultKeyPath, dst, params.BeaconIoConfig().ReadWritePermissions); err != nil {
			return nil, err
		}
		return metaData, nil
	}
	if defaultMetadataExist && mdPath == "" {
		mdPath = defaultKeyPath
	}
	src, err := ioutil.ReadFile(mdPath)
	if err != nil {
		log.WithError(err).Error("Error reading metadata from file")
		return nil, err
	}
	metaData := &pbp2p.MetaData{}
	if err := metaData.Unmarshal(src); err != nil {
		return nil, err
	}
	// Reset Bitfield and increment sequence number
	metaData.Attnets = bitfield.NewBitvector64()
	metaData.SeqNumber = metaData.SeqNumber + 1
	return metaData, nil
}

func enrFromConfig(cfg *Config) (*enr.Record, error) {
	defaultKeyPath := path.Join(cfg.DataDir, enrPath)
	ePath := cfg.ENRDir
	_, err := os.Stat(defaultKeyPath)
	defaultEnrExist := !os.IsNotExist(err)
	if err != nil && defaultEnrExist {
		return nil, err
	}
	if ePath == "" && !defaultEnrExist {
		return nil, nil
	}
	if defaultEnrExist && ePath == "" {
		ePath = defaultKeyPath
	}
	src, err := ioutil.ReadFile(ePath)
	if err != nil {
		log.WithError(err).Error("Error reading metadata from file")
		return nil, err
	}
	return deserializeENR(src)
}

func metadataPathFromCfg(cfg *Config) string {
	defaultKeyPath := path.Join(cfg.DataDir, metaDataPath)
	mdPath := cfg.MetaDataDir

	if mdPath == "" {
		return defaultKeyPath
	}
	return mdPath
}

func enrPathFromCfg(cfg *Config) string {
	defaultKeyPath := path.Join(cfg.DataDir, enrPath)
	ePath := cfg.ENRDir

	if ePath == "" {
		return defaultKeyPath
	}
	return ePath
}

// Processes the previously saved enr and makes sure that they have the same
// pubkey and fork digest before setting the sequence number.
func processENR(prevRecord *enr.Record, node *enode.LocalNode) error {
	nodeKey := &enode.Secp256k1{}
	if err := prevRecord.Load(enr.WithEntry(nodeKey.ENRKey(), nodeKey)); err != nil {
		return err
	}
	pubkey := node.Node().Pubkey()

	// Return if saved ENR has a different pubkey.
	if !(pubkey.X == nodeKey.X && pubkey.Y == nodeKey.Y && pubkey.Curve == nodeKey.Curve) {
		return nil
	}
	firstEntry, err := retrieveForkEntry(prevRecord)
	if err != nil {
		return err
	}
	secondEntry, err := retrieveForkEntry(node.Node().Record())
	if err != nil {
		return err
	}
	if !ssz.DeepEqual(firstEntry, secondEntry) {
		return nil
	}
	// Set ENR sequence to previous entry
	node.Node().Record().SetSeq(prevRecord.Seq() + 1)
	return nil
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
