// This binary is a simple rest API endpoint to calculate
// the ENR value of a node given its private key,ip address and port.
package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"flag"
	"io/ioutil"
	"net"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/crypto"
	_ "github.com/prysmaticlabs/prysm/shared/maxprocs"
	log "github.com/sirupsen/logrus"
)

var (
	privateKey = flag.String("private", "", "Hex encoded Private key to use for calculation of ENR")
	udpPort    = flag.Int("udp-port", 0, "UDP Port to use for calculation of ENR")
	tcpPort    = flag.Int("tcp-port", 0, "TCP Port to use for calculation of ENR")
	ipAddr     = flag.String("ipAddress", "", "IP to use in calculation of ENR")
	outfile    = flag.String("out", "", "Filepath to write ENR")
)

func main() {
	flag.Parse()

	if len(*privateKey) == 0 {
		log.Fatal("No private key given")
	}
	dst, err := hex.DecodeString(*privateKey)
	if err != nil {
		panic(err)
	}
	unmarshalledKey, err := crypto.UnmarshalSecp256k1PrivateKey(dst)
	if err != nil {
		panic(err)
	}
	ecdsaPrivKey := (*ecdsa.PrivateKey)((*btcec.PrivateKey)(unmarshalledKey.(*crypto.Secp256k1PrivateKey)))

	if net.ParseIP(*ipAddr).To4() == nil {
		log.Fatalf("Invalid ipv4 address given: %v\n", err)
	}

	if *udpPort == 0 {
		log.Fatalf("Invalid udp port given: %v\n", err)
		return
	}

	db, err := enode.OpenDB("")
	if err != nil {
		log.Fatalf("Could not open node's peer database: %v\n", err)
		return
	}
	defer db.Close()

	localNode := enode.NewLocalNode(db, ecdsaPrivKey)
	ipEntry := enr.IP(net.ParseIP(*ipAddr))
	udpEntry := enr.UDP(*udpPort)
	localNode.Set(ipEntry)
	localNode.Set(udpEntry)
	if *tcpPort != 0 {
		tcpEntry := enr.TCP(*tcpPort)
		localNode.Set(tcpEntry)
	}
	log.Info(localNode.Node().String())

	if *outfile != "" {
		err := ioutil.WriteFile(*outfile, []byte(localNode.Node().String()), 0644)
		if err != nil {
			panic(err)
		}
		log.Infof("Wrote to %s", *outfile)
	}
}
