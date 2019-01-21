package client

import (
	"context"

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

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context) *ValidatorService {
	ctx, cancel := context.WithCancel(ctx)
	return &ValidatorService{
		ctx:    ctx,
		cancel: cancel,
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
