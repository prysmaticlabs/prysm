package main

import (
	"context"
	"flag"
	"fmt"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1_gateway"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
)

var (
	// Required fields
	datadir = flag.String("datadir", "", "Path to data directory.")
)

func init() {
	fc := featureconfig.Get()
	fc.WriteSSZStateTransitions = true
	featureconfig.Init(fc)
}

func main() {
	flag.Parse()
	fmt.Println("Starting process...")
	d, err := db.NewDB(*datadir)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	cp, _ := d.FinalizedCheckpoint(ctx)
	for e := uint64(0); e < cp.Epoch; e++ {
		d.Attestations(ctx, filters.NewFilter().SetTargetEpoch(e))

	}

	//if err != nil {
	//	panic(err)
	//}
	//if len(roots) != 1 {
	//	fmt.Printf("Expected 1 block root for slot %d, got %d roots", *state, len(roots))
	//}
	//s, err := d.State(ctx, roots[0])
	//if err != nil {
	//	panic(err)
	//}
	//
	//interop.WriteStateToDisk(s)
	fmt.Println("done")
}
func startBeaconClient() {
	var dialOpt grpc.DialOption

	dialOpt = grpc.WithInsecure()
	log.Warn("You are using an insecure gRPC connection to beacon chain! Please provide a certificate and key to use a secure connection.")

	beaconOpts := []grpc.DialOption{
		dialOpt,
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithStreamInterceptor(middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
		)),
		grpc.WithUnaryInterceptor(middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
		)),
	}
	conn, err := grpc.DialContext(s.context, s.beaconProvider, beaconOpts...)
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", s.beaconProvider, err)
		return nil, nil, err
	}
	log.Info("Successfully started gRPC connection")
	beaconClient := eth.NewBeaconChainClient(conn)
	return conn, beaconClient, nil
}
