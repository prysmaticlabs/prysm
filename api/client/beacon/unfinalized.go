package beacon

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v3/encoding/ssz/detect"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	log "github.com/sirupsen/logrus"
)

func DownloadUnfinalizedBlocks(ctx context.Context, client *Client) ([]interfaces.BeaconBlock, error) {
	// Just to get the unmarshaler
	sb, err := client.GetState(ctx, IdHead)
	if err != nil {
		return nil, err
	}
	vu, err := detect.FromState(sb)
	if err != nil {
		return nil, errors.Wrap(err, "error detecting chain config for finalized state")
	}
	log.Printf("detected supported config in remote finalized state, name=%s, fork=%s", vu.Config.ConfigName, version.String(vu.Fork))

	// Get HTR of head block and finalized block
	currRoot, err := client.GetBlockRoot(ctx, IdHead)
	if err != nil {
		return nil, err
	}
	finalizedRoot, err := client.GetBlockRoot(ctx, IdFinalized)
	if err != nil {
		return nil, err
	}

	// Walk backwards from the head block till the finalized block
	var unfinalizedBlocks []interfaces.BeaconBlock
	for {
		if bytes.Equal(currRoot[:], finalizedRoot[:]) {
			break
		}

		bb, err := client.GetBlock(ctx, IdFromRoot(currRoot))
		if err != nil {
			return nil, errors.Wrapf(err, "error requesting block by root = %#x", currRoot)
		}
		b, err := vu.UnmarshalBeaconBlock(bb)
		if err != nil {
			return nil, errors.Wrap(err, "unable to unmarshal block to a supported type using the detected fork schedule")
		}
		realBlock := b.Block()

		unfinalizedBlocks = append(unfinalizedBlocks, realBlock)
		log.Printf("slot: %d", realBlock.Slot())

		// Advance to the previous block root
		currRoot = bytesutil.ToBytes32(realBlock.ParentRoot())
	}

	log.Printf("got %d unfinalizedBlocks", len(unfinalizedBlocks))

	return unfinalizedBlocks, nil
}
