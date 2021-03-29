/*
Package beaconclient defines a service that interacts with a beacon
node via a gRPC client to listen for streamed blocks, attestations, and to
submit proposer/attester slashings to the node in case they are detected.
*/
package beaconclient

import (
	"context"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"github.com/prysmaticlabs/prysm/slasher/cache"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Notifier defines a struct which exposes event feeds regarding beacon blocks,
// attestations, and more information received from a beacon node.
type Notifier interface {
	BlockFeed() *event.Feed
	AttestationFeed() *event.Feed
	ClientReadyFeed() *event.Feed
}

// ChainFetcher defines a struct which can retrieve
// chain information from a beacon node such as the latest chain head.
type ChainFetcher interface {
	ChainHead(ctx context.Context) (*ethpb.ChainHead, error)
}

// Service struct for the beaconclient service of the slasher.
type Service struct {
	cfg                         *Config
	ctx                         context.Context
	cancel                      context.CancelFunc
	conn                        *grpc.ClientConn
	clientFeed                  *event.Feed
	blockFeed                   *event.Feed
	attestationFeed             *event.Feed
	proposerSlashingsChan       chan *ethpb.ProposerSlashing
	attesterSlashingsChan       chan *ethpb.AttesterSlashing
	receivedAttestationsBuffer  chan *ethpb.IndexedAttestation
	collectedAttestationsBuffer chan []*ethpb.IndexedAttestation
	publicKeyCache              *cache.PublicKeyCache
	genesisValidatorRoot        []byte
	beaconDialOptions           []grpc.DialOption
}

// Config options for the beaconclient service.
type Config struct {
	BeaconProvider        string
	BeaconCert            string
	SlasherDB             db.Database
	ProposerSlashingsFeed *event.Feed
	AttesterSlashingsFeed *event.Feed
	BeaconClient          ethpb.BeaconChainClient
	NodeClient            ethpb.NodeClient
}

// NewService instantiation.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // govet fix for lost cancel. Cancel is handled in service.Stop()
	publicKeyCache, err := cache.NewPublicKeyCache(0, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create new cache")
	}

	return &Service{
		cfg:                         cfg,
		ctx:                         ctx,
		cancel:                      cancel,
		blockFeed:                   new(event.Feed),
		clientFeed:                  new(event.Feed),
		attestationFeed:             new(event.Feed),
		proposerSlashingsChan:       make(chan *ethpb.ProposerSlashing, 1),
		attesterSlashingsChan:       make(chan *ethpb.AttesterSlashing, 1),
		receivedAttestationsBuffer:  make(chan *ethpb.IndexedAttestation, 1),
		collectedAttestationsBuffer: make(chan []*ethpb.IndexedAttestation, 1),
		publicKeyCache:              publicKeyCache,
	}, nil
}

// BlockFeed returns a feed other services in slasher can subscribe to
// blocks received via the beacon node through gRPC.
func (s *Service) BlockFeed() *event.Feed {
	return s.blockFeed
}

// AttestationFeed returns a feed other services in slasher can subscribe to
// attestations received via the beacon node through gRPC.
func (s *Service) AttestationFeed() *event.Feed {
	return s.attestationFeed
}

// ClientReadyFeed returns a feed other services in slasher can subscribe to
// to indicate when the gRPC connection is ready.
func (s *Service) ClientReadyFeed() *event.Feed {
	return s.clientFeed
}

// Stop the beacon client service by closing the gRPC connection.
func (s *Service) Stop() error {
	s.cancel()
	log.Info("Stopping service")
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// Status returns an error if there exists a gRPC connection error
// in the service.
func (s *Service) Status() error {
	if s.conn == nil {
		return errors.New("no connection to beacon RPC")
	}
	return nil
}

// Start the main runtime of the beaconclient service, initializing
// a gRPC client connection with a beacon node, listening for
// streamed blocks/attestations, and submitting slashing operations
// after they are detected by other services in the slasher.
func (s *Service) Start() {
	var dialOpt grpc.DialOption
	if s.cfg.BeaconCert != "" {
		creds, err := credentials.NewClientTLSFromFile(s.cfg.BeaconCert, "")
		if err != nil {
			log.Errorf("Could not get valid credentials: %v", err)
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn(
			"You are using an insecure gRPC connection to beacon chain! Please provide a certificate and key to use a secure connection",
		)
	}
	beaconOpts := []grpc.DialOption{
		dialOpt,
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithStreamInterceptor(middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
			grpc_retry.StreamClientInterceptor(),
			grpcutils.LogStream,
		)),
		grpc.WithUnaryInterceptor(middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
			grpc_retry.UnaryClientInterceptor(),
			grpcutils.LogRequests,
		)),
	}
	conn, err := grpc.DialContext(s.ctx, s.cfg.BeaconProvider, beaconOpts...)
	if err != nil {
		log.Fatalf("Could not dial endpoint: %s, %v", s.cfg.BeaconProvider, err)
	}
	s.beaconDialOptions = beaconOpts
	log.Info("Successfully started gRPC connection")
	s.conn = conn
	s.cfg.BeaconClient = ethpb.NewBeaconChainClient(s.conn)
	s.cfg.NodeClient = ethpb.NewNodeClient(s.conn)

	// We poll for the sync status of the beacon node until it is fully synced.
	s.querySyncStatus(s.ctx)

	// We notify other services in slasher that the beacon client is ready
	// and the connection is active.
	s.clientFeed.Send(true)

	// We register subscribers for any detected proposer/attester slashings
	// in the slasher services that we can submit to the beacon node
	// as they are found.
	go s.subscribeDetectedProposerSlashings(s.ctx, s.proposerSlashingsChan)
	go s.subscribeDetectedAttesterSlashings(s.ctx, s.attesterSlashingsChan)

}
