package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	gCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/peerdas"
	"github.com/prysmaticlabs/prysm/v5/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/wrapper"
	ecdsaprysm "github.com/prysmaticlabs/prysm/v5/crypto/ecdsa"
	"github.com/prysmaticlabs/prysm/v5/io/file"
	pb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/metadata"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	keyPath                  = "network-keys"
	custodyColumnSubnetsPath = "custodyColumnsSubnets.json"
	metaDataPath             = "metaData"

	dialTimeout = 1 * time.Second
)

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

// randomPrivKeyWithSubnets generates a random private key which, when derived into a node ID, matches expectedSubnets.
// This is done by brute forcing the generation of a private key until it matches the desired subnets.
// TODO: Run multiple goroutines to speed up the process.
func randomPrivKeyWithSubnets(expectedSubnets map[uint64]bool) (crypto.PrivKey, uint64, time.Duration, error) {
	// Get the current time.
	start := time.Now()

mainLoop:
	for i := uint64(1); ; /* No exit condition */ i++ {
		// Get the subnets count.
		expectedSubnetsCount := len(expectedSubnets)

		// Generate a random keys pair
		privKey, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			return nil, 0, time.Duration(0), errors.Wrap(err, "generate SECP256K1 key")
		}

		ecdsaPrivKey, err := ecdsaprysm.ConvertFromInterfacePrivKey(privKey)
		if err != nil {
			return nil, 0, time.Duration(0), errors.Wrap(err, "convert from interface private key")
		}

		// Compute the node ID from the public key.
		nodeID := enode.PubkeyToIDV4(&ecdsaPrivKey.PublicKey)

		// Retrieve the custody column subnets of the node.
		actualSubnets, err := peerdas.CustodyColumnSubnets(nodeID, uint64(expectedSubnetsCount))
		if err != nil {
			return nil, 0, time.Duration(0), errors.Wrap(err, "custody column subnets")
		}

		// Safe check, just in case.
		actualSubnetsCount := len(actualSubnets)
		if actualSubnetsCount != expectedSubnetsCount {
			return nil, 0, time.Duration(0), errors.Errorf("mismatch counts of custody subnets. Actual %d - Required %d", actualSubnetsCount, expectedSubnetsCount)
		}

		// Check if the expected subnets are the same as the actual subnets.
		for _, subnet := range actualSubnets {
			if !expectedSubnets[subnet] {
				// At least one subnet does not match, so we need to generate a new key.
				continue mainLoop
			}
		}

		// It's a match, return the private key.
		return privKey, i, time.Since(start), nil
	}
}

// privateKeyWithConstraint reads the subnets from a file and generates a private key that matches the subnets.
func privateKeyWithConstraint(subnetsPath string) (crypto.PrivKey, error) {
	// Read the subnets from the file.
	data, err := file.ReadFileAsBytes(subnetsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", subnetsPath)
	}

	var storedSubnets []uint64
	if err := json.Unmarshal(data, &storedSubnets); err != nil {
		return nil, errors.Wrapf(err, "unmarshal subnets %s", subnetsPath)
	}

	storedSubnetsCount := uint64(len(storedSubnets))

	// Retrieve the subnets to custody.
	custodySubnetsCount := params.BeaconConfig().CustodyRequirement
	if flags.Get().SubscribeToAllSubnets {
		custodySubnetsCount = params.BeaconConfig().DataColumnSidecarSubnetCount
	}

	// Check our subnets count is not greater than the subnet count in the file.
	// Such a case is possible if the number of subnets increased after the file was created.
	// This is possible only within a new release. If this happens, we should implement a modification
	// of the file. At the moment, we raise an error.
	if custodySubnetsCount > storedSubnetsCount {
		return nil, errors.Errorf(
			"subnets count in the file %s (%d) is less than the current subnets count (%d)",
			subnetsPath,
			storedSubnetsCount,
			custodySubnetsCount,
		)
	}

	subnetsMap := make(map[uint64]bool, custodySubnetsCount)
	custodySubnetsMap := make(map[uint64]bool, len(storedSubnets))

	for i, subnet := range storedSubnets {
		subnetsMap[subnet] = true
		if uint64(i) < custodySubnetsCount {
			custodySubnetsMap[subnet] = true
		}
	}

	if len(subnetsMap) != len(storedSubnets) {
		return nil, errors.Errorf("duplicated subnets found in the file %s", subnetsPath)
	}

	// Generate a private key that matches the subnets.
	privKey, iterations, duration, err := randomPrivKeyWithSubnets(custodySubnetsMap)
	log.WithFields(logrus.Fields{
		"iterations": iterations,
		"duration":   duration,
	}).Info("Generated P2P private key")

	return privKey, err
}

// privateKeyWithoutConstraint generates a private key, computes the subnets and stores them in a file.
func privateKeyWithoutConstraint(subnetsPath string) (crypto.PrivKey, error) {
	// Get the total number of subnets.
	subnetCount := params.BeaconConfig().DataColumnSidecarSubnetCount

	// Generate the private key.
	privKey, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "generate SECP256K1 key")
	}

	convertedKey, err := ecdsaprysm.ConvertFromInterfacePrivKey(privKey)
	if err != nil {
		return nil, errors.Wrap(err, "convert from interface private key")
	}

	// Compute the node ID from the public key.
	nodeID := enode.PubkeyToIDV4(&convertedKey.PublicKey)

	// Retrieve the custody column subnets of the node.
	subnets, err := peerdas.CustodyColumnSubnets(nodeID, subnetCount)
	if err != nil {
		return nil, errors.Wrap(err, "custody column subnets")
	}

	// Store the subnets in a file.
	data, err := json.Marshal(subnets)
	if err != nil {
		return nil, errors.Wrap(err, "marshal subnets")
	}

	if err := file.WriteFile(subnetsPath, data); err != nil {
		return nil, errors.Wrap(err, "write file")
	}

	return privKey, nil
}

// storePrivateKey stores a private key to a file.
func storePrivateKey(privKey crypto.PrivKey, destFilePath string) error {
	// Get the raw bytes of the private key.
	rawbytes, err := privKey.Raw()
	if err != nil {
		return errors.Wrap(err, "raw")
	}

	// Encode the raw bytes to hex.
	dst := make([]byte, hex.EncodedLen(len(rawbytes)))
	hex.Encode(dst, rawbytes)

	if err := file.WriteFile(destFilePath, dst); err != nil {
		return errors.Wrapf(err, "write file: %s", destFilePath)
	}

	return err
}

// randomPrivKey generates a random private key.
func randomPrivKey(datadir string) (crypto.PrivKey, error) {
	if features.Get().EnablePeerDAS {
		// Check if the file containing the custody column subnets exists.
		subnetsPath := path.Join(datadir, custodyColumnSubnetsPath)
		exists, err := file.Exists(subnetsPath, file.Regular)
		if err != nil {
			return nil, errors.Wrap(err, "exists")
		}

		// If the file does not exist, generate a new private key, compute the subnets and store them.
		if !exists {
			priv, err := privateKeyWithoutConstraint(subnetsPath)
			if err != nil {
				return nil, errors.Wrap(err, "generate private without constraint")
			}

			return priv, nil
		}

		// If the file exists, read the subnets and generate a new private key.
		priv, err := privateKeyWithConstraint(subnetsPath)
		if err != nil {
			return nil, errors.Wrap(err, "generate private key with constraint for PeerDAS")
		}

		return priv, nil
	}

	privKey, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "generate SECP256K1 key")
	}

	return privKey, err
}

// privKey determines a private key for p2p networking from the p2p service's
// configuration struct. If no key is found, it generates a new one.
func privKey(cfg *Config) (*ecdsa.PrivateKey, error) {
	defaultKeyPath := path.Join(cfg.DataDir, keyPath)
	privateKeyPath := cfg.PrivateKey

	// PrivateKey CLI flag takes highest precedence.
	if privateKeyPath != "" {
		return privKeyFromFile(cfg.PrivateKey)
	}

	// Default keys have the next highest precedence, if they exist.
	defaultKeysExist, err := file.Exists(defaultKeyPath, file.Regular)
	if err != nil {
		return nil, errors.Wrap(err, "exists")
	}

	if defaultKeysExist {
		log.WithField("filePath", defaultKeyPath).Info("Reading static P2P private key from a file. To generate a new random private key at every start, please remove this file.")
		return privKeyFromFile(defaultKeyPath)
	}

	// Generate a new (possibly contrained) random private key.
	priv, err := randomPrivKey(cfg.DataDir)
	if err != nil {
		return nil, errors.Wrap(err, "random private key")
	}

	// If the StaticPeerID flag is not set, return the private key.
	if !cfg.StaticPeerID {
		return ecdsaprysm.ConvertFromInterfacePrivKey(priv)
	}

	// Save the generated key as the default key, so that it will be used by
	// default on the next node start.
	log.WithField("file", defaultKeyPath).Info("Wrote network key to")
	if err := storePrivateKey(priv, defaultKeyPath); err != nil {
		return nil, errors.Wrap(err, "store private key")
	}

	// Read the key from the defaultKeyPath file just written
	// for the strongest guarantee that the next start will be the same as this one.
	privKey, err := privKeyFromFile(defaultKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, "private key from file")
	}

	return privKey, nil
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
	src, err := os.ReadFile(metaDataPath) // #nosec G304
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

func ConvertPeerIDToNodeID(pid peer.ID) (enode.ID, error) {
	// Retrieve the public key object of the peer under "crypto" form.
	pubkeyObjCrypto, err := pid.ExtractPublicKey()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "extract public key")
	}
	// Extract the bytes representation of the public key.
	compressedPubKeyBytes, err := pubkeyObjCrypto.Raw()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "public key raw")
	}
	// Retrieve the public key object of the peer under "SECP256K1" form.
	pubKeyObjSecp256k1, err := btcec.ParsePubKey(compressedPubKeyBytes)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "parse public key")
	}
	newPubkey := &ecdsa.PublicKey{Curve: gCrypto.S256(), X: pubKeyObjSecp256k1.X(), Y: pubKeyObjSecp256k1.Y()}
	return enode.PubkeyToIDV4(newPubkey), nil
}
