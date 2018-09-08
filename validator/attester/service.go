// Package attester defines all relevant functionality for a Attester actor
// within Ethereum 2.0.
package attester

import (
	"context"

	"github.com/ethereum/go-ethereum/event"
	"github.com/gogo/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	blake2b "golang.org/x/crypto/blake2b"
)

var log = logrus.WithField("prefix", "attester")

type rpcClientService interface {
	attesterServiceClient() pb.AttesterServiceClient
}

type assignmentAnnouncer interface {
	AttesterAssignmentFeed() *event.Feed
}

// Attester holds functionality required to run a block attester
// in Ethereum 2.0.
type Attester struct {
	ctx              context.Context
	cancel           context.CancelFunc
	assigner         assignmentAnnouncer
	rpcClientService rpcClientService
	assignmentChan   chan *pbp2p.BeaconBlock
	shardID          uint64
}

// Config options for an attester service.
type Config struct {
	AssignmentBuf int
	ShardID       uint64
	Assigner      assignmentAnnouncer
	Client        rpcClientService
}

// NewAttester creates a new attester instance.
func NewAttester(ctx context.Context, cfg *Config) *Attester {
	ctx, cancel := context.WithCancel(ctx)
	return &Attester{
		ctx:              ctx,
		cancel:           cancel,
		assigner:         cfg.Assigner,
		rpcClientService: cfg.Client,
		shardID:          cfg.ShardID,
		assignmentChan:   make(chan *pbp2p.BeaconBlock, cfg.AssignmentBuf),
	}
}

// Start the main routine for an attester.
func (a *Attester) Start() {
	log.Info("Starting service")
	client := a.rpcClientService.attesterServiceClient()
	go a.run(a.ctx.Done(), client)
}

// Stop the main loop.
func (a *Attester) Stop() error {
	defer a.cancel()
	log.Info("Stopping service")
	return nil
}

// run the main event loop that listens for an attester assignment.
func (a *Attester) run(done <-chan struct{}, client pb.AttesterServiceClient) {
	sub := a.assigner.AttesterAssignmentFeed().Subscribe(a.assignmentChan)
	defer sub.Unsubscribe()
	for {
		select {
		case <-done:
			log.Debug("Attester context closed, exiting goroutine")
			return
		case latestBeaconBlock := <-a.assignmentChan:
			log.Info("Performing attester responsibility")

			data, err := proto.Marshal(latestBeaconBlock)
			if err != nil {
				log.Errorf("Could not marshal latest beacon block: %v", err)
				continue
			}
			latestBlockHash := blake2b.Sum512(data)

			req := &pb.AttestRequest{
				Attestation: &pbp2p.AttestationRecord{
					Slot:             latestBeaconBlock.GetSlotNumber(),
					ShardId:          a.shardID,
					ShardBlockHash:   latestBlockHash[:],
					AttesterBitfield: []byte{},   // TODO: Need to find which index this attester represents.
					AggregateSig:     []uint64{}, // TODO: Need Signature verification scheme/library
				},
			}

			res, err := client.AttestHead(a.ctx, req)
			if err != nil {
				log.Errorf("could not attest head: %v", err)
				continue
			}
			log.Infof("Attestation proposed successfully with hash 0x%x", res.AttestationHash)
		}
	}
}
