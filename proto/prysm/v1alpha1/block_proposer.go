package eth

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/prysm/v1alpha1/types"
)

// ExtendedBeaconNodeValidatorServer combines multiple validator interfaces.
type ExtendedBeaconNodeValidatorServer interface {
	BeaconNodeValidatorServer
	BlockProposer
}

// BlockProposer includes methods related to proposing a beacon block.
type BlockProposer interface {
	ProposeGenericBeaconBlock(ctx context.Context, req *GenericSignedBeaconBlock, validation types.BroadcastValidation) (*ProposeResponse, error)
}
