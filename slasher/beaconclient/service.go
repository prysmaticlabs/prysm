package beaconclient

import (
	"context"
	"errors"

	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
)

var log = logrus.WithField("prefix", "beaconclient")

// Service --
type Service struct {
	context  context.Context
	cancel   context.CancelFunc
	cert     string
	conn     *grpc.ClientConn
	provider string
	client   eth.BeaconChainClient
}

// Stop the beacon client service.
func (bs *Service) Stop() error {
	bs.cancel()
	log.Info("Stopping service")
	if bs.conn != nil {
		return bs.conn.Close()
	}
	return nil
}

// Status --
func (bs *Service) Status() error {
	if bs.conn == nil {
		return errors.New("no connection to beacon RPC")
	}
	return nil
}

func (bs *Service) Start() {
	var dialOpt grpc.DialOption

	if bs.cert != "" {
		creds, err := credentials.NewClientTLSFromFile(bs.cert, "")
		if err != nil {
			log.Errorf("Could not get valid credentials: %v", err)
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn("You are using an insecure gRPC connection to beacon chain! Please provide a certificate and key to use a secure connection.")
	}
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
	conn, err := grpc.DialContext(s.context, bs.provider, beaconOpts...)
	if err != nil {
		return fmt.Errorf("could not dial endpoint: %s, %v", bs.provider, err)
	}
	log.Info("Successfully started gRPC connection")
	s.beaconConn = conn
	s.beaconClient = eth.NewBeaconChainClient(bs.conn)
}
