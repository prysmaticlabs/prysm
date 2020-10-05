/**
 * Relay node
 *
 * A simple libp2p relay node peers to connect inbound traffic behind a NAT or
 * other network restriction.
 *
 * Usage: Run relaynode --help for flag options.
 */
package main

import (
	"context"
	"flag"
	"fmt"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multiaddr"
	_ "github.com/prysmaticlabs/prysm/shared/maxprocs"
	"github.com/prysmaticlabs/prysm/shared/version"
)

var (
	privateKey = flag.String("private", "", "Private key to use for peer ID")
	port       = flag.Int("port", 4000, "Port to listen for connections")
	debug      = flag.Bool("debug", false, "Enable debug logging")

	log = logging.Logger("prysm-relaynode")
)

func main() {
	flag.Parse()

	fmt.Printf("Starting relay node. Version: %s\n", version.GetVersion())

	if *debug {
		logging.SetDebugLogging()
	}

	ctx := context.Background()
	log.Start(ctx, "main")
	defer log.Finish(ctx)

	srcMAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))
	if err != nil {
		log.Fatalf("Unable to construct multiaddr %v", err)
	}

	opts := []libp2p.Option{
		libp2p.EnableRelay(circuit.OptHop),
		libp2p.ListenAddrs(srcMAddr),
	}

	if *privateKey != "" {
		b, err := crypto.ConfigDecodeKey(*privateKey)
		if err != nil {
			log.Fatalf("Failed to decode private key %v", err)
		}
		pk, err := crypto.UnmarshalPrivateKey(b)
		if err != nil {
			log.Fatalf("Failed to unmarshal private key %v", err)
		}
		opts = append(opts, libp2p.Identity(pk))
	} else {
		log.Warning("No private key provided. Using random key.")
	}

	h, err := libp2p.New(
		ctx,
		opts...,
	)
	if err != nil {
		log.Fatalf("Failed to create host %v", err)
	}

	fmt.Printf("Relay available: /ip4/0.0.0.0/tcp/%v/p2p/%s\n", *port, h.ID().Pretty())

	select {}
}
