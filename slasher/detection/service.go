package detection

import (
	"context"

	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "detection")

// Service struct for the detection service of the slasher.
type Service struct {
	ctx      context.Context
	cancel   context.CancelFunc
	notifier beaconclient.Notifier
}

// Config options for the detection service.
type Config struct {
	Notifier beaconclient.Notifier
}

// NewDetectionService instantiation.
func NewDetectionService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:      ctx,
		cancel:   cancel,
		notifier: cfg.Notifier,
	}
}

// Stop the notifier service.
func (ds *Service) Stop() error {
	ds.cancel()
	log.Info("Stopping service")
	return nil
}

// Status returns an error if there exists an error in
// the notifier service.
func (ds *Service) Status() error {
	return nil
}

// Start the detection service runtime.
func (ds *Service) Start() {

}
