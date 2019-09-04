package sync

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// pendingBlocks cache.
// TODO(3147): This cache value should represent a doubly linked list or some management to
// process pending blocks after a link to the canonical chain is found.
var pendingBlocks = ccache.New(ccache.Configure())

func (r *RegularSync) beaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	block := msg.(*ethpb.BeaconBlock)

	headState := r.chain.HeadState()

	// Ignore block older than last finalized checkpoint.
	if block.Slot < helpers.StartSlot(headState.FinalizedCheckpoint.Epoch) {
		log.Debugf("Received a block that's older than finalized checkpoint, %d < %d",
			block.Slot, helpers.StartSlot(headState.FinalizedCheckpoint.Epoch))
		return nil
	}

	// Handle block when the parent is unknown.
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(block.ParentRoot)) {
		blockRoot, err := ssz.SigningRoot(block)
		if err != nil {
			return err
		}
		b64BlockRoot := base64.StdEncoding.EncodeToString(blockRoot[:])
		pendingBlocks.Set(b64BlockRoot, block, 2*time.Hour)

		// TODO(3147): Request parent block from peers
		log.Warnf("Received a block from slot %d which we do not have the parent block in the database. "+
			"Requesting missing blocks from peers is not yet implemented.", block.Slot)
		return nil
	}

	return r.chain.ReceiveBlockNoPubsub(ctx, block)
}
