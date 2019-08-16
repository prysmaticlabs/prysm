package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func (r *RegularSync) beaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	m := msg.(*ethpb.BeaconBlock)

	_ = m

	//block, _, isValid, err := rs.validateAndProcessBlock(ctx, msg)
	//if err != nil {
	//	return err
	//}
	//
	//if !isValid {
	//	return nil
	//}
	//
	//blockRoot, err := ssz.SigningRoot(block)
	//if err != nil {
	//	return err
	//}
	//
	//if rs.db.IsEvilBlockHash(blockRoot) {
	//	log.WithField("blockRoot", bytesutil.Trunc(blockRoot[:])).Debug("Skipping blacklisted block")
	//	return nil
	//}
	//
	//// If the block has a child, we then clear it from the blocks pending processing
	//// and call receiveBlock recursively. The recursive function call will stop once
	//// the block we process no longer has children.
	//if child, ok := rs.hasChild(blockRoot); ok {
	//	// We clear the block root from the pending processing map.
	//	rs.clearPendingBlock(blockRoot)
	//	return rs.processBlockAndFetchAncestors(ctx, child)
	//}
	//return nil

	return nil
}
