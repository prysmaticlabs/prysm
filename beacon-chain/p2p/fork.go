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

type enrForkID struct {
	currentForkDigest [4]byte
	nextForkVersion   [4]byte
	nextForkEpoch     uint64
}

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
	// TODO(): Update on any required hard fork.
	nextForkVersion := fork.CurrentVersion
	nextForkEpoch := params.BeaconConfig().FarFutureEpoch
	enrForkID := &enrForkID{
		currentForkDigest: digest,
		nextForkVersion:   bytesutil.ToBytes4(nextForkVersion),
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
