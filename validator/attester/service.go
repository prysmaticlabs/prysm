// Package attester defines all relevant functionality for a Attester actor
// within Ethereum 2.0.
package attester

import (
	"context"

	"github.com/ethereum/go-ethereum/event"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "attester")

type assignmentAnnouncer interface {
	AttesterAssignment() *event.Feed
}

// Attester holds functionality required to run a block attester
// in Ethereum 2.0.
type Attester struct {
	ctx              context.Context
	cancel           context.CancelFunc
	assigner         assignmentAnnouncer
	announcementChan chan bool
}

// Config options for an attester service.
type Config struct {
	AnnouncementBuf int
	Assigner        assignmentAnnouncer
}

// NewAttester creates a new attester instance.
func NewAttester(ctx context.Context, cfg *Config) *Attester {
	ctx, cancel := context.WithCancel(ctx)
	return &Attester{
		ctx:              ctx,
		cancel:           cancel,
		assigner:         cfg.Assigner,
		announcementChan: make(chan bool, cfg.AnnouncementBuf),
	}
}

// Start the main routine for an attester.
func (p *Attester) Start() {
	log.Info("Starting service")
	go p.run(p.ctx.Done())
}

// Stop the main loop.
func (p *Attester) Stop() error {
	defer p.cancel()
	log.Info("Stopping service")
	return nil
}

// run the main event loop that listens for an attester assignment.
func (p *Attester) run(done <-chan struct{}) {
	sub := p.assigner.AttesterAssignment().Subscribe(p.announcementChan)
	defer sub.Unsubscribe()
	for {
		select {
		case <-done:
			log.Debug("Attester context closed, exiting goroutine")
			return
		case <-p.announcementChan:
			log.Info("Performing attester responsibility")
		}
	}
}
