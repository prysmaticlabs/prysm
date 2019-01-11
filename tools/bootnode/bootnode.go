/**
 * Bootnode
 *
 * A simple peer Kademlia distributed hash table (DHT) service for peer
 * discovery. The purpose of this service is to provide a starting point for
 * newly connected services to find other peers outside of their network.
 *
 * Usage: Run bootnode --help for flag options.
 */
package main

import (
	"context"
	"flag"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prysmaticlabs/prysm/shared/version"
)

var (
	debug      = flag.Bool("debug", false, "Enable debug logging")
	privateKey = flag.String("private", "", "Private key to use for peer ID")
	port       = flag.Int("port", 4000, "Port to listen for connections")

	log = logging.Logger("prysm-bootnode")
)

func main() {
	flag.Parse()

	fmt.Printf("Starting bootnode. Version: %s\n", version.GetVersion())

	if *debug {
		logging.SetDebugLogging()
	}

	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))
	if err != nil {
		log.Fatalf("Failed to construct new multiaddress. %v", err)
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(listen),
	}
	opts = addPrivateKeyOpt(opts)

	host, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		log.Fatalf("Failed to create new host. %v", err)
	}

	dstore := dsync.MutexWrap(ds.NewMapDatastore())
	dht := kaddht.NewDHT(context.Background(), host, dstore)
	if err := dht.Bootstrap(context.Background()); err != nil {
		log.Fatalf("Failed to bootstrap DHT. %v", err)
	}

	fmt.Printf("Running bootnode: /ip4/0.0.0.0/tcp/%d/p2p/%s\n", *port, host.ID().Pretty())

	select {}
}

func addPrivateKeyOpt(opts []libp2p.Option) []libp2p.Option {
	if *privateKey != "" {
		b, err := crypto.ConfigDecodeKey(*privateKey)
		if err != nil {
			panic(err)
		}
		pk, err := crypto.UnmarshalPrivateKey(b)
		if err != nil {
			panic(err)
		}
		opts = append(opts, libp2p.Identity(pk))
	} else {
		log.Warning("No private key was provided. Using default/random private key")
	}
	return opts
}
