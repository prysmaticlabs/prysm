package openapi

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/sniff"
	"github.com/prysmaticlabs/prysm/time/slots"
	log "github.com/sirupsen/logrus"
)

type WeakSubjectivityData struct {
	BlockRoot [32]byte
	StateRoot [32]byte
	Epoch     types.Epoch
}

// OriginData represents the BeaconState and SignedBeaconBlock necessary to start an empty Beacon Node
// using Checkpoint Sync.
type OriginData struct {
	wsd *WeakSubjectivityData
	sb  []byte
	bb  []byte
	st  state.BeaconState
	b   block.SignedBeaconBlock
	cf  *sniff.ConfigFork
}

// WeakSubjectivity returns the WeakSubjectivityData determined by DownloadOriginData.
func (od *OriginData) WeakSubjectivity() *WeakSubjectivityData {
	return od.wsd
}

// SaveBlock saves the downloaded block to a unique file in the given path.
// For readability and collision avoidance, the file name includes: type, config name, slot and root
func (od *OriginData) SaveBlock(dir string) (string, error) {
	statePath := path.Join(dir, fname("state", od.cf, od.st.Slot(), od.wsd.BlockRoot))
	return statePath, os.WriteFile(statePath, od.sb, 0600)
}

// SaveState saves the downloaded state to a unique file in the given path.
// For readability and collision avoidance, the file name includes: type, config name, slot and root
func (od *OriginData) SaveState(dir string) (string, error) {
	statePath := path.Join(dir, fname("state", od.cf, od.st.Slot(), od.wsd.StateRoot))
	return statePath, os.WriteFile(statePath, od.sb, 0600)
}

// StateBytes returns the ssz-encoded bytes of the downloaded BeaconState value.
func (od *OriginData) StateBytes() []byte {
	return od.sb
}

// BlockBytes returns the ssz-encoded bytes of the downloaded SignedBeaconBlock value.
func (od *OriginData) BlockBytes() []byte {
	return od.bb
}

func fname(prefix string, cf *sniff.ConfigFork, slot types.Slot, root [32]byte) string {
	return fmt.Sprintf("%s_%s_%s_%d-%#x.ssz", prefix, cf.ConfigName.String(), cf.Fork.String(), slot, root)
}

// this method downloads the head state, which can be used to find the correct chain config
// and use prysm's helper methods to compute the latest weak subjectivity epoch.
func getWeakSubjectivityEpochFromHead(ctx context.Context, client *Client) (types.Epoch, error) {
	headReader, err := client.GetState(ctx, IdHead)
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

	log.Printf("(computed client-side) weak subjectivity epoch = %d", epoch)
	return epoch, nil
}

// for older endpoints or clients that do not support the weak_subjectivity api method (only prysm at release time)
// we gather the necessary data for a checkpoint sync by:
// - inspecting the remote server's head state and computing the weak subjectivity epoch locally
// - requesting the state at the first slot of the epoch
// - using hash_tree_root(state.latest_block_header) to compute the block the state integrates
// - requesting that block by its root
func downloadBackwardsCompatible(ctx context.Context, client *Client) (*OriginData, error) {
	log.Print("falling back to generic checkpoint derivation, weak_subjectivity API not supported by server")
	epoch, err := getWeakSubjectivityEpochFromHead(ctx, client)
	if err != nil {
		return nil, errors.Wrap(err, "error computing weak subjectivity epoch via head state inspection")
	}

	// use first slot of the epoch for the state slot
	slot, err := slots.EpochStart(epoch)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error computing first slot of epoch=%d", epoch))
	}

	log.Printf("requesting checkpoint state at slot %d", slot)
	// get the state at the first slot of the epoch
	sReader, err := client.GetState(ctx, IdFromSlot(slot))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to request state by slot from api, slot=%d", slot))
	}
	stateBytes, err := io.ReadAll(sReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body for get checkpoint state api call")
	}

	// ConfigFork is used to unmarshal the BeaconState so we can read the block root in latest_block_header
	cf, err := sniff.ConfigForkForState(stateBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error detecting chain config for beacon state")
	}
	log.Printf("detected supported config in checkpoint state, name=%s, fork=%s", cf.ConfigName.String(), cf.Fork)

	st, err := sniff.BeaconStateForConfigFork(stateBytes, cf)
	if err != nil {
		return nil, errors.Wrap(err, "error using detected config fork to unmarshal state bytes")
	}
	//blockRoot := blockRootFromState(st)

	// compute state and block roots
	stateRoot, err := st.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error computing hash_tree_root of state")
	}

	header := st.LatestBlockHeader()
	header.StateRoot = stateRoot[:]
	computedBlockRoot, err := header.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "error while computing block root using state data")
	}

	blockBytes, err := client.GetBlock(IdFromRoot(computedBlockRoot))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error requesting block by slot = %d", slot))
	}
	block, err := sniff.BlockForConfigFork(blockBytes, cf)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal block to a supported type using the detected fork schedule")
	}
	blockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "error computing hash_tree_root for block obtained via root")
	}

	log.Printf("BeaconState slot=%d, Block slot=%d", st.Slot(), block.Block().Slot())
	log.Printf("BeaconState htr=%#xd, Block state_root=%#x", stateRoot, block.Block().StateRoot())
	log.Printf("BeaconBlock root computed from state=%#x, Block HTR=%#x", computedBlockRoot, blockRoot)

	return &OriginData{
		wsd: &WeakSubjectivityData{
			BlockRoot: blockRoot,
			StateRoot: stateRoot,
			Epoch:     epoch,
		},
		st: st,
		sb: stateBytes,
		b:  block,
		bb: blockBytes,
		cf: cf,
	}, nil
}

// DownloadOriginData attempts to use the proposed weak_subjectivity beacon node api
// to obtain the weak_subjectivity metadata (epoch, block_root, state_root) needed to sync
// a beacon node from the canonical weak subjectivity checkpoint. As this is a proposed API
// that will only be supported by prysm at first, in the event of a 404 we fallback to using a
// different technique where we first download the head state which can be used to compute the
// weak subjectivity epoch on the client side.
func DownloadOriginData(ctx context.Context, client *Client) (*OriginData, error) {
	ws, err := client.GetWeakSubjectivity()
	if err != nil {
		// a 404 is expected if querying an endpoint that doesn't support the weak subjectivity checkpoint api
		if !errors.Is(err, ErrNotFound) {
			return nil, errors.Wrap(err, "unexpected API response for prysm-only weak subjectivity checkpoint API")
		}
		// ok so it's a 404, use the head state method
		return downloadBackwardsCompatible(ctx, client)
	}
	log.Printf("server weak subjectivity checkpoint response - epoch=%d, block_root=%#x, state_root=%#x", ws.Epoch, ws.BlockRoot, ws.StateRoot)

	// use first slot of the epoch for the block slot
	slot, err := slots.EpochStart(ws.Epoch)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Error computing first slot of epoch=%d", ws.Epoch))
	}
	log.Printf("requesting checkpoint state at slot %d", slot)

	sReader, err := client.GetState(ctx, IdFromSlot(slot))
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
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to compute htr for state at slot=%d", slot))
	}

	blockRoot, err := state.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "error computing hash_tree_root of latest_block_header")
	}
	blockBytes, err := client.GetBlock(IdFromRoot(ws.BlockRoot))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error requesting block by slot = %d", slot))
	}
	block, err := sniff.BlockForConfigFork(blockBytes, cf)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal block to a supported type using the detected fork schedule")
	}
	realBlockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "error computing hash_tree_root of retrieved block")
	}
	log.Printf("BeaconState slot=%d, Block slot=%d", state.Slot(), block.Block().Slot())
	log.Printf("BeaconState htr=%#xd, Block state_root=%#x", stateRoot, block.Block().StateRoot())
	log.Printf("BeaconState latest_block_header htr=%#xd, block htr=%#x", blockRoot, realBlockRoot)
	return &OriginData{
		wsd: ws,
		st:  state,
		b:   block,
		sb:  stateBytes,
		bb:  blockBytes,
		cf:  cf,
	}, nil
}
