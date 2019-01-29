package client

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net/url"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "validator")

// ValidatorService represents a service to manage the validator client
// routine.
type ValidatorService struct {
	ctx       context.Context
	cancel    context.CancelFunc
	validator Validator
	conn     *grpc.ClientConn
	endpoint string
	withCert string
}

// Config for the validator service.
type Config struct {
	Endpoint string
	CertFlag string
}

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context, cfg *Config) *ValidatorService {
	ctx, cancel := context.WithCancel(ctx)
	return &ValidatorService{
		ctx:       ctx,
		cancel:    cancel,
		endpoint: cfg.Endpoint,
		withCert: cfg.CertFlag,
	}
}

// Start the validator service. Launches the main go routine for the validator
// client.
func (v *ValidatorService) Start() {
	var server grpc.DialOption
	if v.withCert != "" {
		creds, err := credentials.NewClientTLSFromFile(v.withCert, "")
		if err != nil {
			log.Errorf("Could not get valid credentials: %v", err)
			return
		}
		server = grpc.WithTransportCredentials(creds)
	} else {
		server = grpc.WithInsecure()
		log.Warn("You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
	}
	providerURL, err := url.Parse(v.endpoint)
	if err != nil {
		log.Fatalf("Unable to parse beacon RPC provider endpoint url: %v", err)
	}
	conn, err := grpc.Dial(fmt.Sprintf("[%s]:%s", providerURL.Hostname(), providerURL.Port()), server)
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", v.endpoint, err)
		return
	}
	log.Info("Successfully started gRPC connection")
	v.conn = conn
	v.validator = &validator{
		beaconClient: v.BeaconServiceClient(),
		validatorClient: v.ValidatorServiceClient(),
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

// BeaconServiceClient initializes a new beacon gRPC service using
// an underlying connection object.
// This wrapper is important because the underlying gRPC connection is
// only defined after the service .Start() function is called.
func (v *ValidatorService) BeaconServiceClient() pb.BeaconServiceClient {
	return pb.NewBeaconServiceClient(v.conn)
}

// ProposerServiceClient initializes a new proposer gRPC service using
// an underlying connection object.
// This wrapper is important because the underlying gRPC connection is
// only defined after the service .Start() function is called.
func (v *ValidatorService) ProposerServiceClient() pb.ProposerServiceClient {
	return pb.NewProposerServiceClient(v.conn)
}

// AttesterServiceClient initializes a new attester gRPC service using
// an underlying connection object.
// This wrapper is important because the underlying gRPC connection is
// only defined after the service .Start() function is called.
func (v *ValidatorService) AttesterServiceClient() pb.AttesterServiceClient {
	return pb.NewAttesterServiceClient(v.conn)
}

// ValidatorServiceClient initializes a new validator gRPC service using
// an underlying connection object.
// This wrapper is important because the underlying gRPC connection is
// only defined after the service .Start() function is called.
func (v *ValidatorService) ValidatorServiceClient() pb.ValidatorServiceClient {
	return pb.NewValidatorServiceClient(v.conn)
}
