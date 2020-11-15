package main

import (
	"flag"
	"fmt"
	"net"

	pb "github.com/prysmaticlabs/prysm/proto/cluster"
	_ "github.com/prysmaticlabs/prysm/shared/maxprocs"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	port                = flag.Int("port", 8000, "The port to server gRPC")
	metricsPort         = flag.Int("metrics-port", 9090, "The port to serve /metrics")
	privateKey          = flag.String("private-key", "", "The private key of funder")
	rpcPath             = flag.String("rpc", "https://goerli.prylabs.net", "RPC address of a running ETH1 node")
	beaconRPCPath       = flag.String("beaconRPC", "localhost:4000", "RPC address of Beacon Node")
	depositContractAddr = flag.String("deposit-contract", "", "Address of the deposit contract")
	depositAmount       = flag.String("deposit-amount", "", "The amount of wei to deposit into the contract")
	dbPath              = flag.String("db-path", "", "The file path for database storage")
	disableWatchtower   = flag.Bool("disable-watchtower", false, "Disable kubernetes pod watcher. Useful for local testing")
	verbose             = flag.Bool("verbose", false, "Enable debug logging")
	ensureDeposited     = flag.Bool("ensure-deposited", false, "Ensure keys are deposited")
	allowNewDeposits    = flag.Bool("allow-new-deposits", true, "Allow cluster PK manager to send new deposits or generate new keys")
)

func main() {
	// Using Medalla as the default configuration.
	params.UseMedallaConfig()

	flag.Parse()
	if *verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if *ensureDeposited {
		log.Warn("--ensure-deposited: Ensuring all keys are deposited or deleting them from database!")
	}
	if !*allowNewDeposits {
		log.Warn("Disallowing new deposits")
	}

	db := newDB(*dbPath)
	srv := newServer(db, *rpcPath, *depositContractAddr, *privateKey, *depositAmount, *beaconRPCPath)
	if !*disableWatchtower {
		wt := newWatchtower(db)
		go wt.WatchPods()
	}

	kc := newkeyChecker(db, *beaconRPCPath)
	go kc.run()

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
