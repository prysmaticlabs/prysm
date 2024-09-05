package structs

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	v2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/proto/migration"
)

func LightClientBootstrapFromConsensus(bootstrap *v2.LightClientBootstrap) *LightClientBootstrap {
	branch := make([]string, len(bootstrap.CurrentSyncCommitteeBranch))
	for i, item := range bootstrap.CurrentSyncCommitteeBranch {
		branch[i] = hexutil.Encode(item)
	}

	return &LightClientBootstrap{
		Header:                     &LightClientHeader{Beacon: BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(bootstrap.Header.Beacon))},
		CurrentSyncCommittee:       SyncCommitteeFromConsensus(migration.V2SyncCommitteeToV1Alpha1(bootstrap.CurrentSyncCommittee)),
		CurrentSyncCommitteeBranch: branch,
	}
}
