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
	AttesterServiceClient() pb.AttesterServiceClient
	ValidatorServiceClient() pb.ValidatorServiceClient
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
	pubKey           uint64
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
	attester := a.rpcClientService.AttesterServiceClient()
	validator := a.rpcClientService.ValidatorServiceClient()
	go a.run(a.ctx.Done(), attester, validator)
}

// Stop the main loop.
func (a *Attester) Stop() error {
	defer a.cancel()
	log.Info("Stopping service")
	return nil
}

// run the main event loop that listens for an attester assignment.
func (a *Attester) run(done <-chan struct{}, attester pb.AttesterServiceClient, validator pb.ValidatorServiceClient) {
	sub := a.assigner.AttesterAssignmentFeed().Subscribe(a.assignmentChan)
	defer sub.Unsubscribe()

	for {
		select {
		case <-done:
			log.Debug("Attester context closed, exiting goroutine")
			return
		case latestBeaconBlock := <-a.assignmentChan:
			log.Info("Performing attester responsibility")

		    pubKeyReq := &pb.PublicKey{
		    	PublicKey: a.pubKey,
			}

			shardID, err := validator.GetValidatorShardID(a.ctx, pubKeyReq)
			if err != nil {
				log.Errorf("Could not get attester Shard ID: %v", err)
				continue
			}
			a.shardID = shardID.ShardId

			attesterIndex, err := validator.GetValidatorIndex(a.ctx, pubKeyReq)
			if err != nil {
				log.Errorf("Could not get attester index: %v", err)
				continue
			}
			bitField := []byte{uint(attesterIndex.Index >> 1)}

			data, err := proto.Marshal(latestBeaconBlock)
			if err != nil {
				log.Errorf("Could not marshal latest beacon block: %v", err)
				continue
			}
			latestBlockHash := blake2b.Sum512(data)

			attestReq := &pb.AttestRequest{
				Attestation: &pbp2p.AggregatedAttestation{
					Slot:             latestBeaconBlock.GetSlotNumber(),
					ShardId:          a.shardID,
					ShardBlockHash:   latestBlockHash[:], // Is a stub for actual shard blockhash.
					AttesterBitfield: []byte{},           // TODO: Need to find which index this attester represents.
					AggregateSig:     []uint64{},         // TODO: Need Signature verification scheme/library
				},
			}

			res, err := attester.AttestHead(a.ctx, attestReq)
			if err != nil {
				log.Errorf("could not attest head: %v", err)
				continue
			}
			log.Infof("Attestation proposed successfully with hash 0x%x", res.AttestationHash)
		}
	}
}
