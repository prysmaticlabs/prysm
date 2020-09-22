package main

import (
	"fmt"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	address := "127.0.0.1:4002"
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}
	defer conn.Close()

	client := slashpb.NewSlasherClient(conn)
	res, err := client.HighestAttestations(context.Background(), &slashpb.HighestAttestationRequest{
		Epoch:                1,
		ValidatorIds:         []uint64{40},
	})
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}

	fmt.Printf("%s", res)



	//lis, err := net.Listen("tcp", address)
	//if err != nil {
	//	log.Errorf("Could not listen to port in Start() %s: %v", address, err)
	//}
	//s.listener = lis
	//log.WithField("address", address).Info("RPC-API listening on port")
	//
	//opts := []grpc.ServerOption{
	//	grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	//	grpc.StreamInterceptor(middleware.ChainStreamServer(
	//		recovery.StreamServerInterceptor(
	//			recovery.WithRecoveryHandlerContext(traceutil.RecoveryHandlerFunc),
	//		),
	//		grpc_prometheus.StreamServerInterceptor,
	//		grpc_opentracing.StreamServerInterceptor(),
	//	)),
	//	grpc.UnaryInterceptor(middleware.ChainUnaryServer(
	//		recovery.UnaryServerInterceptor(
	//			recovery.WithRecoveryHandlerContext(traceutil.RecoveryHandlerFunc),
	//		),
	//		grpc_prometheus.UnaryServerInterceptor,
	//		grpc_opentracing.UnaryServerInterceptor(),
	//	)),
	//}
	//grpc_prometheus.EnableHandlingTimeHistogram()
	//// TODO(#791): Utilize a certificate for secure connections
	//// between beacon nodes and validator clients.
	//if s.withCert != "" && s.withKey != "" {
	//	creds, err := credentials.NewServerTLSFromFile(s.withCert, s.withKey)
	//	if err != nil {
	//		log.Errorf("Could not load TLS keys: %s", err)
	//		s.credentialError = err
	//	}
	//	opts = append(opts, grpc.Creds(creds))
	//} else {
	//	log.Warn("You are using an insecure gRPC server. If you are running your slasher and " +
	//		"validator on the same machines, you can ignore this message. If you want to know " +
	//		"how to enable secure connections, see: https://docs.prylabs.network/docs/prysm-usage/secure-grpc")
	//}
}
