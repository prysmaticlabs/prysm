package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/pkg/errors"
	ssz "github.com/prysmaticlabs/fastssz"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/wrapper"
	ecdsaprysm "github.com/prysmaticlabs/prysm/v5/crypto/ecdsa"
	"github.com/prysmaticlabs/prysm/v5/io/file"
	pb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/metadata"
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
	enrString := base64.RawURLEncoding.EncodeToString(buf.Bytes())
	return enrString, nil
}

// Determines a private key for p2p networking from the p2p service's
// configuration struct. If no key is found, it generates a new one.
func privKey(cfg *Config) (*ecdsa.PrivateKey, error) {
	defaultKeyPath := path.Join(cfg.DataDir, keyPath)
	privateKeyPath := cfg.PrivateKey

	// PrivateKey cli flag takes highest precedence.
	if privateKeyPath != "" {
		return privKeyFromFile(cfg.PrivateKey)
	}

	// Default keys have the next highest precedence, if they exist.
	_, err := os.Stat(defaultKeyPath)
	defaultKeysExist := !os.IsNotExist(err)
	if err != nil && defaultKeysExist {
		return nil, err
	}

	if defaultKeysExist {
		return privKeyFromFile(defaultKeyPath)
	}

	// There are no keys on the filesystem, so we need to generate one.
	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return nil, err
	}

	// If the StaticPeerID flag is not set, return the private key.
	if !cfg.StaticPeerID {
		return ecdsaprysm.ConvertFromInterfacePrivKey(priv)
	}

	// Save the generated key as the default key, so that it will be used by
	// default on the next node start.
	rawbytes, err := priv.Raw()
	if err != nil {
		return nil, err
	}

	dst := make([]byte, hex.EncodedLen(len(rawbytes)))
	hex.Encode(dst, rawbytes)
	if err := file.WriteFile(defaultKeyPath, dst); err != nil {
		return nil, err
	}

	log.Info("Wrote network key to file")
	// Read the key from the defaultKeyPath file just written
	// for the strongest guarantee that the next start will be the same as this one.
	return privKeyFromFile(defaultKeyPath)
}

// Retrieves a p2p networking private key from a file path.
func privKeyFromFile(path string) (*ecdsa.PrivateKey, error) {
	src, err := os.ReadFile(path) // #nosec G304
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
	return ecdsaprysm.ConvertFromInterfacePrivKey(unmarshalledKey)
}

// Retrieves node p2p metadata from a set of configuration values
// from the p2p service.
// When using static peer id, metaDataFromConfig returns default V1 metadata.
func metaDataFromConfig(cfg *Config) (metadata.Metadata, error) {
	// Load V1 metadata by default, since V1 metadata can be covered to V0.
	defaultMd := &pb.MetaDataV1{
		SeqNumber: 0,
		Attnets:   bitfield.NewBitvector64(),
		Syncnets:  bitfield.NewBitvector4(),
	}
	wrappedDefaultMd := wrapper.WrappedMetadataV1(defaultMd)

	// If --p2p-static-id is false, return default metadata for initialization
	if !cfg.StaticPeerID {
		return wrappedDefaultMd, nil
	}

	mdPath, exist := resolveMetaDataPath(cfg)
	if exist {
		md, err := metaDataFromFile(mdPath)
		if err != nil {
			if errors.Is(err, ssz.ErrSize) {
				// In case previous metadata file is encoded by proto,
				// we need to migrate it into ssz encoded version.
				return migrateFromProtoToSsz(mdPath)
			}
			return nil, err
		}
		return md, err
	}
	if err := saveMetaDataToFile(mdPath, wrappedDefaultMd); err != nil {
		return nil, err
	}

	return wrappedDefaultMd, nil
}

// resolveMetaDataPath returns path and the existence of that path.
// Issue while opening a file(e.g. permission issues) will be handled at metaDataFromFile.
func resolveMetaDataPath(cfg *Config) (string, bool) {
	var mdPath string

	// Prioritize if --p2p-metadata is provided.
	if cfg.MetaDataDir != "" {
		mdPath = cfg.MetaDataDir
	} else {
		mdPath = path.Join(cfg.DataDir, metaDataPath)
	}

	// Return path and existence of the file.
	_, err := os.Stat(mdPath)
	if !os.IsNotExist(err) {
		return mdPath, true
	}
	return mdPath, false
}

// metaDataFromFile retrieves unmarshalled p2p metadata from file.
func metaDataFromFile(path string) (metadata.Metadata, error) {
	src, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		log.WithError(err).Error("Error reading metadata from file")
		return nil, err
	}

	// Load V1 version (after altair) regardless of current fork by default.
	md := &pb.MetaDataV1{}
	err = md.UnmarshalSSZ(src)
	if err != nil {
		// If unmarshal failed, try to unmarshal for V0
		log.WithError(err).Info("Error unmarshalling V1 metadata from file, try to unmarshal for V0.")
		md0 := &pb.MetaDataV0{}
		md0Err := md0.UnmarshalSSZ(src)
		if md0Err != nil {
			log.WithError(md0Err).Error("Error unmarshalling V0 metadata from file")
			return nil, md0Err
		}
		return wrapper.WrappedMetadataV0(md0), nil
	}
	return wrapper.WrappedMetadataV1(md), nil
}

// saveMetaDataToFile writes marshalled metadata to given path.
func saveMetaDataToFile(path string, metadata metadata.Metadata) error {
	enc, err := metadata.MarshalSSZ()
	if err != nil {
		log.WithError(err).Error("Error marshalling metadata to SSZ")
		return err
	}

	if err := file.WriteFile(path, enc); err != nil {
		log.WithError(err).Error("Failed to write to disk")
		return err
	}
	return nil
}

// migrateFromProtoToSsz tries to unmarshal by proto, and migrates to ssz encoded file
// if unmarshalling is successful.
func migrateFromProtoToSsz(path string) (metadata.Metadata, error) {
	src, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		log.WithError(err).Error("Error reading metadata from file")
		return nil, err
	}

	md := &pb.MetaDataV0{}
	if err := proto.Unmarshal(src, md); err != nil {
		return nil, err
	}

	wmd := wrapper.WrappedMetadataV0(md)
	// increment sequence number
	seqNum := wmd.SequenceNumber() + 1
	newMd := &pb.MetaDataV1{
		SeqNumber: seqNum,
		Attnets:   wmd.AttnetsBitfield().Bytes(),
		Syncnets:  bitfield.NewBitvector4(),
	}
	wrappedNewMd := wrapper.WrappedMetadataV1(newMd)

	saveErr := saveMetaDataToFile(path, wrappedNewMd)
	if saveErr != nil {
		return nil, saveErr
	}
	return wrapper.WrappedMetadataV1(newMd), nil
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
