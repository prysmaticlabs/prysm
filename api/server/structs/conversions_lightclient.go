package structs

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/proto/migration"
)

func LightClientUpdateFromConsensus(update *v2.LightClientUpdate) (*LightClientUpdate, error) {
	attestedHeader, err := lightClientHeaderContainerToJSON(update.AttestedHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal attested light client header")
	}
	finalizedHeader, err := lightClientHeaderContainerToJSON(update.FinalizedHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal finalized light client header")
	}

	return &LightClientUpdate{
		AttestedHeader:          attestedHeader,
		NextSyncCommittee:       SyncCommitteeFromConsensus(migration.V2SyncCommitteeToV1Alpha1(update.NextSyncCommittee)),
		NextSyncCommitteeBranch: branchToJSON(update.NextSyncCommitteeBranch),
		FinalizedHeader:         finalizedHeader,
		FinalityBranch:          branchToJSON(update.FinalityBranch),
		SyncAggregate:           syncAggregateToJSON(update.SyncAggregate),
		SignatureSlot:           strconv.FormatUint(uint64(update.SignatureSlot), 10),
	}, nil
}

func LightClientFinalityUpdateFromConsensus(update *v2.LightClientFinalityUpdate) (*LightClientFinalityUpdate, error) {
	attestedHeader, err := lightClientHeaderContainerToJSON(update.AttestedHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal attested light client header")
	}
	finalizedHeader, err := lightClientHeaderContainerToJSON(update.FinalizedHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal finalized light client header")
	}

	return &LightClientFinalityUpdate{
		AttestedHeader:  attestedHeader,
		FinalizedHeader: finalizedHeader,
		FinalityBranch:  branchToJSON(update.FinalityBranch),
		SyncAggregate:   syncAggregateToJSON(update.SyncAggregate),
		SignatureSlot:   strconv.FormatUint(uint64(update.SignatureSlot), 10),
	}, nil
}

func LightClientOptimisticUpdateFromConsensus(update *v2.LightClientOptimisticUpdate) (*LightClientOptimisticUpdate, error) {
	attestedHeader, err := lightClientHeaderContainerToJSON(update.AttestedHeader)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal attested light client header")
	}

	return &LightClientOptimisticUpdate{
		AttestedHeader: attestedHeader,
		SyncAggregate:  syncAggregateToJSON(update.SyncAggregate),
		SignatureSlot:  strconv.FormatUint(uint64(update.SignatureSlot), 10),
	}, nil
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

func lightClientHeaderContainerToJSON(container *v2.LightClientHeaderContainer) (json.RawMessage, error) {
	// In the case that a finalizedHeader is nil.
	if container == nil {
		return nil, nil
	}

	beacon, err := container.GetBeacon()
	if err != nil {
		return nil, errors.Wrap(err, "could not get beacon block header")
	}

	var header any

	switch t := (container.Header).(type) {
	case *v2.LightClientHeaderContainer_HeaderAltair:
		header = &LightClientHeader{Beacon: BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(beacon))}
	case *v2.LightClientHeaderContainer_HeaderCapella:
		execution, err := ExecutionPayloadHeaderCapellaFromConsensus(t.HeaderCapella.Execution)
		if err != nil {
			return nil, err
		}
		header = &LightClientHeaderCapella{
			Beacon:          BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(beacon)),
			Execution:       execution,
			ExecutionBranch: branchToJSON(t.HeaderCapella.ExecutionBranch),
		}
	case *v2.LightClientHeaderContainer_HeaderDeneb:
		execution, err := ExecutionPayloadHeaderDenebFromConsensus(t.HeaderDeneb.Execution)
		if err != nil {
			return nil, err
		}
		header = &LightClientHeaderDeneb{
			Beacon:          BeaconBlockHeaderFromConsensus(migration.V1HeaderToV1Alpha1(beacon)),
			Execution:       execution,
			ExecutionBranch: branchToJSON(t.HeaderDeneb.ExecutionBranch),
		}
	default:
		return nil, fmt.Errorf("unsupported header type %T", t)
	}

	return json.Marshal(header)
}
