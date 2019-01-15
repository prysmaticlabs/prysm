// Package proposer defines all relevant functionality for a Proposer actor
// within Ethereum Serenity.
package proposer

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/gogo/protobuf/proto"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "proposer")

type rpcClientService interface {
	ProposerServiceClient() pb.ProposerServiceClient
}

type beaconClientService interface {
	ProposerAssignmentFeed() *event.Feed
}

type rpcAttestationService interface {
	ProcessedAttestationFeed() *event.Feed
}

// Proposer holds functionality required to run a block proposer
// in Ethereum Serenity. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	ctx                context.Context
	cancel             context.CancelFunc
	beaconService      beaconClientService
	rpcClientService   rpcClientService
	assignmentChan     chan *pbp2p.BeaconBlock
	attestationService rpcAttestationService
	attestationChan    chan *pbp2p.Attestation
	pendingAttestation []*pbp2p.Attestation
	lock               sync.Mutex
}

// Config options for proposer service.
type Config struct {
	AssignmentBuf         int
	AttestationBufferSize int
	Assigner              beaconClientService
	AttesterFeed          rpcAttestationService
	Client                rpcClientService
}

// NewProposer creates a new attester instance.
func NewProposer(ctx context.Context, cfg *Config) *Proposer {
	ctx, cancel := context.WithCancel(ctx)
	return &Proposer{
		ctx:                ctx,
		cancel:             cancel,
		beaconService:      cfg.Assigner,
		rpcClientService:   cfg.Client,
		attestationService: cfg.AttesterFeed,
		assignmentChan:     make(chan *pbp2p.BeaconBlock, cfg.AssignmentBuf),
		attestationChan:    make(chan *pbp2p.Attestation, cfg.AttestationBufferSize),
		pendingAttestation: make([]*pbp2p.Attestation, 0),
		lock:               sync.Mutex{},
	}
}

// Start the main routine for a proposer.
func (p *Proposer) Start() {
	log.Info("Starting service")
	client := p.rpcClientService.ProposerServiceClient()

	go p.run(p.ctx.Done(), client)
	go p.processAttestation(p.ctx.Done())
}

// Stop the main loop.
func (p *Proposer) Stop() error {
	defer p.cancel()
	log.Info("Stopping service")
	return nil
}

// Status always returns nil.
// This service will be rewritten in the future so this service check is a
// no-op for now.
func (p *Proposer) Status() error {
	return nil
}

// DoesAttestationExist checks if an attester has already attested to a block.
func (p *Proposer) DoesAttestationExist(attestation *pbp2p.Attestation) bool {
	exists := false
	for _, record := range p.pendingAttestation {
		if bytes.Equal(record.ParticipationBitfield, attestation.ParticipationBitfield) {
			exists = true
			break
		}
	}
	return exists
}

// AddPendingAttestation adds a pending attestation to the memory so that it can be included
// in the next proposed block.
func (p *Proposer) AddPendingAttestation(attestation *pbp2p.Attestation) {
	p.pendingAttestation = append(p.pendingAttestation, attestation)
}

// AggregateAllSignatures aggregates all the signatures of the attesters. This is currently a
// stub for now till BLS/other signature schemes are implemented.
func (p *Proposer) AggregateAllSignatures(attestations []*pbp2p.Attestation) []uint32 {
	// TODO(#258): Implement Signature Aggregation.
	return []uint32{}
}

// GenerateBitmask creates the attestation bitmask from all the attester bitfields in the
// attestation records.
func (p *Proposer) GenerateBitmask(attestations []*pbp2p.Attestation) []byte {
	// TODO(#258): Implement bitmask where all attesters bitfields are aggregated.
	return []byte{}
}

// processAttestation processes incoming broadcasted attestations from the beacon node.
func (p *Proposer) processAttestation(done <-chan struct{}) {
	attestationSub := p.attestationService.ProcessedAttestationFeed().Subscribe(p.attestationChan)
	defer attestationSub.Unsubscribe()

	for {
		select {
		case <-done:
			log.Debug("Proposer context closed, exiting goroutine")
			return
		case attestationRecord := <-p.attestationChan:
			attestationExists := p.DoesAttestationExist(attestationRecord)
			if !attestationExists {
				p.AddPendingAttestation(attestationRecord)
				log.Info("Attestation stored in memory")
			}
		}

	}
}

// run the main event loop that listens for a proposer assignment.
func (p *Proposer) run(done <-chan struct{}, client pb.ProposerServiceClient) {
	sub := p.beaconService.ProposerAssignmentFeed().Subscribe(p.assignmentChan)
	defer sub.Unsubscribe()

	for {
		select {
		case <-done:
			log.Debug("Proposer context closed, exiting goroutine")
			return
		// When we receive an assignment on a slot, we leverage the fields
		// from the latest canonical beacon block to perform a proposal responsibility.
		case latestBeaconBlock := <-p.assignmentChan:
			p.receiveAssignment(latestBeaconBlock, client)
		}
	}
}

// receiveAssignment responds to the latest proposer assignment that the validator has received from the beacon
// node. It handles all the core proposal logic for the validator client, from creating the block to be proposed,
// to the signing of the proposal data and ultimately sending the block proposal to the beacon node.
func (p *Proposer) receiveAssignment(latestBeaconBlock *pbp2p.BeaconBlock, client pb.ProposerServiceClient) {
	log.Info("Performing proposer responsibility")

	block, err := p.computeBlockToBeProposed(latestBeaconBlock, client)
	if err != nil {
		log.Error(err)
		return
	}

	proposalData, err := p.createProposalDataFromBlock(block)
	if err != nil {
		log.Error(err)
		return
	}

	signature := p.signProposalData(proposalData)
	block.Signature[0] = signature[:]

	hash, err := client.ProposeBlock(p.ctx, block)
	if err != nil {
		log.Errorf("Could not propose block %v", err)
		return
	}

	log.Infof("Successfully proposed block with hash %#x", hash.BlockHash)
}

func (p *Proposer) computeBlockToBeProposed(latestBlock *pbp2p.BeaconBlock,
	client pb.ProposerServiceClient) (*pbp2p.BeaconBlock, error) {
	// Extract the hash of the latest beacon block to use as parent hash in
	// the proposal.
	if latestBlock == nil {
		return nil, fmt.Errorf("could not marshal nil latest beacon block")

	}

	latestBlockHash, err := hashutil.HashBeaconBlock(latestBlock)
	if err != nil {
		return nil, fmt.Errorf("could not hash latest beacon block: %v", err)
	}

	powChainHashRes, err := client.LatestPOWChainBlockHash(p.ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("could not get latest pow chain blockhash %v", err)
	}

	indexRes, err := client.ProposerIndex(p.ctx, &pb.ProposerIndexRequest{SlotNumber: latestBlock.Slot})
	if err != nil {
		return nil, fmt.Errorf("could not get proposer index: %v", err)
	}

	proposerBitfield := uint64(math.Pow(2, 7-float64(indexRes.Index)))
	attestation := &pbp2p.Attestation{
		ParticipationBitfield: []byte{byte(proposerBitfield)},
	}

	// To prevent any unaccounted attestations from being added.
	p.lock.Lock()
	defer p.lock.Unlock()

	signature := make([][]byte, 48)
	block := &pbp2p.BeaconBlock{
		Slot:               latestBlock.Slot + 1,
		ParentRootHash32:   latestBlockHash[:],
		RandaoRevealHash32: []byte{},
		DepositRootHash32:  powChainHashRes.BlockHash,
		Body: &pbp2p.BeaconBlockBody{
			Attestations: []*pbp2p.Attestation{attestation},
		},
		Signature: signature,
	}

	block, err = p.addStateRootToBlock(block, client)
	if err != nil {
		return nil, err
	}

	log.Infof("Created block proposal for slot %d", block.Slot)
	p.pendingAttestation = nil

	return block, nil
}

func (p *Proposer) addStateRootToBlock(block *pbp2p.BeaconBlock, client pb.ProposerServiceClient) (*pbp2p.BeaconBlock, error) {
	res, err := client.ComputeStateRoot(p.ctx, block)
	if err != nil {
		return nil, fmt.Errorf("unable to compute state root for block: %v", err)
	}

	block.StateRootHash32 = res.StateRoot

	return block, nil
}

func (p *Proposer) createProposalDataFromBlock(block *pbp2p.BeaconBlock) (*pbp2p.ProposalSignedData, error) {

	encodedBlock, err := proto.Marshal(block)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal block %v", err)
	}

	blockHash := hashutil.Hash(encodedBlock)

	return &pbp2p.ProposalSignedData{
		Slot:            block.Slot,
		Shard:           params.BeaconConfig().BeaconChainShardNumber, // placeholder
		BlockRootHash32: blockHash[:],
	}, nil
}

func (p *Proposer) signProposalData(data *pbp2p.ProposalSignedData) [32]byte {
	return [32]byte{'S', 'I', 'G', 'N', 'A', 'T', 'U', 'R', 'E'}
}
