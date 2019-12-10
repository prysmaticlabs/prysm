package p2p

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	iaddr "github.com/ipfs/go-ipfs-addr"
	core "github.com/libp2p/go-libp2p-core"
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
	if cfg.HostAddress != "" {
		hostIP := net.ParseIP(cfg.HostAddress)
		if hostIP.To4() == nil {
			log.Errorf("Invalid host address given: %s", hostIP.String())
		} else {
			localNode.SetFallbackIP(hostIP)
		}
	}
	dv5Cfg := discover.Config{
		PrivateKey: privKey,
	}
	dv5Cfg.Bootnodes = []*enode.Node{}
	for _, addr := range cfg.Discv5BootStrapAddr {
		bootNode, err := enode.Parse(enode.ValidSchemes, addr)
		if err != nil {
			log.Fatal(err)
		}
		dv5Cfg.Bootnodes = append(dv5Cfg.Bootnodes, bootNode)
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
	localNode.SetFallbackIP(ipAddr)
	localNode.SetFallbackUDP(udpPort)

	return localNode, nil
}

func startDiscoveryV5(addr net.IP, privKey *ecdsa.PrivateKey, cfg *Config) (*discover.UDPv5, error) {
	listener := createListener(addr, privKey, cfg)
	node := listener.Self()
	log.WithField("nodeID", node.ID()).Info("Started discovery v5")
	return listener, nil
}

// startDHTDiscovery supports discovery via DHT.
func startDHTDiscovery(host core.Host, bootstrapAddr string) error {
	multiAddr, err := multiAddrFromString(bootstrapAddr)
	if err != nil {
		return err
	}
	peerInfo, err := peer.AddrInfoFromP2pAddr(multiAddr)
	if err != nil {
		return err
	}
	err = host.Connect(context.Background(), *peerInfo)
	return err
}

func parseBootStrapAddrs(addrs []string) (discv5Nodes []string, kadDHTNodes []string) {
	for _, addr := range addrs {
		if addr == "" {
			// Ignore empty entries
			continue
		}
		_, err := enode.Parse(enode.ValidSchemes, addr)
		if err == nil {
			discv5Nodes = append(discv5Nodes, addr)
			continue
		}
		_, err = multiAddrFromString(addr)
		if err == nil {
			kadDHTNodes = append(kadDHTNodes, addr)
			continue
		}
		log.Errorf("Invalid bootstrap address of %s provided", addr)
	}
	if len(discv5Nodes) == 0 && len(kadDHTNodes) == 0 {
		log.Warn("No bootstrap addresses supplied")
	}
	return discv5Nodes, kadDHTNodes
}

func convertToMultiAddr(nodes []*enode.Node) []ma.Multiaddr {
	var multiAddrs []ma.Multiaddr
	for _, node := range nodes {
		// ignore nodes with no ip address stored
		if node.IP() == nil {
			continue
		}
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
		return nil, errors.Errorf("node doesn't have an ip4 address, it's stated IP is %s", node.IP().String())
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
