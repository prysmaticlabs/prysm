// Bootstrap / DHT query tool
//
// Usage: bazel run //tools/boostrap-query -- $BOOTNODE_ADDRESS
//
// This tool queries the bootstrap / DHT node for peers then attempts to dial
// and ping each of them.
package main

import (
	"context"
	"log"
	"os"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-host"
	dhtpb "github.com/libp2p/go-libp2p-kad-dht/pb"
	net "github.com/libp2p/go-libp2p-net"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
)

const dhtProtocol = "/prysm/0.0.0/dht"

func main() {
	if len(os.Args) == 1 {
		log.Fatal("Error: Bootnode address not provided.")
	}

	ctx := context.Background()
	h, err := libp2p.New(ctx)
	if err != nil {
		panic(err)
	}

	addr := os.Args[1]
	pi, err := p2p.MakePeer(addr)
	if err != nil {
		log.Fatalf("Error: Failed to make peer from string: %v", err)
	}
	if err := h.Connect(ctx, *pi); err != nil {
		log.Fatalf("Error: Failed to create peer from string: %v", err)
	}

	s, err := h.NewStream(ctx, pi.ID, dhtProtocol)
	if err != nil {
		log.Printf("proto = %s", dhtProtocol)
		log.Fatalf("Error: Failed to create ProtocolDHT stream: %v", err)
	}

	resp := sendMessageAndWait(s, dhtpb.NewMessage(dhtpb.Message_FIND_NODE, []byte{}, 0))

	log.Printf("Bootstrap DHT node has %d peers\n", len(resp.GetCloserPeers()))
	for i, p := range resp.GetCloserPeers() {
		log.Printf("Dialing peer %d: %+v\n", i, p.Addresses())

		if err := pingPeer(ctx, h, &p); err != nil {
			log.Printf("Error: unable to ping peer %v\n", err)
		} else {
			log.Println("OK")
		}
	}
}

func pingPeer(ctx context.Context, h host.Host, p *dhtpb.Message_Peer) error {
	pi := dhtpb.PBPeerToPeerInfo(*p)
	if err := h.Connect(ctx, pi); err != nil {
		return err
	}

	s, err := h.NewStream(ctx, pi.ID, dhtProtocol)
	if err != nil {
		return err
	}

	// Any response is OK
	_ = sendMessageAndWait(s, dhtpb.NewMessage(dhtpb.Message_PING, []byte{}, 0))
	return nil
}

func sendMessageAndWait(s net.Stream, msg *dhtpb.Message) dhtpb.Message {
	r := ggio.NewDelimitedReader(s, 2000000)
	w := ggio.NewDelimitedWriter(s)

	if err := w.WriteMsg(msg); err != nil {
		log.Fatalf("Error: failed to write message: %v", err)
	}

	var resp dhtpb.Message
	if err := r.ReadMsg(&resp); err != nil {
		log.Fatalf("Error: failed to read message: %v", err)
	}

	return resp
}
