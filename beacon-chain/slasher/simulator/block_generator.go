package simulator

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/rand"
)

func generateBlockHeadersForSlot(simParams *Parameters, slot types.Slot) []*ethpb.BeaconBlockHeader {
	blocks := make([]*ethpb.BeaconBlockHeader, 1)
	proposer := rand.NewGenerator().Uint64() % simParams.NumValidators
	blocks[0] = &ethpb.BeaconBlockHeader{
		Slot:          slot,
		ProposerIndex: types.ValidatorIndex(proposer),
		ParentRoot:    bytesutil.PadTo([]byte{}, 32),
		StateRoot:     bytesutil.PadTo([]byte{}, 32),
		BodyRoot:      bytesutil.PadTo([]byte("good block"), 32),
	}
	if rand.NewGenerator().Float64() < simParams.ProposerSlashingProbab {
		log.WithField("proposerIndex", proposer).Infof("Slashable block made")
		blocks = append(blocks, &ethpb.BeaconBlockHeader{
			Slot:          slot,
			ProposerIndex: types.ValidatorIndex(proposer),
			ParentRoot:    bytesutil.PadTo([]byte{}, 32),
			StateRoot:     bytesutil.PadTo([]byte{}, 32),
			BodyRoot:      bytesutil.PadTo([]byte("bad block"), 32),
		})
	}
	return blocks
}
