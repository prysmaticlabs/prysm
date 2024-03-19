package migration

import (
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpbv2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// V1Alpha1SyncCommitteeToV2 converts a v1alpha1 SyncCommittee object to its v2 equivalent.
func V1Alpha1SyncCommitteeToV2(alphaCommittee *ethpbalpha.SyncCommittee) *ethpbv2.SyncCommittee {
	if alphaCommittee == nil {
		return nil
	}

	result := &ethpbv2.SyncCommittee{
		Pubkeys:         bytesutil.SafeCopy2dBytes(alphaCommittee.Pubkeys),
		AggregatePubkey: bytesutil.SafeCopyBytes(alphaCommittee.AggregatePubkey),
	}
	return result
}

func V2SyncCommitteeToV1Alpha1(committee *ethpbv2.SyncCommittee) *ethpbalpha.SyncCommittee {
	if committee == nil {
		return nil
	}

	result := &ethpbalpha.SyncCommittee{
		Pubkeys:         bytesutil.SafeCopy2dBytes(committee.Pubkeys),
		AggregatePubkey: bytesutil.SafeCopyBytes(committee.AggregatePubkey),
	}
	return result
}
