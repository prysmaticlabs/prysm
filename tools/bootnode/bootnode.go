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
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"flag"
	"fmt"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	protocol "github.com/libp2p/go-libp2p-protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/shared/version"
	_ "go.uber.org/automaxprocs"
)

var (
	debug      = flag.Bool("debug", false, "Enable debug logging")
	privateKey = flag.String("private", "", "Private key to use for peer ID")
	port       = flag.Int("port", 4000, "Port to listen for connections")

	log = logging.Logger("prysm-bootnode")
)

const dhtProtocol = "/prysm/0.0.0/dht"

// ECDSACurve is the default ecdsa curve used
var ECDSACurve = elliptic.P256()

func main() {
	flag.Parse()

	fmt.Printf("Starting bootnode. Version: %s\n", version.GetVersion())

	if *debug {
		logging.SetDebugLogging()
	}
	nd := discv5.NewNode()
	discv5.ListenUDP()

	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))
	if err != nil {
		log.Fatalf("Failed to construct new multiaddress. %v", err)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(listen),
	}

	privKey := addPrivateKeyOpt(opts)
	discv5.ListenUDP(privKey, conn)

	ctx := context.Background()

	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		log.Fatalf("Failed to create new host. %v", err)
	}

	dopts := []dhtopts.Option{
		dhtopts.Datastore(dsync.MutexWrap(ds.NewMapDatastore())),
		dhtopts.Protocols(
			protocol.ID(dhtProtocol),
		),
	}

	dht, err := kaddht.New(ctx, host, dopts...)
	if err != nil {
		log.Fatalf("Failed to create new dht: %v", err)
	}
	if err := dht.Bootstrap(context.Background()); err != nil {
		log.Fatalf("Failed to bootstrap DHT. %v", err)
	}

	fmt.Printf("Running bootnode: /ip4/0.0.0.0/tcp/%d/p2p/%s\n", *port, host.ID().Pretty())

	select {}
}

func addPrivateKeyOpt(opts []libp2p.Option) *ecdsa.PrivateKey {
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
