package rpc

import (
	"context"
	"strings"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/validator/client"
)

func (s *Server) registerBeaconClient() error {
	streamInterceptor := grpc.WithStreamInterceptor(middleware.ChainStreamClient(
		grpc_opentracing.StreamClientInterceptor(),
		grpc_prometheus.StreamClientInterceptor,
		grpc_retry.StreamClientInterceptor(),
	))
	dialOpts := client.ConstructDialOptions(
		s.clientMaxCallRecvMsgSize,
		s.clientWithCert,
		s.clientGrpcRetries,
		s.clientGrpcRetryDelay,
		streamInterceptor,
	)
	if dialOpts == nil {
		return errors.New("no dial options for beacon chain gRPC client")
	}
	for _, hdr := range s.clientGrpcHeaders {
		if hdr != "" {
			ss := strings.Split(hdr, "=")
			if len(ss) < 2 {
				log.Warnf("Incorrect gRPC header flag format. Skipping %v", ss[0])
				continue
			}
			s.ctx = metadata.AppendToOutgoingContext(s.ctx, ss[0], strings.Join(ss[1:], "="))
		}
	}
	conn, err := grpc.DialContext(s.ctx, s.beaconClientEndpoint, dialOpts...)
	if err != nil {
		return errors.Wrapf(err, "could not dial endpoint: %s", s.beaconClientEndpoint)
	}
	if s.clientWithCert != "" {
		log.Info("Established secure gRPC connection")
	}
	s.beaconChainClient = ethpb.NewBeaconChainClient(conn)
	s.beaconNodeClient = ethpb.NewNodeClient(conn)
	return nil
}

func (s *Server) GetBeaconStatus(ctx context.Context, _ *ptypes.Empty) (*pb.BeaconStatusResponse, error) {
	genesis, err := s.genesisFetcher.GenesisInfo(ctx)
	if err != nil {
		return nil, err
	}
	genesisTime := uint64(time.Unix(genesis.GenesisTime.Seconds, 0).Unix())
	address := genesis.DepositContractAddress
	return &pb.BeaconStatusResponse{
		BeaconNodeEndpoint:     s.beaconClientEndpoint,
		Connected:              false,
		Syncing:                false,
		GenesisTime:            genesisTime,
		DepositContractAddress: address,
		ChainHead:              nil,
	}, nil
}

func (s *Server) GetValidatorParticipation(
	ctx context.Context, req *ethpb.GetValidatorParticipationRequest,
) (*ethpb.ValidatorParticipationResponse, error) {
	return s.beaconChainClient.GetValidatorParticipation(ctx, req)
}

func (s *Server) GetValidatorPerformance(
	ctx context.Context, req *ethpb.ValidatorPerformanceRequest,
) (*ethpb.ValidatorPerformanceResponse, error) {
	return s.beaconChainClient.GetValidatorPerformance(ctx, req)
}

func (s *Server) GetValidatorBalances(
	ctx context.Context, req *ethpb.ListValidatorBalancesRequest,
) (*ethpb.ValidatorBalances, error) {
	return s.beaconChainClient.ListValidatorBalances(ctx, req)
}

func (s *Server) GetValidatorQueue(
	ctx context.Context, req *ptypes.Empty,
) (*ethpb.ValidatorQueue, error) {
	return s.beaconChainClient.GetValidatorQueue(ctx, req)
}

func (s *Server) GetPeers(
	ctx context.Context, req *ptypes.Empty,
) (*ethpb.Peers, error) {
	return s.beaconNodeClient.ListPeers(ctx, req)
}
