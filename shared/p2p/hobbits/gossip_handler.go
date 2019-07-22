package hobbits

import (
	"context"
	"log"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

func (h *HobbitsNode) gossipBlock(message HobbitsMessage, header GossipHeader) {
	block := &pb.BeaconBlockAnnounce{
		Hash: header.Hash[:],
	}

	h.Feed(block).Send(p2p.Message{
		Ctx: context.Background(),
		Data: block,
	})

	h.Broadcast(context.WithValue(context.Background(), "message_hash", header.MessageHash), block)
}

func (h *HobbitsNode) gossipAttestation(message HobbitsMessage, header GossipHeader) {
	attestation := &pb.AttestationAnnounce{
		Hash: header.Hash[:],
	}

	log.Println("attestation announce protobuf built...") // TODO delete

	h.Feed(attestation).Send(p2p.Message{
		Ctx: context.Background(),
		Data: attestation,
	})

	log.Println("attestation sent through the feed") // TODO delete

	h.Broadcast(context.WithValue(context.Background(), "message_hash", header.MessageHash), attestation)

	log.Println("broadcast finished") // TODO delete
}