// Package attester defines all relevant functionality for a Attester actor
// within Ethereum Serenity.
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
}

// Attester holds functionality required to run a block attester
// in Ethereum Serenity.
type Attester struct {
	ctx              context.Context
	cancel           context.CancelFunc
	beaconService    beaconClientService
	rpcClientService rpcClientService
	assignmentChan   chan *pbp2p.BeaconBlock
	shardID          uint64
	publicKey        []byte
}

// Config options for an attester service.
type Config struct {
	AssignmentBuf int
	ShardID       uint64
	Assigner      beaconClientService
	Client        rpcClientService
	PublicKey     []byte
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
		publicKey:        cfg.PublicKey,
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

// Status always returns nil.
// This service will be rewritten in the future so this service check is a
// no-op for now.
func (a *Attester) Status() error {
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

			if latestBeaconBlock == nil {
				log.Errorf("could not marshal nil latest beacon block")
				continue
			}
			data, err := proto.Marshal(latestBeaconBlock)
			if err != nil {
				log.Errorf("could not marshal latest beacon block: %v", err)
				continue
			}
			latestBlockHash := hashutil.Hash(data)

			pubKeyReq := &pb.PublicKey{
				PublicKey: a.publicKey,
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
				Attestation: &pbp2p.Attestation{
					ParticipationBitfield: attesterBitfield,
					AggregateSignature:    []byte{}, // TODO(258): Need Signature verification scheme/library
					Data: &pbp2p.AttestationData{
						Slot:                 latestBeaconBlock.Slot,
						Shard:                a.shardID,
						ShardBlockRootHash32: latestBlockHash[:],
					},
				},
			}

			res, err := attester.AttestHead(a.ctx, attestReq)
			if err != nil {
				log.Errorf("could not attest head: %v", err)
				continue
			}
			log.Infof("Attestation proposed successfully with hash %#x", res.AttestationHash)
		}
	}
}
