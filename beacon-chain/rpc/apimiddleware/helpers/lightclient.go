package helpers

import (
	"errors"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"google.golang.org/protobuf/types/known/timestamppb"

	ethrpc "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	v11 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
)

func bytesFromBigInt(numStr string) ([]byte, error) {
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).SetUint64(num).Bytes(), nil
}

func NewExecutionPayloadHeaderFromJSON(headerJSON *ethrpc.ExecutionPayloadHeaderJson) (*v11.ExecutionPayloadHeader,
	error) {
	header := &v11.ExecutionPayloadHeader{}
	var err error
	if header.ParentHash, err = hexutil.Decode(headerJSON.ParentHash); err != nil {
		return nil, err
	}
	if header.FeeRecipient, err = hexutil.Decode(headerJSON.FeeRecipient); err != nil {
		return nil, err
	}
	if header.StateRoot, err = hexutil.Decode(headerJSON.StateRoot); err != nil {
		return nil, err
	}
	if header.ReceiptsRoot, err = hexutil.Decode(headerJSON.ReceiptsRoot); err != nil {
		return nil, err
	}
	if header.LogsBloom, err = hexutil.Decode(headerJSON.LogsBloom); err != nil {
		return nil, err
	}
	if header.PrevRandao, err = hexutil.Decode(headerJSON.PrevRandao); err != nil {
		return nil, err
	}
	if header.BlockNumber, err = strconv.ParseUint(headerJSON.BlockNumber, 10, 64); err != nil {
		return nil, err
	}
	if header.GasLimit, err = strconv.ParseUint(headerJSON.GasLimit, 10, 64); err != nil {
		return nil, err
	}
	if header.GasUsed, err = strconv.ParseUint(headerJSON.GasUsed, 10, 64); err != nil {
		return nil, err
	}
	if header.Timestamp, err = strconv.ParseUint(headerJSON.TimeStamp, 10, 64); err != nil {
		return nil, err
	}
	if header.ExtraData, err = hexutil.Decode(headerJSON.ExtraData); err != nil {
		return nil, err
	}

	if header.BaseFeePerGas, err = bytesFromBigInt(headerJSON.BaseFeePerGas); err != nil {
		return nil, err
	}
	if len(header.BaseFeePerGas) > 32 {
		return nil, errors.New("base fee per gas is too long")
	} else if len(header.BaseFeePerGas) < 32 {
		padded := make([]byte, 32-len(header.BaseFeePerGas))
		header.BaseFeePerGas = append(padded, header.BaseFeePerGas...)
	}
	for i := 0; i < len(header.BaseFeePerGas)/2; i++ {
		header.BaseFeePerGas[i], header.BaseFeePerGas[len(header.BaseFeePerGas)-1-i] =
			header.BaseFeePerGas[len(header.BaseFeePerGas)-1-i], header.BaseFeePerGas[i]
	}

	if header.BlockHash, err = hexutil.Decode(headerJSON.BlockHash); err != nil {
		return nil, err
	}
	if header.TransactionsRoot, err = hexutil.Decode(headerJSON.TransactionsRoot); err != nil {
		return nil, err
	}
	return header, nil
}

func NewSyncAggregateFromJSON(syncAggregateJSON *ethrpc.SyncAggregateJson) (*ethpbv1.SyncAggregate, error) {
	syncAggregate := &ethpbv1.SyncAggregate{}
	var err error
	if syncAggregate.SyncCommitteeBits, err = hexutil.Decode(syncAggregateJSON.SyncCommitteeBits); err != nil {
		return nil, err
	}
	if syncAggregate.SyncCommitteeSignature, err = hexutil.Decode(syncAggregateJSON.SyncCommitteeSignature); err != nil {
		return nil, err
	}
	return syncAggregate, nil
}

func timeFromJSON(timestamp string) (*time.Time, error) {
	timeInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, err
	}
	t := time.Unix(timeInt, 0)
	return &t, nil
}

func NewGenesisResponse_GenesisFromJSON(genesisJSON *ethrpc.GenesisResponse_GenesisJson) (
	*ethpbv1.GenesisResponse_Genesis, error) {
	genesis := &ethpbv1.GenesisResponse_Genesis{}
	genesisTime, err := timeFromJSON(genesisJSON.GenesisTime)
	if err != nil {
		return nil, err
	}
	genesis.GenesisTime = timestamppb.New(*genesisTime)
	if genesis.GenesisValidatorsRoot, err = hexutil.Decode(genesisJSON.GenesisValidatorsRoot); err != nil {
		return nil, err
	}
	if genesis.GenesisForkVersion, err = hexutil.Decode(genesisJSON.GenesisForkVersion); err != nil {
		return nil, err
	}
	return genesis, nil
}

func headerFromJSON(headerJSON *ethrpc.BeaconBlockHeaderJson) (*ethpbv1.BeaconBlockHeader, error) {
	if headerJSON == nil {
		return nil, nil
	}
	header := &ethpbv1.BeaconBlockHeader{}
	var err error
	slot, err := strconv.ParseUint(headerJSON.Slot, 10, 64)
	if err != nil {
		return nil, err
	}
	header.Slot = types.Slot(slot)
	proposerIndex, err := strconv.ParseUint(headerJSON.ProposerIndex, 10, 64)
	if err != nil {
		return nil, err
	}
	header.ProposerIndex = types.ValidatorIndex(proposerIndex)
	if header.ParentRoot, err = hexutil.Decode(headerJSON.ParentRoot); err != nil {
		return nil, err
	}
	if header.StateRoot, err = hexutil.Decode(headerJSON.StateRoot); err != nil {
		return nil, err
	}
	if header.BodyRoot, err = hexutil.Decode(headerJSON.BodyRoot); err != nil {
		return nil, err
	}
	return header, nil
}

func syncCommitteeFromJSON(syncCommitteeJSON *ethrpc.SyncCommitteeJson) (*ethpbv2.SyncCommittee, error) {
	if syncCommitteeJSON == nil {
		return nil, nil
	}
	syncCommittee := &ethpbv2.SyncCommittee{
		Pubkeys: make([][]byte, len(syncCommitteeJSON.Pubkeys)),
	}
	for i, pubKey := range syncCommitteeJSON.Pubkeys {
		var err error
		if syncCommittee.Pubkeys[i], err = hexutil.Decode(pubKey); err != nil {
			return nil, err
		}
	}
	var err error
	if syncCommittee.AggregatePubkey, err = hexutil.Decode(syncCommitteeJSON.AggregatePubkey); err != nil {
		return nil, err
	}
	return syncCommittee, nil
}

func branchFromJSON(branch []string) ([][]byte, error) {
	var branchBytes [][]byte
	for _, root := range branch {
		branch, err := hexutil.Decode(root)
		if err != nil {
			return nil, err
		}
		branchBytes = append(branchBytes, branch)
	}
	return branchBytes, nil
}

func NewLightClientBootstrapFromJSON(bootstrapJSON *ethrpc.LightClientBootstrapJson) (*ethpbv2.LightClientBootstrap,
	error) {
	bootstrap := &ethpbv2.LightClientBootstrap{}
	var err error
	if bootstrap.Header, err = headerFromJSON(bootstrapJSON.Header); err != nil {
		return nil, err
	}
	if bootstrap.CurrentSyncCommittee, err = syncCommitteeFromJSON(bootstrapJSON.CurrentSyncCommittee); err != nil {
		return nil, err
	}
	if bootstrap.CurrentSyncCommitteeBranch, err = branchFromJSON(bootstrapJSON.CurrentSyncCommitteeBranch); err != nil {
		return nil, err
	}
	return bootstrap, nil
}

func NewLightClientUpdateFromJSON(updateJSON *ethrpc.LightClientUpdateJson) (*ethpbv2.LightClientUpdate, error) {
	update := &ethpbv2.LightClientUpdate{}
	var err error
	if update.AttestedHeader, err = headerFromJSON(updateJSON.AttestedHeader); err != nil {
		return nil, err
	}
	if update.NextSyncCommittee, err = syncCommitteeFromJSON(updateJSON.NextSyncCommittee); err != nil {
		return nil, err
	}
	if update.NextSyncCommitteeBranch, err = branchFromJSON(updateJSON.NextSyncCommitteeBranch); err != nil {
		return nil, err
	}
	if update.FinalizedHeader, err = headerFromJSON(updateJSON.FinalizedHeader); err != nil {
		return nil, err
	}
	if update.FinalityBranch, err = branchFromJSON(updateJSON.FinalityBranch); err != nil {
		return nil, err
	}
	if update.SyncAggregate, err = NewSyncAggregateFromJSON(updateJSON.SyncAggregate); err != nil {
		return nil, err
	}
	signatureSlot, err := strconv.ParseUint(updateJSON.SignatureSlot, 10, 64)
	if err != nil {
		return nil, err
	}
	update.SignatureSlot = types.Slot(signatureSlot)
	return update, nil
}

func NewLightClientUpdateFromFinalityUpdateJSON(updateJSON *ethrpc.LightClientFinalityUpdateJson) (*ethpbv2.
	LightClientUpdate, error) {
	update := &ethpbv2.LightClientUpdate{}
	var err error
	if update.AttestedHeader, err = headerFromJSON(updateJSON.AttestedHeader); err != nil {
		return nil, err
	}
	if update.FinalizedHeader, err = headerFromJSON(updateJSON.FinalizedHeader); err != nil {
		return nil, err
	}
	if update.FinalityBranch, err = branchFromJSON(updateJSON.FinalityBranch); err != nil {
		return nil, err
	}
	if update.SyncAggregate, err = NewSyncAggregateFromJSON(updateJSON.SyncAggregate); err != nil {
		return nil, err
	}
	signatureSlot, err := strconv.ParseUint(updateJSON.SignatureSlot, 10, 64)
	if err != nil {
		return nil, err
	}
	update.SignatureSlot = types.Slot(signatureSlot)
	return update, nil
}

func NewLightClientUpdateFromOptimisticUpdateJSON(updateJSON *ethrpc.LightClientOptimisticUpdateJson) (*ethpbv2.
	LightClientUpdate, error) {
	update := &ethpbv2.LightClientUpdate{}
	var err error
	if update.AttestedHeader, err = headerFromJSON(updateJSON.AttestedHeader); err != nil {
		return nil, err
	}
	if update.SyncAggregate, err = NewSyncAggregateFromJSON(updateJSON.SyncAggregate); err != nil {
		return nil, err
	}
	signatureSlot, err := strconv.ParseUint(updateJSON.SignatureSlot, 10, 64)
	if err != nil {
		return nil, err
	}
	update.SignatureSlot = types.Slot(signatureSlot)
	return update, nil
}

func NewLightClientFinalityUpdateFromJSON(updateJSON *ethrpc.LightClientFinalityUpdateJson) (*ethpbv2.
	LightClientFinalityUpdate, error) {
	update := &ethpbv2.LightClientFinalityUpdate{}
	var err error
	if update.AttestedHeader, err = headerFromJSON(updateJSON.AttestedHeader); err != nil {
		return nil, err
	}
	if update.FinalizedHeader, err = headerFromJSON(updateJSON.FinalizedHeader); err != nil {
		return nil, err
	}
	if update.FinalityBranch, err = branchFromJSON(updateJSON.FinalityBranch); err != nil {
		return nil, err
	}
	if update.SyncAggregate, err = NewSyncAggregateFromJSON(updateJSON.SyncAggregate); err != nil {
		return nil, err
	}
	signatureSlot, err := strconv.ParseUint(updateJSON.SignatureSlot, 10, 64)
	if err != nil {
		return nil, err
	}
	update.SignatureSlot = types.Slot(signatureSlot)
	return update, nil
}

func NewLightClientOptimisticUpdateFromJSON(updateJSON *ethrpc.LightClientOptimisticUpdateJson) (*ethpbv2.LightClientOptimisticUpdate,
	error) {
	update := &ethpbv2.LightClientOptimisticUpdate{}
	var err error
	if update.AttestedHeader, err = headerFromJSON(updateJSON.AttestedHeader); err != nil {
		return nil, err
	}
	if update.SyncAggregate, err = NewSyncAggregateFromJSON(updateJSON.SyncAggregate); err != nil {
		return nil, err
	}
	signatureSlot, err := strconv.ParseUint(updateJSON.SignatureSlot, 10, 64)
	if err != nil {
		return nil, err
	}
	update.SignatureSlot = types.Slot(signatureSlot)
	return update, nil
}
