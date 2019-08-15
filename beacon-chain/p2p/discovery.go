package p2p

import (
	"crypto/ecdsa"
	"net"

	"github.com/ethereum/go-ethereum/p2p/discv5"

	_ "go.uber.org/automaxprocs"
)

func startDiscoveryV5(addr string, privKey *ecdsa.PrivateKey, cfg *Config) (*discv5.Network, error) {
	listener := createListener(addr, int(cfg.UDPPort), privKey)
	nodeID, err := discv5.HexID(cfg.BootstrapNodeAddr)
	if err != nil {
		return nil, err
	}

	bootNode := listener.Resolve(nodeID)
	if err := listener.SetFallbackNodes([]*discv5.Node{bootNode}); err != nil {
		return nil, err
	}
	node := listener.Self()
	log.Infof("Started Discovery: %s", node.ID)

	return listener, nil
}

func createListener(ipAddr string, port int, privKey *ecdsa.PrivateKey) *discv5.Network {
	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP(ipAddr),
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
