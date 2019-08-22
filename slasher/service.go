package slasher

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/keystore"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logrus.WithField("prefix", "validator")

// ValidatorService represents a service to manage the validator client
// routine.
type ValidatorService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	validator            Validator
	conn                 *grpc.ClientConn
	grpcServer           *grpc.Server
	endpoint             string
	withCert             string
	key                  *keystore.Key
	keys                 map[string]*keystore.Key
	logValidatorBalances bool
}

// Config for the validator service.
type Config struct {
	Endpoint             string
	CertFlag             string
	KeystorePath         string
	Password             string
	LogValidatorBalances bool
}

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context, cfg *Config) (*ValidatorService, error) {
	ctx, cancel := context.WithCancel(ctx)
	validatorFolder := cfg.KeystorePath
	validatorPrefix := params.BeaconConfig().ValidatorPrivkeyFileName
	ks := keystore.NewKeystore(cfg.KeystorePath)
	keys, err := ks.GetKeys(validatorFolder, validatorPrefix, cfg.Password)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "could not get private key")
	}
	var key *keystore.Key
	for _, v := range keys {
		key = v
		break
	}
	return &ValidatorService{
		ctx:                  ctx,
		cancel:               cancel,
		endpoint:             cfg.Endpoint,
		withCert:             cfg.CertFlag,
		keys:                 keys,
		key:                  key,
		logValidatorBalances: cfg.LogValidatorBalances,
	}, nil
}

// Start the validator service. Launches the main go routine for the validator
// client.
func (v *ValidatorService) Start() {
	pubkeys := make([][]byte, 0)
	for i := range v.keys {
		log.WithField("publicKey", fmt.Sprintf("%#x", v.keys[i].PublicKey.Marshal())).Info("Initializing new validator service")
		pubkey := v.keys[i].PublicKey.Marshal()
		pubkeys = append(pubkeys, pubkey)
	}

	var dialOpt grpc.DialOption
	if v.withCert != "" {
		creds, err := credentials.NewClientTLSFromFile(v.withCert, "")
		if err != nil {
			log.Errorf("Could not get valid credentials: %v", err)
			return
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn("You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
	}
	conn, err := grpc.DialContext(v.ctx, v.endpoint, dialOpt, grpc.WithStatsHandler(&ocgrpc.ClientHandler{}))
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", v.endpoint, err)
		return
	}
	log.Info("Successfully started gRPC connection")
	v.conn = conn
	v.validator = &validator{
		beaconClient:         pb.NewBeaconServiceClient(v.conn),
		validatorClient:      pb.NewValidatorServiceClient(v.conn),
		attesterClient:       pb.NewAttesterServiceClient(v.conn),
		proposerClient:       pb.NewProposerServiceClient(v.conn),
		keys:                 v.keys,
		pubkeys:              pubkeys,
		logValidatorBalances: v.logValidatorBalances,
		prevBalance:          make(map[[48]byte]uint64),
	}
	go run(v.ctx, v.validator)
}

// Stop the validator service.
func (v *ValidatorService) Stop() error {
	v.cancel()
	log.Info("Stopping service")
	if v.conn != nil {
		return v.conn.Close()
	}
	return nil
}

// Status ...
//
// WIP - not done.
func (v *ValidatorService) Status() error {
	if v.conn == nil {
		return errors.New("no connection to beacon RPC")
	}
	return nil
}

// Start the gRPC server.
func (s *Service) Start() {
	log.Info("Starting service")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		log.Errorf("Could not listen to port in Start() :%s: %v", s.port, err)
	}
	s.listener = lis
	log.WithField("port", s.port).Info("Listening on port")

	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.StreamInterceptor(middleware.ChainStreamServer(
			recovery.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(
			recovery.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
		)),
	}
	// TODO(#791): Utilize a certificate for secure connections
	// between beacon nodes and validator clients.
	if s.withCert != "" && s.withKey != "" {
		creds, err := credentials.NewServerTLSFromFile(s.withCert, s.withKey)
		if err != nil {
			log.Errorf("Could not load TLS keys: %s", err)
			s.credentialError = err
		}
		opts = append(opts, grpc.Creds(creds))
	} else {
		log.Warn("You are using an insecure gRPC connection! Provide a certificate and key to connect securely")
	}
	s.grpcServer = grpc.NewServer(opts...)

	beaconServer := &BeaconServer{
		beaconDB:            s.beaconDB,
		ctx:                 s.ctx,
		powChainService:     s.powChainService,
		chainService:        s.chainService,
		targetsFetcher:      s.chainService,
		operationService:    s.operationService,
		incomingAttestation: s.incomingAttestation,
		canonicalStateChan:  s.canonicalStateChan,
		chainStartChan:      make(chan time.Time, 1),
	}
	proposerServer := &ProposerServer{
		beaconDB:           s.beaconDB,
		chainService:       s.chainService,
		powChainService:    s.powChainService,
		operationService:   s.operationService,
		canonicalStateChan: s.canonicalStateChan,
	}
	attesterServer := &AttesterServer{
		beaconDB:         s.beaconDB,
		operationService: s.operationService,
		p2p:              s.p2p,
		cache:            cache.NewAttestationCache(),
	}
	validatorServer := &ValidatorServer{
		ctx:                s.ctx,
		beaconDB:           s.beaconDB,
		chainService:       s.chainService,
		canonicalStateChan: s.canonicalStateChan,
		powChainService:    s.powChainService,
	}
	nodeServer := &NodeServer{
		beaconDB:    s.beaconDB,
		server:      s.grpcServer,
		syncChecker: s.syncService,
	}
	beaconChainServer := &BeaconChainServer{
		beaconDB: s.beaconDB,
		pool:     s.operationService,
	}
	pb.RegisterBeaconServiceServer(s.grpcServer, beaconServer)
	pb.RegisterProposerServiceServer(s.grpcServer, proposerServer)
	pb.RegisterAttesterServiceServer(s.grpcServer, attesterServer)
	pb.RegisterValidatorServiceServer(s.grpcServer, validatorServer)
	ethpb.RegisterNodeServer(s.grpcServer, nodeServer)
	ethpb.RegisterBeaconChainServer(s.grpcServer, beaconChainServer)

	// Register reflection service on gRPC server.
	reflection.Register(s.grpcServer)

	go func() {
		for s.syncService.Status() != nil {
			time.Sleep(time.Second * params.BeaconConfig().RPCSyncCheck)
		}
		if s.listener != nil {
			if err := s.grpcServer.Serve(s.listener); err != nil {
				log.Errorf("Could not serve gRPC: %v", err)
			}
		}
	}()
}

// Stop the service.
func (s *Service) Stop() error {
	log.Info("Stopping service")
	s.cancel()
	if s.listener != nil {
		s.grpcServer.GracefulStop()
		log.Debug("Initiated graceful stop of gRPC server")
	}
	return nil
}

// Status returns nil or credentialError
func (s *Service) Status() error {
	if s.credentialError != nil {
		return s.credentialError
	}
	return nil
}
