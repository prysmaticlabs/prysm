package client

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "validator")

// ValidatorService represents a service to manage the validator client
// routine.
type ValidatorService struct {
	ctx       context.Context
	cancel    context.CancelFunc
	validator Validator
	conn      *grpc.ClientConn
	endpoint  string
	withCert  string
	p2p       p2p.Broadcaster
}

// Config for the validator service.
type Config struct {
	Endpoint string
	CertFlag string
	P2p      p2p.Broadcaster
}

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context, cfg *Config) *ValidatorService {
	ctx, cancel := context.WithCancel(ctx)
	return &ValidatorService{
		ctx:      ctx,
		cancel:   cancel,
		endpoint: cfg.Endpoint,
		withCert: cfg.CertFlag,
		p2p:      cfg.P2p,
	}
}

// Start the validator service. Launches the main go routine for the validator
// client.
func (v *ValidatorService) Start() {
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
	conn, err := grpc.DialContext(v.ctx, v.endpoint, dialOpt)
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", v.endpoint, err)
		return
	}
	log.Info("Successfully started gRPC connection")
	v.conn = conn
	v.validator = &validator{
		beaconClient:    pb.NewBeaconServiceClient(v.conn),
		validatorClient: pb.NewValidatorServiceClient(v.conn),
		p2p:             v.p2p,
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
