package p2p

import (
	"crypto/ecdsa"
	"net"

	"github.com/ethereum/go-ethereum/p2p/discv5"

	_ "go.uber.org/automaxprocs"
)

func startDiscoveryV5(addr string, port int, privKey *ecdsa.PrivateKey, bootStrapAddr string) error {
	listener := createListener(addr, port, privKey)
	nodeID, err := discv5.HexID(bootStrapAddr)
	if err != nil {
		return err
	}

	bootNode := listener.Resolve(nodeID)
	if err := listener.SetFallbackNodes([]*discv5.Node{bootNode}); err != nil {
		return err
	}

	node := listener.Self()
	log.Infof("Started Discovery: %s", node.ID)
	return nil
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
