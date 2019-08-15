package main

import (
	"testing"

	"github.com/ethereum/go-ethereum/p2p/discv5"
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
