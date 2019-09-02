package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	iaddr "github.com/ipfs/go-ipfs-addr"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
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
	listener := createListener(addr, int(cfg.Port), privKey)
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
		multiAddr, err := convertToSingleMultiAddr(node)
		if err != nil {
			log.WithError(err).Error("Could not convert to multiAddr")
			continue
		}
		multiAddrs = append(multiAddrs, multiAddr)
	}
	return multiAddrs
}

func convertToSingleMultiAddr(node *discv5.Node) (ma.Multiaddr, error) {
	ip4 := node.IP.To4()
	if ip4 == nil {
		return nil, errors.New("node doesn't have an ip4 address")
	}
	pubkey, err := node.ID.Pubkey()
	if err != nil {
		return nil, errors.Wrap(err, "could not get pubkey from node ID")
	}
	assertedKey := convertToInterfacePubkey(pubkey)
	id, err := peer.IDFromPublicKey(assertedKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not get peer id")
	}
	multiAddrString := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ip4.String(), node.TCP, id)
	multiAddr, err := ma.NewMultiaddr(multiAddrString)
	if err != nil {
		return nil, errors.Wrap(err, "could not get multiaddr")
	}
	return multiAddr, nil
}

func manyMultiAddrsFromString(addrs []string) ([]ma.Multiaddr, error) {
	var allAddrs []ma.Multiaddr
	for _, stringAddr := range addrs {
		addr, err := multiAddrFromString(stringAddr)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not get multiaddr from string")
		}
		allAddrs = append(allAddrs, addr)
	}
	return allAddrs, nil
}

func multiAddrFromString(address string) (ma.Multiaddr, error) {
	addr, err := iaddr.ParseString(address)
	if err != nil {
		return nil, err
	}
	return addr.Multiaddr(), nil
}
