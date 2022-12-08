package helpers

import (
	"math/big"
	"math/bits"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethrpc "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	v11 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
)

const (
	CurrentSyncCommitteeIndex = uint64(54)
	NextSyncCommitteeIndex    = uint64(55)
	FinalizedRootIndex        = uint64(105)
)

func bytesFromBigInt(numStr string) ([]byte, error) {
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return nil, err
	}
	return big.NewInt(0).SetUint64(num).Bytes(), nil
}

func NewExecutionPayloadHeaderFromJSON(header *ethrpc.ExecutionPayloadHeaderJson) (*v11.ExecutionPayloadHeader,
	error) {
	blockNumber, err := strconv.ParseUint(header.BlockNumber, 10, 64)
	if err != nil {
		return nil, err
	}
	gasLimit, err := strconv.ParseUint(header.GasLimit, 10, 64)
	if err != nil {
		return nil, err
	}
	gasUsed, err := strconv.ParseUint(header.GasUsed, 10, 64)
	if err != nil {
		return nil, err
	}
	timestamp, err := strconv.ParseUint(header.TimeStamp, 10, 64)
	if err != nil {
		return nil, err
	}
	baseFeePerGas, err := bytesFromBigInt(header.BaseFeePerGas)
	if err != nil {
		return nil, err
	}
	return &v11.ExecutionPayloadHeader{
		ParentHash:       hexutil.MustDecode(header.ParentHash),
		FeeRecipient:     hexutil.MustDecode(header.FeeRecipient),
		StateRoot:        hexutil.MustDecode(header.StateRoot),
		ReceiptsRoot:     hexutil.MustDecode(header.ReceiptsRoot),
		LogsBloom:        hexutil.MustDecode(header.LogsBloom),
		PrevRandao:       hexutil.MustDecode(header.PrevRandao),
		BlockNumber:      blockNumber,
		GasLimit:         gasLimit,
		GasUsed:          gasUsed,
		Timestamp:        timestamp,
		ExtraData:        hexutil.MustDecode(header.ExtraData),
		BaseFeePerGas:    baseFeePerGas,
		BlockHash:        hexutil.MustDecode(header.BlockHash),
		TransactionsRoot: hexutil.MustDecode(header.TransactionsRoot),
	}, nil
}

func FloorLog2(x uint64) int {
	return bits.Len64(uint64(x - 1))
}

func NewSyncAggregateFromJSON(syncAggregate *ethrpc.SyncAggregateJson) (*ethpbv1.SyncAggregate, error) {
	return &ethpbv1.SyncAggregate{
		SyncCommitteeBits:      hexutil.MustDecode(syncAggregate.SyncCommitteeBits),
		SyncCommitteeSignature: hexutil.MustDecode(syncAggregate.SyncCommitteeSignature),
	}, nil
}

func timeFromJSON(timestamp string) (*time.Time, error) {
	timeInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, err
	}
	t := time.Unix(timeInt, 0)
	return &t, nil
}

func NewGenesisResponse_GenesisFromJSON(genesis *ethrpc.GenesisResponse_GenesisJson) (
	*ethpbv1.GenesisResponse_Genesis, error) {
	genesisTime, err := timeFromJSON(genesis.GenesisTime)
	if err != nil {
		return nil, err
	}
	return &ethpbv1.GenesisResponse_Genesis{
		GenesisTime:           timestamppb.New(*genesisTime),
		GenesisValidatorsRoot: hexutil.MustDecode(genesis.GenesisValidatorsRoot),
		GenesisForkVersion:    hexutil.MustDecode(genesis.GenesisForkVersion),
	}, nil
}

func headerFromJSON(header *ethrpc.BeaconBlockHeaderJson) (*ethpbv1.BeaconBlockHeader, error) {
	slot, err := strconv.ParseUint(header.Slot, 10, 64)
	if err != nil {
		return nil, err
	}
	proposerIndex, err := strconv.ParseUint(header.ProposerIndex, 10, 64)
	if err != nil {
		return nil, err
	}
	return &ethpbv1.BeaconBlockHeader{
		Slot:          types.Slot(slot),
		ProposerIndex: types.ValidatorIndex(proposerIndex),
		ParentRoot:    hexutil.MustDecode(header.ParentRoot),
		StateRoot:     hexutil.MustDecode(header.StateRoot),
		BodyRoot:      hexutil.MustDecode(header.BodyRoot),
	}, nil
}

func syncCommitteeFromJSON(syncCommittee *ethrpc.SyncCommitteeJson) *ethpbv2.SyncCommittee {
	pubKeys := make([][]byte, len(syncCommittee.Pubkeys))
	for i, pubKey := range syncCommittee.Pubkeys {
		pubKeys[i] = hexutil.MustDecode(pubKey)
	}
	return &ethpbv2.SyncCommittee{
		Pubkeys:         pubKeys,
		AggregatePubkey: hexutil.MustDecode(syncCommittee.AggregatePubkey),
	}
}

func branchFromJSON(branch []string) [][]byte {
	branchBytes := [][]byte{}
	for _, root := range branch {
		branchBytes = append(branchBytes, hexutil.MustDecode(root))
	}
	return branchBytes
}

func NewLightClientBootstrapFromJSON(bootstrap *ethrpc.LightClientBootstrapJson) (*ethpbv2.LightClientBootstrap,
	error) {
	header, err := headerFromJSON(bootstrap.Header)
	if err != nil {
		return nil, err
	}
	return &ethpbv2.LightClientBootstrap{
		Header:                     header,
		CurrentSyncCommittee:       syncCommitteeFromJSON(bootstrap.CurrentSyncCommittee),
		CurrentSyncCommitteeBranch: branchFromJSON(bootstrap.CurrentSyncCommitteeBranch),
	}, nil
}

func NewLightClientUpdateFromJSON(update *ethrpc.LightClientUpdateDataJson) (*ethpbv2.LightClientUpdate, error) {
	attestedHeader, err := headerFromJSON(update.AttestedHeader)
	if err != nil {
		return nil, err
	}
	finalizedHeader, err := headerFromJSON(update.FinalizedHeader)
	if err != nil {
		return nil, err
	}
	syncAggregate, err := NewSyncAggregateFromJSON(update.SyncAggregate)
	if err != nil {
		return nil, err
	}
	signatureSlot, err := strconv.ParseUint(update.SignatureSlot, 10, 64)
	if err != nil {
		return nil, err
	}
	return &ethpbv2.LightClientUpdate{
		AttestedHeader:          attestedHeader,
		NextSyncCommittee:       syncCommitteeFromJSON(update.NextSyncCommittee),
		NextSyncCommitteeBranch: branchFromJSON(update.NextSyncCommitteeBranch),
		FinalizedHeader:         finalizedHeader,
		FinalityBranch:          branchFromJSON(update.FinalityBranch),
		SyncAggregate:           syncAggregate,
		SignatureSlot:           types.Slot(signatureSlot),
	}, nil
}
