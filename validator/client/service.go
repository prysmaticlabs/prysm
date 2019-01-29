package client

import (
	"context"

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
}

// Config for the validator service.
type Config struct {
	BeaconClient    pb.BeaconServiceClient
	ValidatorClient pb.ValidatorServiceClient
}

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context, cfg *Config) *ValidatorService {
	ctx, cancel := context.WithCancel(ctx)
	validator := &validator{
		beaconClient:    cfg.BeaconClient,
		validatorClient: cfg.ValidatorClient,
	}
	return &ValidatorService{
		ctx:       ctx,
		cancel:    cancel,
		validator: validator,
	}
}

// Start the validator service. Launches the main go routine for the validator
// client.
func (v *ValidatorService) Start() {
	go run(v.ctx, v.validator)
}

// Stop the validator service.
func (v *ValidatorService) Stop() error {
	v.cancel()
	return nil
}

// Status ...
//
// WIP - not done.
func (v *ValidatorService) Status() error {
	return nil
}
