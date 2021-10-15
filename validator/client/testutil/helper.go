package testutil

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ActiveKey represents a public key whose status is ACTIVE.
var ActiveKey = bytesutil.ToBytes48([]byte("active"))

// GenerateMultipleValidatorStatusResponse prepares a response from the passed in keys.
func GenerateMultipleValidatorStatusResponse(pubkeys [][]byte) *ethpb.MultipleValidatorStatusResponse {
	resp := &ethpb.MultipleValidatorStatusResponse{
		PublicKeys: make([][]byte, len(pubkeys)),
		Statuses:   make([]*ethpb.ValidatorStatusResponse, len(pubkeys)),
		Indices:    make([]types.ValidatorIndex, len(pubkeys)),
	}
	for i, key := range pubkeys {
		resp.PublicKeys[i] = key
		resp.Statuses[i] = &ethpb.ValidatorStatusResponse{
			Status: ethpb.ValidatorStatus_UNKNOWN_STATUS,
		}
		resp.Indices[i] = types.ValidatorIndex(i)
	}

	return resp
}
