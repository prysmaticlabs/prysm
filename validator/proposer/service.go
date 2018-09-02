// Package proposer defines all relevant functionality for a Proposer actor
// within Ethereum 2.0.
package proposer

import (
	"context"

	"github.com/ethereum/go-ethereum/event"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "proposer")

type rpcClientService interface {
	ProposerServiceClient() pb.ProposerServiceClient
}

type assignmentAnnouncer interface {
	ProposerAssignment() *event.Feed
}

// Proposer holds functionality required to run a block proposer
// in Ethereum 2.0. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	ctx              context.Context
	cancel           context.CancelFunc
	assigner         assignmentAnnouncer
	rpcClientService rpcClientService
}

// NewProposer creates a new attester instance.
func NewProposer(ctx context.Context, assigner assignmentAnnouncer, client rpcClientService) *Proposer {
	ctx, cancel := context.WithCancel(ctx)
	return &Proposer{
		ctx:              ctx,
		cancel:           cancel,
		assigner:         assigner,
		rpcClientService: client,
	}
}

// Start the main routine for a proposer.
func (p *Proposer) Start() {
	log.Info("Starting service")
	client := p.rpcClientService.ProposerServiceClient()
	go p.run(p.ctx.Done(), client)
}

// Stop the main loop.
func (p *Proposer) Stop() error {
	defer p.cancel()
	log.Info("Stopping service")
	return nil
}

// run the main event loop that listens for a proposer assignment.
func (p *Proposer) run(done <-chan struct{}, client pb.ProposerServiceClient) {
	announcement := make(chan bool, 100)
	sub := p.assigner.ProposerAssignment().Subscribe(announcement)
	defer sub.Unsubscribe()
	for {
		select {
		case <-done:
			log.Debug("Proposer context closed, exiting goroutine")
			return
		// TODO: Instead subscribe to a proposal assignment feed that contains
		// important fields from the currently received beacon block that will be put into
		// a proposal RPC message such as slot number and parent hash.
		//
		// TODO: On the beacon node side, calculate active and crystallized and update the
		// active/crystallize state hash values in the proposed block.
		case <-announcement:
			log.Info("Performing proposer responsibility")

			// Sending empty values for now.
			// TODO: Implement real proposals with randao reveals and attestation fields.
			// TODO: Add timestamp, parent hash, and slot number.
			req := &pb.ProposeRequest{
				RandaoReveal:            []byte{},
				AttestationBitmask:      []byte{},
				AttestationAggregateSig: []uint32{},
			}

			res, err := client.ProposeBlock(p.ctx, req)
			if err != nil {
				log.Errorf("Could not propose block: %v", err)
				continue
			}
			log.Infof("Block proposed successfully with hash 0x%x", res.BlockHash)
		}
	}
}
