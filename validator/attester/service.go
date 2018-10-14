// Package attester defines all relevant functionality for a Attester actor
// within Ethereum 2.0.
package attester

import (
	"context"

	"github.com/gogo/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "attester")

type rpcClientService interface {
	AttesterServiceClient() pb.AttesterServiceClient
	ValidatorServiceClient() pb.ValidatorServiceClient
}

type beaconClientService interface {
	AttesterAssignmentFeed() *event.Feed
	PublicKey() []byte
}

// Attester holds functionality required to run a block attester
// in Ethereum 2.0.
type Attester struct {
	ctx              context.Context
	cancel           context.CancelFunc
	beaconService    beaconClientService
	rpcClientService rpcClientService
	assignmentChan   chan *pbp2p.BeaconBlock
	shardID          uint64
}

// Config options for an attester service.
type Config struct {
	AssignmentBuf int
	ShardID       uint64
	Assigner      beaconClientService
	Client        rpcClientService
}

// NewAttester creates a new attester instance.
func NewAttester(ctx context.Context, cfg *Config) *Attester {
	ctx, cancel := context.WithCancel(ctx)
	return &Attester{
		ctx:              ctx,
		cancel:           cancel,
		beaconService:    cfg.Assigner,
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
	sub := a.beaconService.AttesterAssignmentFeed().Subscribe(a.assignmentChan)
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
			latestBlockHash := hashutil.Hash(data)

			pubKeyReq := &pb.PublicKey{
				PublicKey: a.beaconService.PublicKey(),
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
			attesterBitfield := bitutil.SetBitfield(int(attesterIndex.Index))

			attestReq := &pb.AttestRequest{
				Attestation: &pbp2p.AggregatedAttestation{
					Slot:             latestBeaconBlock.GetSlot(),
					Shard:            a.shardID,
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
