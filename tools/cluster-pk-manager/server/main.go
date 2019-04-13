package main

import (
	"flag"
	"fmt"
	"net"

	pb "github.com/prysmaticlabs/prysm/proto/cluster"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	port                = flag.Int("port", 8000, "The port to server gRPC")
	metricsPort         = flag.Int("metrics-port", 9090, "The port to serve /metrics")
	privateKey          = flag.String("private-key", "", "The private key of funder")
	rpcPath             = flag.String("rpc", "https://goerli.prylabs.net", "RPC address of a running ETH1 node")
	depositContractAddr = flag.String("deposit-contract", "", "Address of the deposit contract")
	depositAmount       = flag.Int64("deposit-amount", 0, "The amount of wei to deposit into the contract")
	dbPath              = flag.String("db-path", "", "The file path for database storage")
	disableWatchtower   = flag.Bool("disable-watchtower", false, "Disable kubernetes pod watcher. Useful for local testing")
	verbose             = flag.Bool("verbose", false, "Enable debug logging")
)

func main() {
	flag.Parse()
	if *verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	db := newDB(*dbPath)
	srv := newServer(db, *rpcPath, *depositContractAddr, *privateKey, *depositAmount)
	if !*disableWatchtower {
		wt := newWatchtower(db)
		go wt.WatchPods()
	}

	s := grpc.NewServer()
	pb.RegisterPrivateKeyServiceServer(s, srv)

	go prometheus.RunSimpleServerOrDie(fmt.Sprintf(":%d", *metricsPort))
	srv.serveAllocationsHTTPPage()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(err)
	}
	log.Infof("Listening for gRPC requests on port %d", *port)
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
