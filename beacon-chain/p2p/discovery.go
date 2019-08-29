package p2p

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discv5"
	iaddr "github.com/ipfs/go-ipfs-addr"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
)

var TopicAdvertisementPeriod = 60 * time.Second
var TopicSearchReqInterval = 10 * time.Second
var TopicSearchDelay = 5 * time.Second

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

// Advertise the topic for a period of time, before closing off advertisement of the topic.
func (s *Service) Advertise(ctx context.Context, topic string, opts ...libp2p.Option) (time.Duration, error) {
	stopChan := make(chan struct{})
	timeChan := time.After(TopicAdvertisementPeriod)

	go func() {
		<-timeChan
		stopChan <- struct{}{}
	}()

	go s.dv5Listener.RegisterTopic(discv5.Topic(topic), stopChan)
	return TopicAdvertisementPeriod, nil
}

func (s *Service) FindPeers(ctx context.Context, topic string, opts ...libp2p.Option) (<-chan peer.AddrInfo, error) {
	ticker := time.NewTicker(TopicSearchReqInterval)
	setPeriod := make(chan time.Duration)
	found := make(chan *discv5.Node)
	lookup := make(chan bool)
	peerInfo := make(chan peer.AddrInfo)
	// set off the topic search routine, to process any search results
	go func() {
		for {
			select {
			case <-ticker.C:
				setPeriod <- TopicSearchDelay
			case node := <-found:
				multiAddr, err := convertSingleNodeToMultiAddr(node)
				if err != nil {
					log.WithError(err).Error("Could not convert to multiAddr")
					continue
				}
				addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddr)
				if err != nil {
					log.WithError(err).Error("Could not make it into peer addr info")
					continue
				}
				peerInfo <- *addrInfo
			case <-ctx.Done():
				break
			}
		}
	}()

	// set off topic search in another routine
	go s.dv5Listener.SearchTopic(discv5.Topic(topic), setPeriod, found, lookup)
	return peerInfo, nil
}

func (s *Service) FindPeer(ctx context.Context, id peer.ID) (peer.AddrInfo, error) {
	pubkey, err := id.ExtractPublicKey()
	if err != nil {
		return peer.AddrInfo{}, errors.Wrapf(err, "could not extract public key from peer id")
	}
	ecdsaKey := convertFromInterfacePubKey(pubkey)
	node := s.dv5Listener.Resolve(discv5.PubkeyID(ecdsaKey))
	if node == nil {
		return peer.AddrInfo{}, errors.New("could not find peer with that id")
	}
	multiAddr, err := convertSingleNodeToMultiAddr(node)
	if err != nil {
		return peer.AddrInfo{}, errors.Wrapf(err, "could not convert to multiaddr")
	}
	addrInfo, err := peer.AddrInfoFromP2pAddr(multiAddr)
	if err != nil {
		return peer.AddrInfo{}, errors.Wrapf(err, "Could not make it into peer addr info")
	}
	return *addrInfo, nil
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
		multiAddr, err := convertSingleNodeToMultiAddr(node)
		if err != nil {
			log.WithError(err).Error("Could not convert discv5 node to multiaddr")
			continue
		}
		multiAddrs = append(multiAddrs, multiAddr)
	}
	return multiAddrs
}

func convertSingleNodeToMultiAddr(node *discv5.Node) (ma.Multiaddr, error) {
	ip4 := node.IP.To4()
	if ip4 == nil {
		return nil, errors.New("node doesn't have an ip4 address")
	}
	pubkey, err := node.ID.Pubkey()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pubkey from node ID")
	}
	assertedKey := convertToInterfacePubkey(pubkey)
	id, err := peer.IDFromPublicKey(assertedKey)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get peer id")
	}
	multiAddrString := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ip4.String(), node.TCP, id)
	multiAddr, err := ma.NewMultiaddr(multiAddrString)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get multiaddr")
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
