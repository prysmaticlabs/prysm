package beacon

import (
	"context"
	"fmt"
	"path"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/encoding/ssz/detect"
	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	log "github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

// OriginData represents the BeaconState and SignedBeaconBlock necessary to start an empty Beacon Node
// using Checkpoint Sync.
type OriginData struct {
	wsd *WeakSubjectivityData
	sb  []byte
	bb  []byte
	st  state.BeaconState
	b   block.SignedBeaconBlock
	cf  *detect.VersionedUnmarshaler
}

// CheckpointString returns the standard string representation of a Checkpoint for the block root and epoch for the
// SignedBeaconBlock value found by DownloadOriginData.
// The format is a a hex-encoded block root, followed by the epoch of the block, separated by a colon. For example:
// "0x1c35540cac127315fabb6bf29181f2ae0de1a3fc909d2e76ba771e61312cc49a:74888"
func (od *OriginData) CheckpointString() string {
	return fmt.Sprintf("%#x:%d", od.wsd.BlockRoot, od.wsd.Epoch)
}

// SaveBlock saves the downloaded block to a unique file in the given path.
// For readability and collision avoidance, the file name includes: type, config name, slot and root
func (od *OriginData) SaveBlock(dir string) (string, error) {
	blockPath := path.Join(dir, fname("state", od.cf, od.st.Slot(), od.wsd.BlockRoot))
	return blockPath, file.WriteFile(blockPath, od.sb)
}

// SaveState saves the downloaded state to a unique file in the given path.
// For readability and collision avoidance, the file name includes: type, config name, slot and root
func (od *OriginData) SaveState(dir string) (string, error) {
	statePath := path.Join(dir, fname("state", od.cf, od.st.Slot(), od.wsd.StateRoot))
	return statePath, file.WriteFile(statePath, od.sb)
}

// StateBytes returns the ssz-encoded bytes of the downloaded BeaconState value.
func (od *OriginData) StateBytes() []byte {
	return od.sb
}

// BlockBytes returns the ssz-encoded bytes of the downloaded SignedBeaconBlock value.
func (od *OriginData) BlockBytes() []byte {
	return od.bb
}

func fname(prefix string, cf *detect.VersionedUnmarshaler, slot types.Slot, root [32]byte) string {
	return fmt.Sprintf("%s_%s_%s_%d-%#x.ssz", prefix, cf.Config.ConfigName, version.String(cf.Fork), slot, root)
}

// this method downloads the head state, which can be used to find the correct chain config
// and use prysm's helper methods to compute the latest weak subjectivity epoch.
func getWeakSubjectivityEpochFromHead(ctx context.Context, client *Client) (types.Epoch, error) {
	headBytes, err := client.GetState(ctx, IdHead)
	if err != nil {
		return 0, err
	}
	cf, err := detect.FromState(headBytes)
	if err != nil {
		return 0, errors.Wrap(err, "error detecting chain config for beacon state")
	}
	log.Printf("detected supported config in remote head state, name=%s, fork=%s", cf.Config.ConfigName, version.String(cf.Fork))
	headState, err := cf.UnmarshalBeaconState(headBytes)
	if err != nil {
		return 0, errors.Wrap(err, "error unmarshaling state to correct version")
	}

	epoch, err := helpers.LatestWeakSubjectivityEpoch(ctx, headState, cf.Config)
	if err != nil {
		return 0, errors.Wrap(err, "error computing the weak subjectivity epoch from head state")
	}

	log.Printf("(computed client-side) weak subjectivity epoch = %d", epoch)
	return epoch, nil
}

const (
	prysmMinimumVersion     = "v2.0.7"
	prysmImplementationName = "Prysm"
)

// ErrUnsupportedPrysmCheckpointVersion indicates remote beacon node can't be used for checkpoint retrieval.
var ErrUnsupportedPrysmCheckpointVersion = errors.New("node does not meet minimum version requirements for checkpoint retrieval")

// for older endpoints or clients that do not support the weak_subjectivity api method
// we gather the necessary data for a checkpoint sync by:
// - inspecting the remote server's head state and computing the weak subjectivity epoch locally
// - requesting the state at the first slot of the epoch
// - using hash_tree_root(state.latest_block_header) to compute the block the state integrates
// - requesting that block by its root
func downloadBackwardsCompatible(ctx context.Context, client *Client) (*OriginData, error) {
	log.Print("falling back to generic checkpoint derivation, weak_subjectivity API not supported by server")
	nv, err := client.GetNodeVersion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to proceed with fallback method without confirming node version")
	}
	if nv.implementation == prysmImplementationName && semver.Compare(nv.semver, prysmMinimumVersion) < 0 {
		return nil, errors.Wrapf(ErrUnsupportedPrysmCheckpointVersion, "%s < minimum (%s)", nv.semver, prysmMinimumVersion)
	}
	epoch, err := getWeakSubjectivityEpochFromHead(ctx, client)
	if err != nil {
		return nil, errors.Wrap(err, "error computing weak subjectivity epoch via head state inspection")
	}

	// use first slot of the epoch for the state slot
	slot, err := slots.EpochStart(epoch)
	if err != nil {
		return nil, errors.Wrapf(err, "error computing first slot of epoch=%d", epoch)
	}

	log.Printf("requesting checkpoint state at slot %d", slot)
	// get the state at the first slot of the epoch
	stateBytes, err := client.GetState(ctx, IdFromSlot(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to request state by slot from api, slot=%d", slot)
	}

	// ConfigFork is used to unmarshal the BeaconState so we can read the block root in latest_block_header
	cf, err := detect.FromState(stateBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error detecting chain config for beacon state")
	}
	log.Printf("detected supported config in checkpoint state, name=%s, fork=%s", cf.Config.ConfigName, version.String(cf.Fork))

	st, err := cf.UnmarshalBeaconState(stateBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error using detected config fork to unmarshal state bytes")
	}

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

	blockBytes, err := client.GetBlock(ctx, IdFromRoot(computedBlockRoot))
	if err != nil {
		return nil, errors.Wrapf(err, "error requesting block by root = %d", computedBlockRoot)
	}
	block, err := cf.UnmarshalBeaconBlock(blockBytes)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal block to a supported type using the detected fork schedule")
	}
	blockRoot, err := block.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "error computing hash_tree_root for block obtained via root")
	}

	log.Printf("BeaconState slot=%d, Block slot=%d", st.Slot(), block.Block().Slot())
	log.Printf("BeaconState htr=%#xd, Block state_root=%#x", stateRoot, block.Block().StateRoot())
	log.Printf("BeaconBlock root computed from state=%#x, Block htr=%#x", computedBlockRoot, blockRoot)

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
	ws, err := client.GetWeakSubjectivity(ctx)
	if err != nil {
		// a 404/405 is expected if querying an endpoint that doesn't support the weak subjectivity checkpoint api
		if !errors.Is(err, ErrNotOK) {
			return nil, errors.Wrap(err, "unexpected API response for prysm-only weak subjectivity checkpoint API")
		}
		// fall back to vanilla Beacon Node API method
		return downloadBackwardsCompatible(ctx, client)
	}
	log.Printf("server weak subjectivity checkpoint response - epoch=%d, block_root=%#x, state_root=%#x", ws.Epoch, ws.BlockRoot, ws.StateRoot)

	// use first slot of the epoch for the block slot
	slot, err := slots.EpochStart(ws.Epoch)
	if err != nil {
		return nil, errors.Wrapf(err, "error computing first slot of epoch=%d", ws.Epoch)
	}
	log.Printf("requesting checkpoint state at slot %d", slot)

	stateBytes, err := client.GetState(ctx, IdFromSlot(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to request state by slot from api, slot=%d", slot)
	}
	cf, err := detect.FromState(stateBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error detecting chain config for beacon state")
	}
	log.Printf("detected supported config in checkpoint state, name=%s, fork=%s", cf.Config.ConfigName, version.String(cf.Fork))

	state, err := cf.UnmarshalBeaconState(stateBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error using detected config fork to unmarshal state bytes")
	}
	stateRoot, err := state.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compute htr for state at slot=%d", slot)
	}

	blockRoot, err := state.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "error computing hash_tree_root of latest_block_header")
	}
	blockBytes, err := client.GetBlock(ctx, IdFromRoot(ws.BlockRoot))
	if err != nil {
		return nil, errors.Wrapf(err, "error requesting block by slot = %d", slot)
	}
	block, err := cf.UnmarshalBeaconBlock(blockBytes)
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
