package structs

import (
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/proto/migration"
)

func LightClientUpdateFromConsensus(update *v2.LightClientUpdate) *LightClientUpdate {
	return &LightClientUpdate{
		AttestedHeader:          &LightClientHeader{Beacon: BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(update.AttestedHeader.Beacon))},
		NextSyncCommittee:       SyncCommitteeFromConsensus(migration.V2SyncCommitteeToV1Alpha1(update.NextSyncCommittee)),
		NextSyncCommitteeBranch: branchToJSON(update.NextSyncCommitteeBranch),
		FinalizedHeader:         &LightClientHeader{Beacon: BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(update.FinalizedHeader.Beacon))},
		FinalityBranch:          branchToJSON(update.FinalityBranch),
		SyncAggregate:           syncAggregateToJSON(update.SyncAggregate),
		SignatureSlot:           strconv.FormatUint(uint64(update.SignatureSlot), 10),
	}
}

func LightClientFinalityUpdateFromConsensus(update *v2.LightClientFinalityUpdate) *LightClientFinalityUpdate {
	return &LightClientFinalityUpdate{
		AttestedHeader:  BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(update.AttestedHeader.Beacon)),
		FinalizedHeader: BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(update.FinalizedHeader.Beacon)),
		FinalityBranch:  branchToJSON(update.FinalityBranch),
		SyncAggregate:   syncAggregateToJSON(update.SyncAggregate),
		SignatureSlot:   strconv.FormatUint(uint64(update.SignatureSlot), 10),
	}
}

func LightClientOptimisticUpdateFromConsensus(update *v2.LightClientOptimisticUpdate) *LightClientOptimisticUpdate {
	return &LightClientOptimisticUpdate{
		AttestedHeader: BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(update.AttestedHeader.Beacon)),
		SyncAggregate:  syncAggregateToJSON(update.SyncAggregate),
		SignatureSlot:  strconv.FormatUint(uint64(update.SignatureSlot), 10),
	}
}

func branchToJSON(branchBytes [][]byte) []string {
	if branchBytes == nil {
		return nil
	}
	branch := make([]string, len(branchBytes))
	for i, root := range branchBytes {
		branch[i] = hexutil.Encode(root)
	}
	return branch
}

func syncAggregateToJSON(input *v1.SyncAggregate) *SyncAggregate {
	return &SyncAggregate{
		SyncCommitteeBits:      hexutil.Encode(input.SyncCommitteeBits),
		SyncCommitteeSignature: hexutil.Encode(input.SyncCommitteeSignature),
	}
}
