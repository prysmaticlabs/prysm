/**
 * Bootnode
 *
 * A node which implements the DiscoveryV5 protocol for peer
 * discovery. The purpose of this service is to provide a starting point for
 * newly connected services to find other peers outside of their network.
 *
 * Usage: Run bootnode --help for flag options.
 */
package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"flag"
	"fmt"
	"net"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	_ "go.uber.org/automaxprocs"
)

var (
	debug      = flag.Bool("debug", false, "Enable debug logging")
	privateKey = flag.String("private", "", "Private key to use for peer ID")
	port       = flag.Int("port", 4000, "Port to listen for connections")
	externalIP = flag.String("external-ip", "127.0.0.1", "External IP for the bootnode")

	log = logrus.WithField("prefix", "bootnode")
)

func main() {
	flag.Parse()

	fmt.Printf("Starting bootnode. Version: %s\n", version.GetVersion())

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	cfg := discover.Config{
		PrivateKey: extractPrivateKey(),
	}
	listener := createListener(*externalIP, *port, cfg)

	node := listener.Self()
	log.Infof("Running bootnode: %s", node.String())

	select {}
}

func createListener(ipAddr string, port int, cfg discover.Config) *discover.UDPv5 {
	ip := net.ParseIP(ipAddr)
	if ip.To4() == nil {
		log.Fatalf("IPV4 address not provided instead %s was provided", ipAddr)
	}
	udpAddr := &net.UDPAddr{
		IP:   ip,
		Port: port,
	}
	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	localNode, err := createLocalNode(cfg.PrivateKey, ip, port)
	if err != nil {
		log.Fatal(err)
	}

	network, err := discover.ListenV5(conn, localNode, cfg)
	if err != nil {
		log.Fatal(err)
	}
	return network
}

func createLocalNode(privKey *ecdsa.PrivateKey, ipAddr net.IP, port int) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not open node's peer database")
	}

	localNode := enode.NewLocalNode(db, privKey)
	ipEntry := enr.IP(ipAddr)
	udpEntry := enr.UDP(port)
	localNode.Set(ipEntry)
	localNode.Set(udpEntry)

	return localNode, nil
}

func extractPrivateKey() *ecdsa.PrivateKey {
	var privKey *ecdsa.PrivateKey
	if *privateKey != "" {
		b, err := crypto.ConfigDecodeKey(*privateKey)
		if err != nil {
			panic(err)
		}
		unmarshalledKey, err := crypto.UnmarshalPrivateKey(b)
		if err != nil {
			panic(err)
		}
		privKey = (*ecdsa.PrivateKey)((*btcec.PrivateKey)(unmarshalledKey.(*crypto.Secp256k1PrivateKey)))

	} else {
		privInterfaceKey, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			panic(err)
		}
		privKey = (*ecdsa.PrivateKey)((*btcec.PrivateKey)(privInterfaceKey.(*crypto.Secp256k1PrivateKey)))
		log.Warning("No private key was provided. Using default/random private key")

		b, err := privInterfaceKey.Bytes()
		if err != nil {
			panic(err)
		}
		log.Debugf("Private key %s", crypto.ConfigEncodeKey(b))
	}

	return privKey
}
