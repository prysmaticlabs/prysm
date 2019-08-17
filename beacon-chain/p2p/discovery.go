package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	ma "github.com/multiformats/go-multiaddr"
	_ "go.uber.org/automaxprocs"
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

var discv5codec = 0x01C1
var discv5Protocol = ma.Protocol{
	Name:       "discv5",
	Code:       discv5codec,
	VCode:      ma.CodeToVarint(discv5codec),
	Size:       64,
	Transcoder: ma.TranscoderUnix, // doesn't do any transcoding since the argument to the protocol is a pubkey
	// TODO(#3147): Add Transcoder to validate pubkey argument.
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
	log.Infof("Started Discovery: %s", node.ID)
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
		if node.TCP < 1024 {
			log.Errorf("Invalid port, the tcp port of the node is a reserved port: %d", node.TCP)
		}
		multiAddrString := fmt.Sprintf("/ip4/%s/tcp/%d/discv5/%s", ip4.String(), node.TCP, node.ID.String())
		multiAddr, err := ma.NewMultiaddr(multiAddrString)
		if err != nil {
			log.Errorf("Could not get multiaddr:%v", err)
			continue
		}
		multiAddrs = append(multiAddrs, multiAddr)
	}
	return multiAddrs
}

func addDiscv5protocol() error {
	for _, p := range ma.Protocols {
		if p.Name == discv5Protocol.Name {
			return nil
		}
	}
	return ma.AddProtocol(discv5Protocol)
}
