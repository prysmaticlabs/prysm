package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Listener defines the discovery V5 network interface that is used
// to communicate with other peers.
type Listener interface {
	Self() *discv5.Node
	Close()
	Lookup(discv5.NodeID) []*discv5.Node
	ReadRandomNodes([]*discv5.Node) int
	SetFallbackNodes([]*discv5.Node) error
	Resolve(discv5.NodeID) *discv5.Node
	RegisterTopic(discv5.Topic, <-chan struct{})
	SearchTopic(discv5.Topic, <-chan time.Duration, chan<- *discv5.Node, chan<- bool)
}

func createListener(ipAddr net.IP, port int, privKey *ecdsa.PrivateKey) *discv5.Network {
	udpAddr := &net.UDPAddr{
		IP:   ipAddr,
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

func startDiscoveryV5(addr net.IP, privKey *ecdsa.PrivateKey, cfg *Config) (*discv5.Network, error) {
	listener := createListener(addr, int(cfg.UDPPort), privKey)
	bootNode, err := discv5.ParseNode(cfg.BootstrapNodeAddr)
	if err != nil {
		return nil, err
	}
	if err := listener.SetFallbackNodes([]*discv5.Node{bootNode}); err != nil {
		return nil, err
	}
	node := listener.Self()
	log.Infof("Started Discovery: %s", node.String())
	return listener, nil
}

func convertToMultiAddr(nodes []*discv5.Node) []ma.Multiaddr {
	var multiAddrs []ma.Multiaddr
	for _, node := range nodes {
		ip4 := node.IP.To4()
		if ip4 == nil {
			log.Error("Node doesn't have an ip4 address")
			continue
		}
		pubkey, err := node.ID.Pubkey()
		if err != nil {
			log.Errorf("Could not get pubkey from node ID: %v", err)
			continue
		}
		assertedKey := convertToInterfacePubkey(pubkey)
		id, err := peer.IDFromPublicKey(assertedKey)
		if err != nil {
			log.Errorf("Could not get peer id: %v", err)
		}
		multiAddrString := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ip4.String(), node.TCP, id)
		multiAddr, err := ma.NewMultiaddr(multiAddrString)
		if err != nil {
			log.Errorf("Could not get multiaddr:%v", err)
			continue
		}
		multiAddrs = append(multiAddrs, multiAddr)
	}
	return multiAddrs
}
