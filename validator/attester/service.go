// Package attester defines all relevant functionality for a Attester actor
// within Ethereum 2.0.
package attester

import (
	"context"

	"github.com/ethereum/go-ethereum/event"
	"github.com/gogo/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
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
	pubkey           []byte
}

// Config options for an attester service.
type Config struct {
	AssignmentBuf int
	ShardID       uint64
	Assigner      assignmentAnnouncer
	Client        rpcClientService
	Pubkey        []byte
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
		pubkey:           cfg.Pubkey,
	}
}

// Start the main routine for an attester.
func (a *Attester) Start() {
	log.Info("Starting service")
	attester := a.rpcClientService.AttesterServiceClient()
	validator := a.rpcClientService.ValidatorServiceClient()
	go a.run(attester, validator)
}

// Stop the main loop.
func (a *Attester) Stop() error {
	defer a.cancel()
	log.Info("Stopping service")
	return nil
}

// run the main event loop that listens for an attester assignment.
func (a *Attester) run(attester pb.AttesterServiceClient, validator pb.ValidatorServiceClient) {
	sub := a.assigner.AttesterAssignmentFeed().Subscribe(a.assignmentChan)
	defer sub.Unsubscribe()

	for {
		select {
		case <-a.ctx.Done():
			log.Debug("Attester context closed, exiting goroutine")
			return
		case latestBeaconBlock := <-a.assignmentChan:
			log.Info("Performing attester responsibility")

			data, err := proto.Marshal(latestBeaconBlock)
			if err != nil {
				log.Errorf("could not marshal latest beacon block: %v", err)
				continue
			}
			latestBlockHash := blake2b.Sum512(data)

			pubKeyReq := &pb.PublicKey{
				PublicKey: []byte{},
			}
			shardID, err := validator.ValidatorShardID(a.ctx, pubKeyReq)
			if err != nil {
				log.Errorf("could not get attester Shard ID: %v", err)
				continue
			}

			a.shardID = shardID.ShardId

			attesterIndex, err := validator.ValidatorIndex(a.ctx, pubKeyReq)
			if err != nil {
				log.Errorf("could not get attester index: %v", err)
				continue
			}
			attesterBitfield := shared.SetBitfield(int(attesterIndex.Index))

			attestReq := &pb.AttestRequest{
				Attestation: &pbp2p.AggregatedAttestation{
					Slot:             latestBeaconBlock.GetSlotNumber(),
					ShardId:          a.shardID,
					AttesterBitfield: attesterBitfield,
					ShardBlockHash:   latestBlockHash[:], // Is a stub for actual shard blockhash.
					AggregateSig:     []uint64{},         // TODO(258): Need Signature verification scheme/library
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
