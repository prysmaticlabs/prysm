package interop_cold_start

import (
	"github.com/prysmaticlabs/prysm/shared"
)

var _ = shared.Service(&Service{})

type Service struct {

}

// NewColdStartService is an interoperability testing service to inject a deterministically generated genesis state
// into the beacon chain database and running services at start up. This service should not be used in production
// as it does not have any value other than ease of use for testing purposes.
func NewColdStartService() *Service {
	return &Service{}
}

// Start initializes the genesis state from configured flags.
func (s *Service) Start() {

}

// Stop does nothing.
func (s *Service) Stop() error {
	return nil
}

// Status always returns nil.
func (s *Service) Status() error {
	return nil
}