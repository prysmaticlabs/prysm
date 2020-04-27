package p2putils

import (
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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

// ForkDigest creates a fork digest from a target epoch and genesis
// validators root, returns the active fork version in the node.
func ForkDigest(
	targetEpoch uint64,
	genesisValidatorsRoot []byte,
) ([4]byte, error) {
	if len(genesisValidatorsRoot) == 0 {
		return [4]byte{}, errors.New("genesis validators root is not set")
	}

	// We retrieve a list of scheduled forks by epoch.
	// We loop through the keys in this map to determine the current
	// fork version based on the requested epoch.
	retrievedForkVersion := params.BeaconConfig().GenesisForkVersion
	scheduledForks := params.BeaconConfig().ForkVersionSchedule
	for epoch, forkVersion := range scheduledForks {
		if epoch <= targetEpoch {
			retrievedForkVersion = forkVersion
		}
	}

	digest, err := helpers.ComputeForkDigest(retrievedForkVersion, genesisValidatorsRoot)
	if err != nil {
		return [4]byte{}, err
	}
	return digest, nil
}
