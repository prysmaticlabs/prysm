package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/prysmaticlabs/prysm/shared/iputils"
	_ "go.uber.org/automaxprocs"
)

func TestBootnode_OK(t *testing.T) {
	ipAddr, err := iputils.ExternalIPv4()
	if err != nil {
		t.Fatal(err)
	}
	privKey := extractPrivateKey()
	listener := createListener(ipAddr, 4000, privKey)
	defer listener.Close()

	privKey = extractPrivateKey()
	listener2 := createListener(ipAddr, 4001, privKey)
	defer listener.Close()

	err = listener.SetFallbackNodes([]*discv5.Node{listener2.Self()})
	if err != nil {
		t.Fatal(err)
	}

	err = listener2.SetFallbackNodes([]*discv5.Node{listener.Self()})
	if err != nil {
		t.Fatal(err)
	}

	// test that both the nodes have the other peer stored in their local table.
	listenerNode := listener.Self()
	listenerNode2 := listener2.Self()

	nodes := listener.Lookup(listenerNode2.ID)
	if len(nodes) != 2 {
		t.Errorf("Length of nodes stored in table is not expected. Wanted %d but got %d", 2, len(nodes))

	}
	if nodes[0].ID != listenerNode2.ID {
		t.Errorf("Wanted node ID of %s but got %s", listenerNode2.ID, nodes[1].ID)
	}

	nodes = listener2.Lookup(listenerNode.ID)
	if len(nodes) != 2 {
		t.Errorf("Length of nodes stored in table is not expected. Wanted %d but got %d", 2, len(nodes))

	}
	if nodes[0].ID != listenerNode.ID {
		t.Errorf("Wanted node ID of %s but got %s", listenerNode.ID, nodes[1].ID)
	}
}

func TestPrivateKey_ParsesCorrectly(t *testing.T) {
	privKey, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	marshalledKey, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		t.Fatal(err)
	}
	encodedKey := crypto.ConfigEncodeKey(marshalledKey)
	*privateKey = encodedKey

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
