package openapi

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/sniff"
	"github.com/prysmaticlabs/prysm/time/slots"
	log "github.com/sirupsen/logrus"
	"io"
	"strconv"
)

type OriginData struct {
	Checkpoint *ethpb.WeakSubjectivityCheckpoint
	StateBytes []byte
	blockBytes []byte
}

func getWeakSubjectivityEpochFromHead(ctx context.Context, client *Client) (types.Epoch, error) {
	headReader, err := client.GetStateById(StateIdHead)
	headBytes, err := io.ReadAll(headReader)
	if err != nil {
		return 0, errors.Wrap(err, "failed to read response body for get head state api call")
	}
	headState, err := sniff.BeaconState(headBytes)
	if err != nil {
		return 0, errors.Wrap(err, "error unmarshaling state to correct version")
	}
	cf, err := sniff.ConfigForkForState(headBytes)
	if err != nil {
		return 0, errors.Wrap(err, "error detecting chain config for beacon state")
	}
	log.Printf("detected supported config in remote head state, name=%s, fork=%s", cf.ConfigName.String(), cf.Fork)

	// LatestWeakSubjectivityEpoch uses package-level vars from the params package, so we need to override it
	params.OverrideBeaconConfig(cf.Config)
	epoch, err := helpers.LatestWeakSubjectivityEpoch(ctx, headState)
	if err != nil {
		return 0, errors.Wrap(err, "error computing the weak subjectivity epoch from head state")
	}

	return epoch, nil
}

func downloadBackwardsCompatible(ctx context.Context, client *Client) (*OriginData, error) {
	epoch, err := getWeakSubjectivityEpochFromHead(ctx, client)
	if err != nil {
		return nil, errors.Wrap(err, "error computing weak subjectivity epoch via head state inspection")
	}
	// use first slot of the epoch for the block slot
	slot, err := slots.EpochStart(epoch)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error computing first slot of epoch=%d", epoch))
	}

	sReader, err := client.GetStateById(strconv.Itoa(int(slot)))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to request state by slot from api, slot=%d", slot))
	}

	stateBytes, err := io.ReadAll(sReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body for get checkpoint state api call")
	}
	cf, err := sniff.ConfigForkForState(stateBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error detecting chain config for beacon state")
	}
	log.Printf("detected supported config in checkpoint state, name=%s, fork=%s", cf.ConfigName.String(), cf.Fork)

	state, err := sniff.BeaconStateForConfigFork(stateBytes, cf)
	if err != nil {
		return nil, errors.Wrap(err, "error using detected config fork to unmarshal state bytes")
	}

	// compute state and block roots
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error computing hash_tree_root of state")
	}
	blockRoot, err := state.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "error computing hash_tree_root of latest_block_header")
	}

	bReader, err := client.GetBlockByRoot(fmt.Sprintf("%#x", blockRoot))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error requesting block by root = %#x", blockRoot))
	}
	blockBytes, err := io.ReadAll(bReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body for get checkpoint block api call")
	}

	return &OriginData{
		Checkpoint:         &ethpb.WeakSubjectivityCheckpoint{
			BlockRoot: blockRoot[:],
			StateRoot: stateRoot[:],
			Epoch:     epoch,
		},
		StateBytes: stateBytes,
		blockBytes: blockBytes,
	}, nil
}

func DownloadOriginData(ctx context.Context, client *Client) (*OriginData, error) {
	wsc, err := client.GetWeakSubjectivityCheckpoint()
	if err != nil {
		// a 404 is expected if querying an endpoint that doesn't support the weak subjectivity checkpoint api
		if !errors.Is(err, ErrNotFound) {
			return nil, errors.Wrap(err, "unexpected API response for prysm-only weak subjectivity checkpoint API")
		}
		// ok so it's a 404, use the head state method
		return downloadBackwardsCompatible(ctx, client)
	}
	log.Printf("server weak subjectivity checkpoint response - epoch=%d, block_root=%#x, state_root=%#x", wsc.Epoch, wsc.BlockRoot, wsc.StateRoot)

	// use first slot of the epoch for the block slot
	slot, err := slots.EpochStart(wsc.Epoch)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error computing first slot of epoch=%d", wsc.Epoch))
	}
	log.Printf("wtf 1")

	sReader, err := client.GetStateByRoot(fmt.Sprintf("%#x", wsc.StateRoot))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to request state by slot from api, slot=%d", slot))
	}
	log.Printf("wtf 2")

	stateBytes, err := io.ReadAll(sReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body for get checkpoint state api call")
	}
	log.Printf("wtf 3")
	cf, err := sniff.ConfigForkForState(stateBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error detecting chain config for beacon state")
	}
	log.Printf("wtf 4")
	log.Printf("detected supported config in checkpoint state, name=%s, fork=%s", cf.ConfigName.String(), cf.Fork)

	state, err := sniff.BeaconStateForConfigFork(stateBytes, cf)
	if err != nil {
		return nil, errors.Wrap(err, "error using detected config fork to unmarshal state bytes")
	}
	log.Printf("wtf 5")

	blockRoot, err := state.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "error computing hash_tree_root of latest_block_header")
	}
	log.Printf("wtf 6")
	if blockRoot != bytesutil.ToBytes32(wsc.BlockRoot) {
		//return nil, fmt.Errorf()
		log.Warn("checkpoint block root doesn't match hash_tree_root(state.latest_block_header)")
	}
	log.Printf("block_root in latest_block_header == %#x", blockRoot)

	bReader, err := client.GetBlockByRoot(fmt.Sprintf("%#x", blockRoot))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error requesting block by root = %#x", blockRoot))
	}
	blockBytes, err := io.ReadAll(bReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body for get checkpoint block api call")
	}
	block, err := sniff.BlockForConfigFork(blockBytes, cf)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal block to a supported type using the detected fork schedule")
	}
	log.Printf("BeaconState slot=%d, Block slot=%d", state.Slot(), block.Block().Slot())
	log.Printf("BeaconState htr=%d, Block state_root=%d", state.Slot(), block.Block().StateRoot())
	return &OriginData{
		Checkpoint: wsc,
		StateBytes: stateBytes,
		blockBytes: blockBytes,
	}, nil
}