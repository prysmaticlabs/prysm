package validator

import (
	"github.com/prysmaticlabs/prysm/v3/config/params"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// constructBeaconBlockCapellaFromBellatrix returns a wrapped Cappella Beacon block starting from the
// given blockData object
func constructBeaconBlockCapellaFromBlockData(blkData *blockData) *ethpb.BeaconBlockCapella {
	return &ethpb.BeaconBlockCapella{
		Slot:          blkData.Slot,
		ProposerIndex: blkData.ProposerIdx,
		ParentRoot:    blkData.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		Body: &ethpb.BeaconBlockBodyCapella{
			RandaoReveal:      blkData.RandaoReveal,
			Eth1Data:          blkData.Eth1Data,
			Graffiti:          blkData.Graffiti[:],
			ProposerSlashings: blkData.ProposerSlashings,
			AttesterSlashings: blkData.AttesterSlashings,
			Attestations:      blkData.Attestations,
			Deposits:          blkData.Deposits,
			VoluntaryExits:    blkData.VoluntaryExits,
			SyncAggregate:     blkData.SyncAggregate,
			ExecutionPayload: &enginev1.ExecutionPayloadCapella{
				ParentHash:    blkData.ExecutionPayload.ParentHash,
				FeeRecipient:  blkData.ExecutionPayload.FeeRecipient,
				StateRoot:     blkData.ExecutionPayload.StateRoot,
				ReceiptsRoot:  blkData.ExecutionPayload.ReceiptsRoot,
				LogsBloom:     blkData.ExecutionPayload.LogsBloom,
				PrevRandao:    blkData.ExecutionPayload.PrevRandao,
				BlockNumber:   blkData.ExecutionPayload.BlockNumber,
				GasLimit:      blkData.ExecutionPayload.GasLimit,
				GasUsed:       blkData.ExecutionPayload.GasUsed,
				Timestamp:     blkData.ExecutionPayload.Timestamp,
				ExtraData:     blkData.ExecutionPayload.ExtraData,
				BaseFeePerGas: blkData.ExecutionPayload.BaseFeePerGas,
				BlockHash:     blkData.ExecutionPayload.BlockHash,
				Transactions:  blkData.ExecutionPayload.Transactions,
				Withdrawals:   blkData.Withdrawals,
			},
			BlsToExecutionChanges: blkData.BlsToExecutionChanges,
		},
	}
}
