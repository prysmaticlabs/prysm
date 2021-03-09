package simulator

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
)

func generateBlockHeadersForSlot(
	simParams *Parameters, slot types.Slot,
) ([]*ethpb.SignedBeaconBlockHeader, []*ethpb.ProposerSlashing) {
	blocks := make([]*ethpb.SignedBeaconBlockHeader, 0)
	slashings := make([]*ethpb.ProposerSlashing, 0)
	proposer := rand.NewGenerator().Uint64() % simParams.NumValidators
	block := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			Slot:          slot,
			ProposerIndex: types.ValidatorIndex(proposer),
			ParentRoot:    bytesutil.PadTo([]byte{}, 32),
			StateRoot:     bytesutil.PadTo([]byte{}, 32),
			BodyRoot:      bytesutil.PadTo([]byte("good block"), 32),
		},
		Signature: params.BeaconConfig().EmptySignature[:],
	}
	blocks = append(blocks, block)
	if rand.NewGenerator().Float64() < simParams.ProposerSlashingProbab {
		log.WithField("proposerIndex", proposer).Infof("Slashable block made")
		slashableBlock := &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:          slot,
				ProposerIndex: types.ValidatorIndex(proposer),
				ParentRoot:    bytesutil.PadTo([]byte{}, 32),
				StateRoot:     bytesutil.PadTo([]byte{}, 32),
				BodyRoot:      bytesutil.PadTo([]byte("bad block"), 32),
			},
			Signature: params.BeaconConfig().EmptySignature[:],
		}
		blocks = append(blocks, slashableBlock)
		slashings = append(slashings, &ethpb.ProposerSlashing{
			Header_1: block,
			Header_2: slashableBlock,
		})
	}
	return blocks, slashings
}
