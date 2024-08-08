package operations

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	common "github.com/prysmaticlabs/prysm/v5/testing/spectest/shared/common/operations"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func blockWithConsolidation(ssz []byte) (interfaces.SignedBeaconBlock, error) {
	cr := &enginev1.ConsolidationRequest{}
	if err := cr.UnmarshalSSZ(ssz); err != nil {
		return nil, err
	}
	b := util.NewBeaconBlockElectra()
	b.Block.Body = &ethpb.BeaconBlockBodyElectra{ExecutionPayload: &enginev1.ExecutionPayloadElectra{ConsolidationRequests: []*enginev1.ConsolidationRequest{cr}}}
	return blocks.NewSignedBeaconBlock(b)
}

func RunConsolidationTest(t *testing.T, config string) {
	common.RunConsolidationTest(t, config, version.String(version.Electra), blockWithConsolidation, sszToState)
}
