/**
 * Bootnode
 *
 * A node which implements the DiscoveryV5 protocol for peer
 * discovery. The purpose of this service is to provide a starting point for
 * newly connected services to find other peers outside of their network.
 *
 * Usage: Run bootnode --help for flag options.
 */
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/btcsuite/btcd/btcec"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/sirupsen/logrus"
	_ "go.uber.org/automaxprocs"
)

var (
	debug        = flag.Bool("debug", false, "Enable debug logging")
	privateKey   = flag.String("private", "", "Private key to use for peer ID")
	discv5port   = flag.Int("discv5-port", 4000, "Port to listen for discv5 connections")
	kademliaPort = flag.Int("kad-port", 4500, "Port to listen for connections to kad DHT")
	metricsPort  = flag.Int("metrics-port", 5000, "Port to listen for connections")
	externalIP   = flag.String("external-ip", "127.0.0.1", "External IP for the bootnode")

	log = logrus.WithField("prefix", "bootnode")
)

const dhtProtocol = "/prysm/0.0.0/dht"

type handler struct {
	listener *discover.UDPv5
}

func main() {
	flag.Parse()

	fmt.Printf("Starting bootnode. Version: %s\n", version.GetVersion())

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)

		// Geth specific logging.
		glogger := gethlog.NewGlogHandler(gethlog.StreamHandler(os.Stderr, gethlog.TerminalFormat(false)))
		glogger.Verbosity(gethlog.LvlTrace)
		gethlog.Root().SetHandler(glogger)

		log.Debug("Debug logging enabled.")
	}
	privKey, interfacePrivKey := extractPrivateKey()
	cfg := discover.Config{
		PrivateKey: privKey,
	}
	listener := createListener(*externalIP, *discv5port, cfg)

	node := listener.Self()
	log.Infof("Running bootnode: %s", node.String())

	startKademliaDHT(interfacePrivKey)

	handler := &handler{
		listener: listener,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/p2p", handler.httpHandler)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *metricsPort), mux); err != nil {
		log.Fatalf("Failed to start server %v", err)
	}

	select {}
}

func startKademliaDHT(privKey crypto.PrivKey) {

	if *debug {
		logging.SetDebugLogging()
	}

	listen, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", *externalIP, *kademliaPort))
	if err != nil {
		log.Fatalf("Failed to construct new multiaddress. %v", err)
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrs(listen),
	}
	opts = append(opts, libp2p.Identity(privKey))

	ctx := context.Background()
	host, err := libp2p.New(ctx, opts...)
	if err != nil {
		log.Fatalf("Failed to create new host. %v", err)
	}

	dopts := []dhtopts.Option{
		dhtopts.Datastore(dsync.MutexWrap(ds.NewMapDatastore())),
		dhtopts.Protocols(
			dhtProtocol,
		),
	}

	dht, err := kaddht.New(ctx, host, dopts...)
	if err != nil {
		log.Fatalf("Failed to create new dht: %v", err)
	}
	if err := dht.Bootstrap(context.Background()); err != nil {
		log.Fatalf("Failed to bootstrap DHT. %v", err)
	}

	fmt.Printf("Running Kademlia DHT bootnode: /ip4/%s/tcp/%d/p2p/%s\n", *externalIP, *kademliaPort, host.ID().Pretty())
}

func createListener(ipAddr string, port int, cfg discover.Config) *discover.UDPv5 {
	ip := net.ParseIP(ipAddr)
	if ip.To4() == nil {
		log.Fatalf("IPV4 address not provided instead %s was provided", ipAddr)
	}
	udpAddr := &net.UDPAddr{
		IP:   ip,
		Port: port,
	}
	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	localNode, err := createLocalNode(cfg.PrivateKey, ip, port)
	if err != nil {
		log.Fatal(err)
	}

	network, err := discover.ListenV5(conn, localNode, cfg)
	if err != nil {
		log.Fatal(err)
	}
	return network
}

func (h *handler) httpHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	allNodes := h.listener.AllNodes()
	w.Write([]byte("Nodes stored in the table:\n"))
	for i, n := range allNodes {
		w.Write([]byte(fmt.Sprintf("Node %d\n", i)))
		w.Write([]byte(n.String() + "\n"))
		w.Write([]byte("Node ID: " + n.ID().String() + "\n"))
		w.Write([]byte("IP: " + n.IP().String() + "\n"))
		w.Write([]byte(fmt.Sprintf("UDP Port: %d", n.UDP()) + "\n"))
		w.Write([]byte(fmt.Sprintf("TCP Port: %d", n.UDP()) + "\n\n"))
	}

}

func createLocalNode(privKey *ecdsa.PrivateKey, ipAddr net.IP, port int) (*enode.LocalNode, error) {
	db, err := enode.OpenDB("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not open node's peer database")
	}

	localNode := enode.NewLocalNode(db, privKey)
	ipEntry := enr.IP(ipAddr)
	udpEntry := enr.UDP(port)
	localNode.SetFallbackIP(ipAddr)
	localNode.SetFallbackUDP(port)
	localNode.Set(ipEntry)
	localNode.Set(udpEntry)

	return localNode, nil
}

func extractPrivateKey() (*ecdsa.PrivateKey, crypto.PrivKey) {
	var privKey *ecdsa.PrivateKey
	var interfaceKey crypto.PrivKey
	if *privateKey != "" {
		dst, err := hex.DecodeString(*privateKey)
		if err != nil {
			panic(err)
		}
		unmarshalledKey, err := crypto.UnmarshalSecp256k1PrivateKey(dst)
		if err != nil {
			panic(err)
		}
		interfaceKey = unmarshalledKey
		privKey = (*ecdsa.PrivateKey)((*btcec.PrivateKey)(unmarshalledKey.(*crypto.Secp256k1PrivateKey)))

	} else {
		privInterfaceKey, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
		if err != nil {
			panic(err)
		}
		interfaceKey = privInterfaceKey
		privKey = (*ecdsa.PrivateKey)((*btcec.PrivateKey)(privInterfaceKey.(*crypto.Secp256k1PrivateKey)))
		log.Warning("No private key was provided. Using default/random private key")
		b, err := privInterfaceKey.Raw()
		if err != nil {
			panic(err)
		}
		log.Debugf("Private key %x", b)
	}

	return privKey, interfaceKey
}
