package core

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func getEmptyBlock(slot primitives.Slot) (interfaces.SignedBeaconBlock, error) {
	var sBlk interfaces.SignedBeaconBlock
	var err error
	switch {
	case slots.ToEpoch(slot) < params.BeaconConfig().AltairForkEpoch:
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}})
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize block for proposal")
		}
	case slots.ToEpoch(slot) < params.BeaconConfig().BellatrixForkEpoch:
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{Body: &ethpb.BeaconBlockBodyAltair{}}})
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize block for proposal")
		}
	case slots.ToEpoch(slot) < params.BeaconConfig().CapellaForkEpoch:
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockBellatrix{Block: &ethpb.BeaconBlockBellatrix{Body: &ethpb.BeaconBlockBodyBellatrix{}}})
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize block for proposal")
		}
	case slots.ToEpoch(slot) < params.BeaconConfig().DenebForkEpoch:
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockCapella{Block: &ethpb.BeaconBlockCapella{Body: &ethpb.BeaconBlockBodyCapella{}}})
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize block for proposal")
		}
	default:
		sBlk, err = blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlockDeneb{Block: &ethpb.BeaconBlockDeneb{Body: &ethpb.BeaconBlockBodyDeneb{}}})
		if err != nil {
			return nil, errors.Wrap(err, "could not initialize block for proposal")
		}
	}
	return sBlk, err
}
