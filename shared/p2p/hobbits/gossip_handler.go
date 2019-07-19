package hobbits

import (
	"context"

	"github.com/gogo/protobuf/proto"
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

	h.Broadcast(context.Background(), //message)
}

func (h *HobbitsNode) gossipAttestation(message HobbitsMessage, header GossipHeader) {
	attestation := &pb.AttestationAnnounce{
		Hash: header.Hash[:],
	}

	h.Feed(attestation).Send(p2p.Message{
		Ctx: context.Background(),
		Data: attestation,
	})

	h.Broadcast(context.Background(), // message)
}