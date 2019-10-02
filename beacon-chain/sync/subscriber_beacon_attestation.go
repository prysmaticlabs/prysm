package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// beaconAttestationSubscriber forwards the incoming validated attestation to the blockchain
// service for processing.
func (r *RegularSync) beaconAttestationSubscriber(ctx context.Context, msg proto.Message) error {
	if err := r.operations.HandleAttestation(ctx, msg.(*ethpb.Attestation)); err != nil {
		return err
	}

	err := r.chain.ReceiveAttestationNoPubsub(ctx, msg.(*ethpb.Attestation))
	if err != nil {
		root, sszErr := ssz.HashTreeRoot(msg)
		if sszErr != nil {
			return sszErr
		}
		inValidKey := invalid + string(root[:])
		recentlySeenRoots.Set(inValidKey, true, oneYear)
	}
	return err
}
