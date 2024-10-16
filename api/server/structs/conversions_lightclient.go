package structs

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	v1 "github.com/prysmaticlabs/prysm/v5/proto/eth/v1"
	v2 "github.com/prysmaticlabs/prysm/v5/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v5/proto/migration"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
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

func branchToJSON[S [][32]byte](branchBytes S) []string {
	if branchBytes == nil {
		return nil
	}
	branch := make([]string, len(branchBytes))
	for i, root := range branchBytes {
		branch[i] = hexutil.Encode(root[:])
	}
	return branch
}

func syncAggregateToJSON(input *v1.SyncAggregate) *SyncAggregate {
	return &SyncAggregate{
		SyncCommitteeBits:      hexutil.Encode(input.SyncCommitteeBits),
		SyncCommitteeSignature: hexutil.Encode(input.SyncCommitteeSignature),
	}
}

func lightClientHeaderContainerToJSON(header interfaces.LightClientHeader) (json.RawMessage, error) {
	// In the case that a finalizedHeader is nil.
	if header == nil {
		return nil, nil
	}

	var result any

	switch v := header.Version(); v {
	case version.Altair:
		result = &LightClientHeader{Beacon: BeaconBlockHeaderFromConsensus(header.Beacon())}
	case version.Capella:
		exInterface, err := header.Execution()
		if err != nil {
			return nil, err
		}
		ex, ok := exInterface.Proto().(*enginev1.ExecutionPayloadHeaderCapella)
		if !ok {
			return nil, fmt.Errorf("execution data is not %T", &enginev1.ExecutionPayloadHeaderCapella{})
		}
		execution, err := ExecutionPayloadHeaderCapellaFromConsensus(ex)
		if err != nil {
			return nil, err
		}
		executionBranch, err := header.ExecutionBranch()
		if err != nil {
			return nil, err
		}
		result = &LightClientHeaderCapella{
			Beacon:          BeaconBlockHeaderFromConsensus(header.Beacon()),
			Execution:       execution,
			ExecutionBranch: branchToJSON(executionBranch[:]),
		}
	case version.Deneb:
		exInterface, err := header.Execution()
		if err != nil {
			return nil, err
		}
		ex, ok := exInterface.Proto().(*enginev1.ExecutionPayloadHeaderDeneb)
		if !ok {
			return nil, fmt.Errorf("execution data is not %T", &enginev1.ExecutionPayloadHeaderDeneb{})
		}
		execution, err := ExecutionPayloadHeaderDenebFromConsensus(ex)
		if err != nil {
			return nil, err
		}
		executionBranch, err := header.ExecutionBranch()
		if err != nil {
			return nil, err
		}
		result = &LightClientHeaderDeneb{
			Beacon:          BeaconBlockHeaderFromConsensus(header.Beacon()),
			Execution:       execution,
			ExecutionBranch: branchToJSON(executionBranch[:]),
		}
	default:
		return nil, fmt.Errorf("unsupported header version %s", version.String(v))
	}

	return json.Marshal(result)
}
