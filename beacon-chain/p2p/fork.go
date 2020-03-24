package p2p

import (
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ENR key used for eth2-related fork data.
const eth2EnrKey = "eth2"

// EnrForkID represents a special value ssz-encoded into
// the local node's ENR for discovery purposes. Peers should
// only connect if their enrForkID matches.
type EnrForkID struct {
	CurrentForkDigest [4]byte
	NextForkVersion   [4]byte
	NextForkEpoch     uint64
}

// Adds a fork entry as an ENR record under the eth2EnrKey for
// the local node. The fork entry is an ssz-encoded enrForkID type
// which takes into account the current fork version from the current
// epoch to create a fork digest, the next fork version,
// and the next fork epoch.
func addForkEntry(
	node *enode.LocalNode,
	genesisTime time.Time,
	genesisValidatorsRoot []byte,
) (*enode.LocalNode, error) {
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
		return nil, err
	}
	nextForkEpoch := params.BeaconConfig().NextForkEpoch
	enrForkID := &EnrForkID{
		CurrentForkDigest: digest,
		NextForkVersion:   bytesutil.ToBytes4(params.BeaconConfig().NextForkVersion),
		NextForkEpoch:     nextForkEpoch,
	}
	enc, err := ssz.Marshal(enrForkID)
	if err != nil {
		return nil, err
	}
	forkEntry := enr.WithEntry(eth2EnrKey, enc)
	node.Set(forkEntry)
	return node, nil
}

// Retrieves an enrForkID from an ENR record by key lookup
// under the eth2EnrKey.
func retrieveForkEntry(record *enr.Record) (*EnrForkID, error) {
	sszEncodedForkEntry := make([]byte, 16)
	entry := enr.WithEntry(eth2EnrKey, &sszEncodedForkEntry)
	err := record.Load(entry)
	if err != nil {
		return nil, err
	}
	forkEntry := &EnrForkID{}
	if err := ssz.Unmarshal(sszEncodedForkEntry, forkEntry); err != nil {
		return nil, err
	}
	return forkEntry, nil
}
