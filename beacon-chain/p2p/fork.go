package p2p

import (
	"errors"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

const eth2EnrKey = "eth2"

// enrForkID represents a special value ssz-encoded into
// the local node's ENR for discovery purposes. Peers should
// only connect if their enrForkID matches.
type enrForkID struct {
	currentForkDigest [4]byte
	nextForkVersion   [4]byte
	nextForkEpoch     uint64
}

// Adds a fork entry as an ENR record under the eth2EnrKey for
// the local node. The fork entry is an ssz-encoded enrForkID type
// which takes into account the current fork version from the beacon
// state to create a fork digest, the next fork version,
// and the next fork epoch.
func addForkEntry(node *enode.LocalNode, st *stateTrie.BeaconState) (*enode.LocalNode, error) {
	fork := st.Fork()
	if fork == nil {
		return nil, errors.New("nil fork version in state")
	}
	genesisValidatorsRoot := st.GenesisValidatorRoot()
	if genesisValidatorsRoot == nil {
		return nil, errors.New("nil genesis validator root in state")
	}
	digest, err := helpers.ComputeForkDigest(fork.CurrentVersion, genesisValidatorsRoot)
	if err != nil {
		return nil, err
	}
	nextForkEpoch := params.BeaconConfig().NextForkEpoch
	enrForkID := &enrForkID{
		currentForkDigest: digest,
		nextForkVersion:   bytesutil.ToBytes4(params.BeaconConfig().NextForkVersion),
		nextForkEpoch:     nextForkEpoch,
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
func retrieveForkEntry(record *enr.Record) (*enrForkID, error) {
	sszEncodedForkEntry := make([]byte, 16)
	entry := enr.WithEntry(eth2EnrKey, &sszEncodedForkEntry)
	err := record.Load(entry)
	if err != nil {
		return nil, err
	}
	forkEntry := &enrForkID{}
	if err := ssz.Unmarshal(sszEncodedForkEntry, forkEntry); err != nil {
		return nil, err
	}
	return forkEntry, nil
}
