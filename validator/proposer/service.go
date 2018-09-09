// Package proposer defines all relevant functionality for a Proposer actor
// within Ethereum 2.0.
package proposer

import (
	"context"
	"errors"
	"reflect"

	"github.com/ethereum/go-ethereum/event"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	blake2b "github.com/minio/blake2b-simd"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	shardingp2p "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "proposer")

type rpcClientService interface {
	ProposerServiceClient() pb.ProposerServiceClient
}

type assignmentAnnouncer interface {
	ProposerAssignmentFeed() *event.Feed
}

// Proposer holds functionality required to run a block proposer
// in Ethereum 2.0. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	ctx              context.Context
	cancel           context.CancelFunc
	assigner         assignmentAnnouncer
	rpcClientService rpcClientService
	assignmentChan   chan *pbp2p.BeaconBlock
	p2p              shared.P2P
	attestationBuf   chan p2p.Message
	blockMapping     map[uint64]*pbp2p.BeaconBlock
}

// Config options for proposer service.
type Config struct {
	AssignmentBuf         int
	AttestationBufferSize int
	Assigner              assignmentAnnouncer
	Client                rpcClientService
}

// NewProposer creates a new attester instance.
func NewProposer(ctx context.Context, cfg *Config, validatorP2P shared.P2P) *Proposer {
	ctx, cancel := context.WithCancel(ctx)
	return &Proposer{
		ctx:              ctx,
		cancel:           cancel,
		assigner:         cfg.Assigner,
		rpcClientService: cfg.Client,
		assignmentChan:   make(chan *pbp2p.BeaconBlock, cfg.AssignmentBuf),
		p2p:              validatorP2P,
		attestationBuf:   make(chan p2p.Message, cfg.AttestationBufferSize),
		blockMapping:     make(map[uint64]*pbp2p.BeaconBlock),
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

func (p *Proposer) SaveBlockToMemory(block *pbp2p.BeaconBlock) {
	p.blockMapping[block.GetSlotNumber()] = block
}

func (p *Proposer) GetBlockFromMemory(slotnumber uint64) (*pbp2p.BeaconBlock, error) {
	block := p.blockMapping[slotnumber]
	if block.GetSlotNumber() != slotnumber {
		return nil, errors.New("invalid block saved in memory")
	}
	return block, nil
}

func (p *Proposer) DoesAttestationExist(attestation *pbp2p.AttestationRecord, records []*pbp2p.AttestationRecord) bool {
	exists := false
	for _, record := range records {
		if reflect.DeepEqual(record, attestation) {
			exists = true
			break
		}
	}
	return exists
}

// run the main event loop that listens for a proposer assignment.
func (p *Proposer) run(done <-chan struct{}, client pb.ProposerServiceClient) {
	attestationSub := p.p2p.Subscribe(shardingp2p.AttestationBroadcast{}, p.attestationBuf)
	sub := p.assigner.ProposerAssignmentFeed().Subscribe(p.assignmentChan)

	defer attestationSub.Unsubscribe()
	defer sub.Unsubscribe()

	for {
		select {
		case <-done:
			log.Debug("Proposer context closed, exiting goroutine")
			return

		// TODO: On the beacon node side, calculate active and crystallized and update the
		// active/crystallize state hash values in the proposed block.

		// When we receive an assignment on a slot, we leverage the fields
		// from the latest canonical beacon block to perform a proposal responsibility.
		case latestBeaconBlock := <-p.assignmentChan:
			log.Info("Performing proposer responsibility")

			// Extract the hash of the latest beacon block to use as parent hash in
			// the proposal.
			data, err := proto.Marshal(latestBeaconBlock)
			if err != nil {
				log.Errorf("Could not marshal latest beacon block: %v", err)
				continue
			}
			latestBlockHash := blake2b.Sum512(data)

			proposalBlock := &pbp2p.BeaconBlock{
				ParentHash:   latestBlockHash[:],
				SlotNumber:   latestBeaconBlock.GetSlotNumber() + 1,
				RandaoReveal: []byte{},
				Attestations: make([]*pbp2p.AttestationRecord, 0),
			}

			blockToBroadcast := &shardingp2p.BlockBroadcast{BeaconBlock: proposalBlock}

			p.SaveBlockToMemory(proposalBlock)
			p.p2p.Broadcast(blockToBroadcast)

			// TODO: Implement real proposals with randao reveals and attestation fields.
			req := &pb.ProposeRequest{
				ParentHash:              latestBlockHash[:],
				SlotNumber:              latestBeaconBlock.GetSlotNumber() + 1,
				RandaoReveal:            []byte{},
				AttestationBitmask:      []byte{},
				AttestationAggregateSig: []uint32{},
				Timestamp:               ptypes.TimestampNow(),
			}

			res, err := client.ProposeBlock(p.ctx, req)
			if err != nil {
				log.Errorf("Could not propose block: %v", err)
				continue
			}
			log.Infof("Block proposed successfully with hash 0x%x", res.BlockHash)
		case msg := <-p.attestationBuf:
			data, ok := msg.Data.(*shardingp2p.AttestationBroadcast)
			// TODO: Handle this at p2p layer.
			if !ok {
				log.Error("Received malformed attestation p2p message")
				continue
			}

			attestationRecord := data.GetAttestationRecord()
			block, err := p.GetBlockFromMemory(attestationRecord.GetSlot())
			if err != nil {
				log.Errorf("Unable to retrieve block from memory %v", err)
				continue
			}
			attestationExists := p.DoesAttestationExist(attestationRecord, block.GetAttestations())
			if !attestationExists {
				block.Attestations = append(block.Attestations, attestationRecord)
			}
		}
	}
}
