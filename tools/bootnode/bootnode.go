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
	"crypto/x509"
	"flag"
	"fmt"
	"net"

	curve "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discv5"

	logging "github.com/ipfs/go-log"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/prysmaticlabs/prysm/shared/version"
	_ "go.uber.org/automaxprocs"
)

var (
	debug      = flag.Bool("debug", false, "Enable debug logging")
	privateKey = flag.String("private", "", "Private key to use for peer ID")
	port       = flag.Int("port", 4000, "Port to listen for connections")

	log = logging.Logger("prysm-bootnode")
)

// ECDSACurve is the default ecdsa curve used(secpk2561)
var ECDSACurve = curve.S256()

func main() {
	flag.Parse()

	fmt.Printf("Starting bootnode. Version: %s\n", version.GetVersion())

	if *debug {
		logging.SetDebugLogging()
	}

	defaultIP := "0.0.0.0"

	privKey := extractPrivateKey()
	listener := createListener(defaultIP, *port, privKey)

	node := listener.Self()
	fmt.Printf("Running bootnode: /ip4/%s/udp/%d/discv5/%s\n", node.IP.String(), node.UDP, node.ID.String())

	select {}
}

func createListener(ipAddr string, port int, privKey *ecdsa.PrivateKey) *discv5.Network {
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
	var err error
	if *privateKey != "" {
		b, err := crypto.ConfigDecodeKey(*privateKey)
		if err != nil {
			panic(err)
		}
		privKey, err = x509.ParseECPrivateKey(b)
		if err != nil {
			panic(err)
		}

	} else {
		privKey, err = ecdsa.GenerateKey(ECDSACurve, rand.Reader)
		if err != nil {
			panic(err)
		}
		log.Warning("No private key was provided. Using default/random private key")
	}
	return privKey
}
