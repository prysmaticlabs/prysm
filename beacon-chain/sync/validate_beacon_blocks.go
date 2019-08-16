package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"

	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

// validateBeaconBlockPubSub checks that the incoming block has a valid BLS signature.
// Blocks that have already been seen are ignored. If the BLS signature is any valid signature,
// this method rebroadcasts the message.
func (r *RegularSync) validateBeaconBlockPubSub(ctx context.Context, msg proto.Message, p p2p.Broadcaster) bool {
	m := msg.(*ethpb.BeaconBlock)

	blockRoot, err := ssz.SigningRoot(m)
	if err != nil {
		log.WithField("validate", "beacon block").WithError(err).Error("Failed to get signing root of block")
		return false
	}

	if r.db.HasBlock(ctx, blockRoot) {
		return false
	}

	_, err = bls.SignatureFromBytes(m.Signature)
	if err == nil {
		p.Broadcast(m)
	}
	return err == nil
}
