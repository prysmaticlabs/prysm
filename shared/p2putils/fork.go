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

	// We retrieve a list of scheduled forks by epoch.
	// We loop through the keys in this map to determine the current
	// fork version based on the current, time-based epoch number
	// since the genesis time.
	currentForkVersion := params.BeaconConfig().GenesisForkVersion
	scheduledForks := params.BeaconConfig().ForkVersionSchedule
	for epoch, forkVersion := range scheduledForks {
		if epoch <= currentEpoch {
			currentForkVersion = forkVersion
		}
	}

	digest, err := helpers.ComputeForkDigest(currentForkVersion, genesisValidatorsRoot)
	if err != nil {
		return [4]byte{}, err
	}
	return digest, nil
}

// Fork given a target epoch and genesis validators root,
// returns the active fork version in the node.
func Fork(
	targetEpoch uint64,
	genesisValidatorsRoot []byte,
) (*pb.Fork, error) {
	if len(genesisValidatorsRoot) == 0 {
		return &pb.Fork{}, errors.New("genesis validators root is not set")
	}

	// We retrieve a list of scheduled forks by epoch.
	// We loop through the keys in this map to determine the current
	// fork version based on the requested epoch.
	retrievedForkVersion := params.BeaconConfig().GenesisForkVersion
	previousForkVersion := params.BeaconConfig().GenesisForkVersion
	scheduledForks := params.BeaconConfig().ForkVersionSchedule
	forkEpoch := uint64(0)
	for epoch, forkVersion := range scheduledForks {
		if epoch <= targetEpoch {
			previousForkVersion = retrievedForkVersion
			retrievedForkVersion = forkVersion
			forkEpoch = epoch

		}
	}

	return &pb.Fork{
		PreviousVersion: previousForkVersion,
		CurrentVersion:  retrievedForkVersion,
		Epoch:           forkEpoch,
	}, nil
}
