package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	iaddr "github.com/ipfs/go-ipfs-addr"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

// Listener defines the discovery V5 network interface that is used
// to communicate with other peers.
type Listener interface {
	Self() *enode.Node
	Close()
	Lookup(enode.ID) []*enode.Node
	ReadRandomNodes([]*enode.Node) int
	Resolve(*enode.Node) *enode.Node
	LookupRandom() []*enode.Node
	Ping(*enode.Node) error
	RequestENR(*enode.Node) (*enode.Node, error)
}

func createListener(ipAddr net.IP, privKey *ecdsa.PrivateKey, cfg *Config) *discover.UDPv5 {
	udpAddr := &net.UDPAddr{
		IP:   ipAddr,
		Port: int(cfg.UDPPort),
	}
	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	localNode, err := createLocalNode(privKey, ipAddr, int(cfg.UDPPort), int(cfg.TCPPort))
	if err != nil {
		log.Fatal(err)
	}
	dv5Cfg := discover.Config{
		PrivateKey: privKey,
	}
	if cfg.BootstrapNodeAddr != "" {
		bootNode, err := enode.Parse(enode.ValidSchemes, cfg.BootstrapNodeAddr)
		if err != nil {
			log.Fatal(err)
		}
		dv5Cfg.Bootnodes = []*enode.Node{bootNode}
	}

	network, err := discover.ListenV5(conn, localNode, dv5Cfg)
	if err != nil {
		log.Fatal(err)
	}
	return network
}

func createLocalNode(privKey *ecdsa.PrivateKey, ipAddr net.IP, udpPort int, tcpPort int) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, errors.Wrap(err, "could not open node's peer database")
	}
	localNode := enode.NewLocalNode(db, privKey)
	ipEntry := enr.IP(ipAddr)
	udpEntry := enr.UDP(udpPort)
	tcpEntry := enr.TCP(tcpPort)
	localNode.Set(ipEntry)
	localNode.Set(udpEntry)
	localNode.Set(tcpEntry)

	return localNode, nil
}

func startDiscoveryV5(addr net.IP, privKey *ecdsa.PrivateKey, cfg *Config) (*discover.UDPv5, error) {
	listener := createListener(addr, privKey, cfg)
	node := listener.Self()
	log.Infof("Started Discovery: %s", node.ID())
	return listener, nil
}

func convertToMultiAddr(nodes []*enode.Node) []ma.Multiaddr {
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

func convertToSingleMultiAddr(node *enode.Node) (ma.Multiaddr, error) {
	ip4 := node.IP().To4()
	if ip4 == nil {
		return nil, errors.New("node doesn't have an ip4 address")
	}
	pubkey := node.Pubkey()
	assertedKey := convertToInterfacePubkey(pubkey)
	id, err := peer.IDFromPublicKey(assertedKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not get peer id")
	}
	multiAddrString := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ip4.String(), node.TCP(), id)
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
