package hobbits

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

func (h *HobbitsNode) gossipBlock(message HobbitsMessage, header GossipHeader) error {
	block := &pb.BeaconBlockAnnounce{
		Hash: header.Hash[:],
	}

	h.Feed(block).Send(p2p.Message{
		Ctx: context.Background(),
		Data: block,
	})

	err := h.Broadcast(context.Background(), //message)
	if err != nil {
		return errors.Wrap(err, "could not broadcast block")
	}

	return nil
}

func (h *HobbitsNode) gossipAttestation(message HobbitsMessage, header GossipHeader) error {
	attestation := &pb.AttestationAnnounce{
		Hash: header.Hash[:],
	}

	h.Feed(attestation).Send(p2p.Message{
		Ctx: context.Background(),
		Data: attestation,
	})

	err := h.Broadcast(context.Background(), // message)
	if err != nil {
		return errors.Wrap(err, "could not broadcast attestation")
	}

	return nil
}