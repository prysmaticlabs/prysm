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
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	_ "go.uber.org/automaxprocs"
)

var (
	debug      = flag.Bool("debug", false, "Enable debug logging")
	privateKey = flag.String("private", "", "Private key to use for peer ID")
	port       = flag.Int("port", 4000, "Port to listen for connections")
	externalIP = flag.String("external-ip", "0.0.0.0", "External IP for the bootnode")

	log = logrus.WithField("prefix", "bootnode")
)

func main() {
	flag.Parse()

	fmt.Printf("Starting bootnode. Version: %s\n", version.GetVersion())

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	privKey := extractPrivateKey()
	listener := createListener(*externalIP, *port, privKey)

	node := listener.Self()
	log.Infof("Running bootnode, url: %s", node.String())

	select {}
}

func createListener(ipAddr string, port int, privKey *ecdsa.PrivateKey) *discv5.Network {
	ip := net.ParseIP(ipAddr)
	if ip.To4() == nil {
		log.Fatalf("IPV4 address not provided instead %s was provided", ipAddr)
	}
	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP(ipAddr),
		Port: port,
	}
	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	network, err := discv5.ListenUDP(privKey, conn, "", nil)
	if err != nil {
		log.Fatal(err)
	}
	return network
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
