package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prysmaticlabs/prysm/shared/iputils"
	_ "github.com/prysmaticlabs/prysm/shared/maxprocs"
)

func TestBootnode_OK(t *testing.T) {
	ipAddr, err := iputils.ExternalIPv4()
	if err != nil {
		t.Fatal(err)
	}
	privKey := extractPrivateKey()
	cfg := discover.Config{
		PrivateKey: privKey,
	}
	listener := createListener(ipAddr, 4000, cfg)
	defer listener.Close()

	cfg.PrivateKey = extractPrivateKey()
	bootNode, err := enode.Parse(enode.ValidSchemes, listener.Self().String())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Bootnodes = []*enode.Node{bootNode}
	listener2 := createListener(ipAddr, 4001, cfg)
	defer listener2.Close()

	// test that both the nodes have the other peer stored in their local table.
	listenerNode := listener.Self()
	listenerNode2 := listener2.Self()

	time.Sleep(1 * time.Second)

	nodes := listener.Lookup(listenerNode2.ID())
	if len(nodes) == 0 {
		t.Fatalf("Length of nodes stored in table is not expected. Wanted to be more than %d but got %d", 0, len(nodes))

	}
	if nodes[0].ID() != listenerNode2.ID() {
		t.Errorf("Wanted node ID of %s but got %s", listenerNode2.ID(), nodes[1].ID())
	}

	nodes = listener2.Lookup(listenerNode.ID())
	if len(nodes) == 0 {
		t.Errorf("Length of nodes stored in table is not expected. Wanted to be more than %d but got %d", 0, len(nodes))

	}
	if nodes[0].ID() != listenerNode.ID() {
		t.Errorf("Wanted node ID of %s but got %s", listenerNode.ID(), nodes[1].ID())
	}
}

func TestPrivateKey_ParsesCorrectly(t *testing.T) {
	privKey, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pk, err := privKey.Raw()
	if err != nil {
		t.Fatal(err)
	}
	*privateKey = fmt.Sprintf("%x", pk)

	extractedKey := extractPrivateKey()

	rawKey := (*ecdsa.PrivateKey)((*btcec.PrivateKey)(privKey.(*crypto.Secp256k1PrivateKey)))

	r, s, err := ecdsa.Sign(rand.Reader, extractedKey, []byte{'t', 'e', 's', 't'})
	if err != nil {
		t.Fatal(err)
	}

	isVerified := ecdsa.Verify(&rawKey.PublicKey, []byte{'t', 'e', 's', 't'}, r, s)
	if !isVerified {
		t.Error("Unmarshalled key is not the same as the key that was given to the function")
	}
	*privateKey = ""
}
