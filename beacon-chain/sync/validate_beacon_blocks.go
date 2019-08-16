package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

// validateBeaconBlockPubSub checks that the incoming block has a valid BLS signature.
// If the BLS signature is any valid signature, this method rebroadcasts the message.
func validateBeaconBlockPubSub(ctx context.Context, msg proto.Message, p p2p.Broadcaster) bool {
	m := msg.(*ethpb.BeaconBlock)

	// TODO(3147): is this enough validation?
	_, err := bls.SignatureFromBytes(m.Signature)
	if err == nil {
		p.Broadcast(m)
	}
	return err == nil
}
