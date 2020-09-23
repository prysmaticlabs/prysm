// Package p2putils contains useful helpers for eth2 fork-related functionality.
package p2putils

import (
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// CreateForkDigest creates a fork digest from a genesis time and genesis
// validators root, utilizing the current slot to determine
// the active fork version in the node.
func CreateForkDigest(
	genesisTime time.Time,
	genesisValidatorsRoot []byte,
) ([4]byte, error) {
	if genesisTime.IsZero() {
		return [4]byte{}, errors.New("genesis time is not set")
	}
	if len(genesisValidatorsRoot) == 0 {
		return [4]byte{}, errors.New("genesis validators root is not set")
	}
	currentSlot := helpers.SlotsSince(genesisTime)
	currentEpoch := helpers.SlotToEpoch(currentSlot)

	forkData, err := Fork(currentEpoch)
	if err != nil {
		return [4]byte{}, err
	}

	digest, err := helpers.ComputeForkDigest(forkData.CurrentVersion, genesisValidatorsRoot)
	if err != nil {
		return [4]byte{}, err
	}
	return digest, nil
}

// Fork given a target epoch,
// returns the active fork version during this epoch.
func Fork(
	targetEpoch uint64,
) (*pb.Fork, error) {
	// We retrieve a list of scheduled forks by epoch.
	// We loop through the keys in this map to determine the current
	// fork version based on the requested epoch.
	scheduledForks := params.BeaconConfig().ForkVersionSchedule
	f := &pb.Fork{
		Epoch: 0,
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		CurrentVersion: params.BeaconConfig().GenesisForkVersion,
	}
	for _, fork := range scheduledForks {
		if fork.Epoch <= targetEpoch && fork.Epoch > f.Epoch {
			f = fork
		}
	}
	return f, nil
}
