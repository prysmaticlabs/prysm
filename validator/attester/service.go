// Package attester defines all relevant functionality for a Attester actor
// within Ethereum 2.0.
package attester

import (
	"context"

	"github.com/ethereum/go-ethereum/event"
	"github.com/gogo/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	shardingp2p "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
	blake2b "golang.org/x/crypto/blake2b"
)

var log = logrus.WithField("prefix", "attester")

type rpcClientService interface {
	AttesterServiceClient() pb.AttesterServiceClient
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
	p2p              shared.P2P
	blockBuf         chan p2p.Message
}

// Config options for an attester service.
type Config struct {
	AssignmentBuf   int
	ShardID         uint64
	Assigner        assignmentAnnouncer
	BlockBufferSize int
	Client          rpcClientService
}

// NewAttester creates a new attester instance.
func NewAttester(ctx context.Context, cfg *Config, validatorP2P shared.P2P) *Attester {
	ctx, cancel := context.WithCancel(ctx)
	return &Attester{
		ctx:              ctx,
		cancel:           cancel,
		assigner:         cfg.Assigner,
		rpcClientService: cfg.Client,
		shardID:          cfg.ShardID,
		assignmentChan:   make(chan *pbp2p.BeaconBlock, cfg.AssignmentBuf),
		p2p:              validatorP2P,
		blockBuf:         make(chan p2p.Message, cfg.BlockBufferSize),
	}
}

// Start the main routine for an attester.
func (a *Attester) Start() {
	log.Info("Starting service")
	client := a.rpcClientService.AttesterServiceClient()
	go a.run(a.ctx.Done(), client)
}

// Stop the main loop.
func (a *Attester) Stop() error {
	defer a.cancel()
	log.Info("Stopping service")
	return nil
}

// CreateAttestation will cause the attester to attest to the provided beacon block.
func (a *Attester) CreateAttestation(block *pbp2p.BeaconBlock) *pbp2p.AttestationRecord {
	attestation := &pbp2p.AttestationRecord{}
	attestation.Slot = block.GetSlotNumber()

	// TODO(#487): Attesters responsibilities will be handled by this PR(#487).
	attestation.AttesterBitfield = []byte{}

	return attestation
}

// run the main event loop that listens for an attester assignment.
func (a *Attester) run(done <-chan struct{}, client pb.AttesterServiceClient) {
	sub := a.assigner.AttesterAssignmentFeed().Subscribe(a.assignmentChan)
	blocksub := a.p2p.Subscribe(&shardingp2p.BlockBroadcast{}, a.blockBuf)
	defer sub.Unsubscribe()
	defer blocksub.Unsubscribe()

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
		case msg := <-a.blockBuf:
			data, ok := msg.Data.(*shardingp2p.BlockBroadcast)
			if !ok {
				log.Error("Received malformed attestation p2p message")
				continue
			}
			attestation := a.CreateAttestation(data.GetBeaconBlock())
			a.p2p.Broadcast(attestation)

			log.Info("Attestation Broadcasted to network")
		}
	}
}
