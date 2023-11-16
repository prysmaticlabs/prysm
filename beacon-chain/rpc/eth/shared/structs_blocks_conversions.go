package shared

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	bytesutil2 "github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/math"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

var errNilValue = errors.New("nil value")

func (b *SignedBeaconBlock) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}

	sig, err := DecodeHexWithLength(b.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}

	bl, err := b.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}

	block := &eth.SignedBeaconBlock{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Phase0{Phase0: block}}, nil
}

func (b *BeaconBlock) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := b.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Phase0{Phase0: block}}, nil
}

func (b *BeaconBlock) ToConsensus() (*eth.BeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}
	if b.Body == nil {
		return nil, NewDecodeError(errNilValue, "Body")
	}
	if b.Body.Eth1Data == nil {
		return nil, NewDecodeError(errNilValue, "Body.Eth1Data")
	}

	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(b.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(b.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(b.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(b.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(b.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(b.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ProposerSlashings")
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.AttesterSlashings")
	}
	atts, err := AttsToConsensus(b.Body.Attestations)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Attestations")
	}
	deposits, err := DepositsToConsensus(b.Body.Deposits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Deposits")
	}
	exits, err := ExitsToConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.VoluntaryExits")
	}

	return &eth.BeaconBlock{
		Slot:          primitives.Slot(slot),
		ProposerIndex: primitives.ValidatorIndex(proposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body: &eth.BeaconBlockBody{
			RandaoReveal: randaoReveal,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  depositRoot,
				DepositCount: depositCount,
				BlockHash:    blockHash,
			},
			Graffiti:          graffiti,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
		},
	}, nil
}

func (b *SignedBeaconBlockAltair) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}

	sig, err := DecodeHexWithLength(b.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	bl, err := b.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}
	block := &eth.SignedBeaconBlockAltair{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Altair{Altair: block}}, nil
}

func (b *BeaconBlockAltair) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := b.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Altair{Altair: block}}, nil
}

func (b *BeaconBlockAltair) ToConsensus() (*eth.BeaconBlockAltair, error) {
	if b == nil {
		return nil, errNilValue
	}
	if b.Body == nil {
		return nil, NewDecodeError(errNilValue, "Body")
	}
	if b.Body.Eth1Data == nil {
		return nil, NewDecodeError(errNilValue, "Body.Eth1Data")
	}
	if b.Body.SyncAggregate == nil {
		return nil, NewDecodeError(errNilValue, "Body.SyncAggregate")
	}

	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(b.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(b.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(b.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(b.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(b.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(b.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ProposerSlashings")
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.AttesterSlashings")
	}
	atts, err := AttsToConsensus(b.Body.Attestations)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Attestations")
	}
	deposits, err := DepositsToConsensus(b.Body.Deposits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Deposits")
	}
	exits, err := ExitsToConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.VoluntaryExits")
	}
	syncCommitteeBits, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	return &eth.BeaconBlockAltair{
		Slot:          primitives.Slot(slot),
		ProposerIndex: primitives.ValidatorIndex(proposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body: &eth.BeaconBlockBodyAltair{
			RandaoReveal: randaoReveal,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  depositRoot,
				DepositCount: depositCount,
				BlockHash:    blockHash,
			},
			Graffiti:          graffiti,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &eth.SyncAggregate{
				SyncCommitteeBits:      syncCommitteeBits,
				SyncCommitteeSignature: syncCommitteeSig,
			},
		},
	}, nil
}

func (b *SignedBeaconBlockBellatrix) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}

	sig, err := DecodeHexWithLength(b.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	bl, err := b.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}
	block := &eth.SignedBeaconBlockBellatrix{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Bellatrix{Bellatrix: block}}, nil
}

func (b *BeaconBlockBellatrix) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := b.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Bellatrix{Bellatrix: block}}, nil
}

func (b *BeaconBlockBellatrix) ToConsensus() (*eth.BeaconBlockBellatrix, error) {
	if b == nil {
		return nil, errNilValue
	}
	if b.Body == nil {
		return nil, NewDecodeError(errNilValue, "Body")
	}
	if b.Body.Eth1Data == nil {
		return nil, NewDecodeError(errNilValue, "Body.Eth1Data")
	}
	if b.Body.SyncAggregate == nil {
		return nil, NewDecodeError(errNilValue, "Body.SyncAggregate")
	}
	if b.Body.ExecutionPayload == nil {
		return nil, NewDecodeError(errNilValue, "Body.ExecutionPayload")
	}

	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(b.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(b.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(b.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(b.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(b.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(b.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ProposerSlashings")
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.AttesterSlashings")
	}
	atts, err := AttsToConsensus(b.Body.Attestations)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Attestations")
	}
	deposits, err := DepositsToConsensus(b.Body.Deposits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Deposits")
	}
	exits, err := ExitsToConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.VoluntaryExits")
	}
	syncCommitteeBits, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(b.Body.ExecutionPayload.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(b.Body.ExecutionPayload.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(b.Body.ExecutionPayload.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(b.Body.ExecutionPayload.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(b.Body.ExecutionPayload.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(b.Body.ExecutionPayload.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Body.ExecutionPayload.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(b.Body.ExecutionPayload.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExtraData")
	}
	payloadBaseFeePerGas, err := Uint256ToSSZBytes(b.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(b.Body.ExecutionPayload.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockHash")
	}
	err = VerifyMaxLength(b.Body.ExecutionPayload.Transactions, fieldparams.MaxTxsPerPayloadLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.Transactions")
	}
	payloadTxs := make([][]byte, len(b.Body.ExecutionPayload.Transactions))
	for i, tx := range b.Body.ExecutionPayload.Transactions {
		payloadTxs[i], err = DecodeHexWithMaxLength(tx, fieldparams.MaxBytesPerTxLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Transactions[%d]", i))
		}
	}

	return &eth.BeaconBlockBellatrix{
		Slot:          primitives.Slot(slot),
		ProposerIndex: primitives.ValidatorIndex(proposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body: &eth.BeaconBlockBodyBellatrix{
			RandaoReveal: randaoReveal,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  depositRoot,
				DepositCount: depositCount,
				BlockHash:    blockHash,
			},
			Graffiti:          graffiti,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &eth.SyncAggregate{
				SyncCommitteeBits:      syncCommitteeBits,
				SyncCommitteeSignature: syncCommitteeSig,
			},
			ExecutionPayload: &enginev1.ExecutionPayload{
				ParentHash:    payloadParentHash,
				FeeRecipient:  payloadFeeRecipient,
				StateRoot:     payloadStateRoot,
				ReceiptsRoot:  payloadReceiptsRoot,
				LogsBloom:     payloadLogsBloom,
				PrevRandao:    payloadPrevRandao,
				BlockNumber:   payloadBlockNumber,
				GasLimit:      payloadGasLimit,
				GasUsed:       payloadGasUsed,
				Timestamp:     payloadTimestamp,
				ExtraData:     payloadExtraData,
				BaseFeePerGas: payloadBaseFeePerGas,
				BlockHash:     payloadBlockHash,
				Transactions:  payloadTxs,
			},
		},
	}, nil
}

func (b *SignedBlindedBeaconBlockBellatrix) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}

	sig, err := DecodeHexWithLength(b.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	bl, err := b.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}
	block := &eth.SignedBlindedBeaconBlockBellatrix{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (b *BlindedBeaconBlockBellatrix) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := b.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (b *BlindedBeaconBlockBellatrix) ToConsensus() (*eth.BlindedBeaconBlockBellatrix, error) {
	if b == nil {
		return nil, errNilValue
	}
	if b.Body == nil {
		return nil, NewDecodeError(errNilValue, "Body")
	}
	if b.Body.Eth1Data == nil {
		return nil, NewDecodeError(errNilValue, "Body.Eth1Data")
	}
	if b.Body.SyncAggregate == nil {
		return nil, NewDecodeError(errNilValue, "Body.SyncAggregate")
	}
	if b.Body.ExecutionPayloadHeader == nil {
		return nil, NewDecodeError(errNilValue, "Body.ExecutionPayloadHeader")
	}

	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(b.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(b.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(b.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(b.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(b.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(b.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ProposerSlashings")
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.AttesterSlashings")
	}
	atts, err := AttsToConsensus(b.Body.Attestations)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Attestations")
	}
	deposits, err := DepositsToConsensus(b.Body.Deposits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Deposits")
	}
	exits, err := ExitsToConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.VoluntaryExits")
	}
	syncCommitteeBits, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(b.Body.ExecutionPayloadHeader.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ExtraData")
	}
	payloadBaseFeePerGas, err := Uint256ToSSZBytes(b.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockHash")
	}
	payloadTxsRoot, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.TransactionsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.TransactionsRoot")
	}
	return &eth.BlindedBeaconBlockBellatrix{
		Slot:          primitives.Slot(slot),
		ProposerIndex: primitives.ValidatorIndex(proposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body: &eth.BlindedBeaconBlockBodyBellatrix{
			RandaoReveal: randaoReveal,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  depositRoot,
				DepositCount: depositCount,
				BlockHash:    blockHash,
			},
			Graffiti:          graffiti,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &eth.SyncAggregate{
				SyncCommitteeBits:      syncCommitteeBits,
				SyncCommitteeSignature: syncCommitteeSig,
			},
			ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeader{
				ParentHash:       payloadParentHash,
				FeeRecipient:     payloadFeeRecipient,
				StateRoot:        payloadStateRoot,
				ReceiptsRoot:     payloadReceiptsRoot,
				LogsBloom:        payloadLogsBloom,
				PrevRandao:       payloadPrevRandao,
				BlockNumber:      payloadBlockNumber,
				GasLimit:         payloadGasLimit,
				GasUsed:          payloadGasUsed,
				Timestamp:        payloadTimestamp,
				ExtraData:        payloadExtraData,
				BaseFeePerGas:    payloadBaseFeePerGas,
				BlockHash:        payloadBlockHash,
				TransactionsRoot: payloadTxsRoot,
			},
		},
	}, nil
}

func (b *SignedBeaconBlockCapella) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}

	sig, err := DecodeHexWithLength(b.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	bl, err := b.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}
	block := &eth.SignedBeaconBlockCapella{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Capella{Capella: block}}, nil
}

func (b *BeaconBlockCapella) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := b.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Capella{Capella: block}}, nil
}

func (b *BeaconBlockCapella) ToConsensus() (*eth.BeaconBlockCapella, error) {
	if b == nil {
		return nil, errNilValue
	}
	if b.Body == nil {
		return nil, NewDecodeError(errNilValue, "Body")
	}
	if b.Body.Eth1Data == nil {
		return nil, NewDecodeError(errNilValue, "Body.Eth1Data")
	}
	if b.Body.SyncAggregate == nil {
		return nil, NewDecodeError(errNilValue, "Body.SyncAggregate")
	}
	if b.Body.ExecutionPayload == nil {
		return nil, NewDecodeError(errNilValue, "Body.ExecutionPayload")
	}

	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(b.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(b.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(b.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(b.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(b.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(b.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ProposerSlashings")
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.AttesterSlashings")
	}
	atts, err := AttsToConsensus(b.Body.Attestations)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Attestations")
	}
	deposits, err := DepositsToConsensus(b.Body.Deposits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Deposits")
	}
	exits, err := ExitsToConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.VoluntaryExits")
	}
	syncCommitteeBits, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(b.Body.ExecutionPayload.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(b.Body.ExecutionPayload.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(b.Body.ExecutionPayload.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(b.Body.ExecutionPayload.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(b.Body.ExecutionPayload.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(b.Body.ExecutionPayload.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Body.ExecutionPayload.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(b.Body.ExecutionPayload.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExtraData")
	}
	payloadBaseFeePerGas, err := Uint256ToSSZBytes(b.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(b.Body.ExecutionPayload.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockHash")
	}
	err = VerifyMaxLength(b.Body.ExecutionPayload.Transactions, fieldparams.MaxTxsPerPayloadLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.Transactions")
	}
	payloadTxs := make([][]byte, len(b.Body.ExecutionPayload.Transactions))
	for i, tx := range b.Body.ExecutionPayload.Transactions {
		payloadTxs[i], err = DecodeHexWithMaxLength(tx, fieldparams.MaxBytesPerTxLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Transactions[%d]", i))
		}
	}
	err = VerifyMaxLength(b.Body.ExecutionPayload.Withdrawals, fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.Withdrawals")
	}
	withdrawals := make([]*enginev1.Withdrawal, len(b.Body.ExecutionPayload.Withdrawals))
	for i, w := range b.Body.ExecutionPayload.Withdrawals {
		withdrawalIndex, err := strconv.ParseUint(w.WithdrawalIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Withdrawals[%d].WithdrawalIndex", i))
		}
		validatorIndex, err := strconv.ParseUint(w.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Withdrawals[%d].ValidatorIndex", i))
		}
		address, err := DecodeHexWithLength(w.ExecutionAddress, common.AddressLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Withdrawals[%d].ExecutionAddress", i))
		}
		amount, err := strconv.ParseUint(w.Amount, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Withdrawals[%d].Amount", i))
		}
		withdrawals[i] = &enginev1.Withdrawal{
			Index:          withdrawalIndex,
			ValidatorIndex: primitives.ValidatorIndex(validatorIndex),
			Address:        address,
			Amount:         amount,
		}
	}
	blsChanges, err := BlsChangesToConsensus(b.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, NewDecodeError(err, "Body.BlsToExecutionChanges")
	}

	return &eth.BeaconBlockCapella{
		Slot:          primitives.Slot(slot),
		ProposerIndex: primitives.ValidatorIndex(proposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body: &eth.BeaconBlockBodyCapella{
			RandaoReveal: randaoReveal,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  depositRoot,
				DepositCount: depositCount,
				BlockHash:    blockHash,
			},
			Graffiti:          graffiti,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &eth.SyncAggregate{
				SyncCommitteeBits:      syncCommitteeBits,
				SyncCommitteeSignature: syncCommitteeSig,
			},
			ExecutionPayload: &enginev1.ExecutionPayloadCapella{
				ParentHash:    payloadParentHash,
				FeeRecipient:  payloadFeeRecipient,
				StateRoot:     payloadStateRoot,
				ReceiptsRoot:  payloadReceiptsRoot,
				LogsBloom:     payloadLogsBloom,
				PrevRandao:    payloadPrevRandao,
				BlockNumber:   payloadBlockNumber,
				GasLimit:      payloadGasLimit,
				GasUsed:       payloadGasUsed,
				Timestamp:     payloadTimestamp,
				ExtraData:     payloadExtraData,
				BaseFeePerGas: payloadBaseFeePerGas,
				BlockHash:     payloadBlockHash,
				Transactions:  payloadTxs,
				Withdrawals:   withdrawals,
			},
			BlsToExecutionChanges: blsChanges,
		},
	}, nil
}

func (b *SignedBlindedBeaconBlockCapella) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}

	sig, err := DecodeHexWithLength(b.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	bl, err := b.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}
	block := &eth.SignedBlindedBeaconBlockCapella{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_BlindedCapella{BlindedCapella: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (b *BlindedBeaconBlockCapella) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := b.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_BlindedCapella{BlindedCapella: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (b *BlindedBeaconBlockCapella) ToConsensus() (*eth.BlindedBeaconBlockCapella, error) {
	if b == nil {
		return nil, errNilValue
	}
	if b.Body == nil {
		return nil, NewDecodeError(errNilValue, "Body")
	}
	if b.Body.Eth1Data == nil {
		return nil, NewDecodeError(errNilValue, "Body.Eth1Data")
	}
	if b.Body.SyncAggregate == nil {
		return nil, NewDecodeError(errNilValue, "Body.SyncAggregate")
	}
	if b.Body.ExecutionPayloadHeader == nil {
		return nil, NewDecodeError(errNilValue, "Body.ExecutionPayloadHeader")
	}

	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(b.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(b.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(b.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(b.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(b.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(b.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ProposerSlashings")
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.AttesterSlashings")
	}
	atts, err := AttsToConsensus(b.Body.Attestations)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Attestations")
	}
	deposits, err := DepositsToConsensus(b.Body.Deposits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Deposits")
	}
	exits, err := ExitsToConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.VoluntaryExits")
	}
	syncCommitteeBits, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(b.Body.ExecutionPayloadHeader.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ExtraData")
	}
	payloadBaseFeePerGas, err := Uint256ToSSZBytes(b.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithMaxLength(b.Body.ExecutionPayloadHeader.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockHash")
	}
	payloadTxsRoot, err := DecodeHexWithMaxLength(b.Body.ExecutionPayloadHeader.TransactionsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.TransactionsRoot")
	}
	payloadWithdrawalsRoot, err := DecodeHexWithMaxLength(b.Body.ExecutionPayloadHeader.WithdrawalsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.WithdrawalsRoot")
	}
	blsChanges, err := BlsChangesToConsensus(b.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, NewDecodeError(err, "Body.BlsToExecutionChanges")
	}

	return &eth.BlindedBeaconBlockCapella{
		Slot:          primitives.Slot(slot),
		ProposerIndex: primitives.ValidatorIndex(proposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body: &eth.BlindedBeaconBlockBodyCapella{
			RandaoReveal: randaoReveal,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  depositRoot,
				DepositCount: depositCount,
				BlockHash:    blockHash,
			},
			Graffiti:          graffiti,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &eth.SyncAggregate{
				SyncCommitteeBits:      syncCommitteeBits,
				SyncCommitteeSignature: syncCommitteeSig,
			},
			ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderCapella{
				ParentHash:       payloadParentHash,
				FeeRecipient:     payloadFeeRecipient,
				StateRoot:        payloadStateRoot,
				ReceiptsRoot:     payloadReceiptsRoot,
				LogsBloom:        payloadLogsBloom,
				PrevRandao:       payloadPrevRandao,
				BlockNumber:      payloadBlockNumber,
				GasLimit:         payloadGasLimit,
				GasUsed:          payloadGasUsed,
				Timestamp:        payloadTimestamp,
				ExtraData:        payloadExtraData,
				BaseFeePerGas:    payloadBaseFeePerGas,
				BlockHash:        payloadBlockHash,
				TransactionsRoot: payloadTxsRoot,
				WithdrawalsRoot:  payloadWithdrawalsRoot,
			},
			BlsToExecutionChanges: blsChanges,
		},
	}, nil
}

func (b *SignedBeaconBlockContentsDeneb) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}

	var signedBlobSidecars []*eth.SignedBlobSidecar
	if len(b.SignedBlobSidecars) != 0 {
		err := VerifyMaxLength(b.SignedBlobSidecars, fieldparams.MaxBlobsPerBlock)
		if err != nil {
			return nil, NewDecodeError(err, "SignedBlobSidecars")
		}
		signedBlobSidecars = make([]*eth.SignedBlobSidecar, len(b.SignedBlobSidecars))
		for i := range b.SignedBlobSidecars {
			signedBlob, err := b.SignedBlobSidecars[i].ToConsensus()
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("SignedBlobSidecars[%d]", i))
			}
			signedBlobSidecars[i] = signedBlob
		}
	}
	signedDenebBlock, err := b.SignedBlock.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "SignedBlock")
	}
	block := &eth.SignedBeaconBlockAndBlobsDeneb{
		Block: signedDenebBlock,
		Blobs: signedBlobSidecars,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Deneb{Deneb: block}}, nil
}

func (b *SignedBeaconBlockContentsDeneb) ToUnsigned() *BeaconBlockContentsDeneb {
	var blobSidecars []*BlobSidecar
	if len(b.SignedBlobSidecars) != 0 {
		blobSidecars = make([]*BlobSidecar, len(b.SignedBlobSidecars))
		for i, s := range b.SignedBlobSidecars {
			blobSidecars[i] = s.Message
		}
	}
	return &BeaconBlockContentsDeneb{
		Block:        b.SignedBlock.Message,
		BlobSidecars: blobSidecars,
	}
}

func (b *BeaconBlockContentsDeneb) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := b.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Deneb{Deneb: block}}, nil
}

func (b *BeaconBlockContentsDeneb) ToConsensus() (*eth.BeaconBlockAndBlobsDeneb, error) {
	if b == nil {
		return nil, errNilValue
	}

	var blobSidecars []*eth.DeprecatedBlobSidecar
	if len(b.BlobSidecars) != 0 {
		err := VerifyMaxLength(b.BlobSidecars, fieldparams.MaxBlobsPerBlock)
		if err != nil {
			return nil, NewDecodeError(err, "BlobSidecars")
		}
		blobSidecars = make([]*eth.DeprecatedBlobSidecar, len(b.BlobSidecars))
		for i := range b.BlobSidecars {
			blob, err := b.BlobSidecars[i].ToConsensus()
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("BlobSidecars[%d]", i))
			}
			blobSidecars[i] = blob
		}
	}
	denebBlock, err := b.Block.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Block")
	}
	return &eth.BeaconBlockAndBlobsDeneb{
		Block: denebBlock,
		Blobs: blobSidecars,
	}, nil
}

func (b *SignedBlindedBeaconBlockContentsDeneb) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}

	var signedBlindedBlobSidecars []*eth.SignedBlindedBlobSidecar
	if len(b.SignedBlindedBlobSidecars) != 0 {
		err := VerifyMaxLength(b.SignedBlindedBlobSidecars, fieldparams.MaxBlobsPerBlock)
		if err != nil {
			return nil, NewDecodeError(err, "SignedBlindedBlobSidecars")
		}
		signedBlindedBlobSidecars = make([]*eth.SignedBlindedBlobSidecar, len(b.SignedBlindedBlobSidecars))
		for i := range b.SignedBlindedBlobSidecars {
			signedBlob, err := b.SignedBlindedBlobSidecars[i].ToConsensus()
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("SignedBlindedBlobSidecars[%d]", i))
			}
			signedBlindedBlobSidecars[i] = signedBlob
		}
	}
	signedBlindedBlock, err := b.SignedBlindedBlock.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "SignedBlindedBlock")
	}
	block := &eth.SignedBlindedBeaconBlockAndBlobsDeneb{
		SignedBlindedBlock:        signedBlindedBlock,
		SignedBlindedBlobSidecars: signedBlindedBlobSidecars,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_BlindedDeneb{BlindedDeneb: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (b *SignedBlindedBeaconBlockContentsDeneb) ToUnsigned() *BlindedBeaconBlockContentsDeneb {
	var blobSidecars []*BlindedBlobSidecar
	if len(b.SignedBlindedBlobSidecars) != 0 {
		blobSidecars = make([]*BlindedBlobSidecar, len(b.SignedBlindedBlobSidecars))
		for i := range b.SignedBlindedBlobSidecars {
			blobSidecars[i] = b.SignedBlindedBlobSidecars[i].Message
		}
	}
	return &BlindedBeaconBlockContentsDeneb{
		BlindedBlock:        b.SignedBlindedBlock.Message,
		BlindedBlobSidecars: blobSidecars,
	}
}

func (b *BlindedBeaconBlockContentsDeneb) ToGeneric() (*eth.GenericBeaconBlock, error) {
	if b == nil {
		return nil, errNilValue
	}

	var blindedBlobSidecars []*eth.BlindedBlobSidecar
	if len(b.BlindedBlobSidecars) != 0 {
		err := VerifyMaxLength(b.BlindedBlobSidecars, fieldparams.MaxBlobsPerBlock)
		if err != nil {
			return nil, NewDecodeError(err, "BlindedBlobSidecars")
		}
		blindedBlobSidecars = make([]*eth.BlindedBlobSidecar, len(b.BlindedBlobSidecars))
		for i := range b.BlindedBlobSidecars {
			blob, err := b.BlindedBlobSidecars[i].ToConsensus()
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("BlindedBlobSidecars[%d]", i))
			}
			blindedBlobSidecars[i] = blob
		}
	}
	blindedBlock, err := b.BlindedBlock.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "BlindedBlock")
	}
	block := &eth.BlindedBeaconBlockAndBlobsDeneb{
		Block: blindedBlock,
		Blobs: blindedBlobSidecars,
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_BlindedDeneb{BlindedDeneb: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (b *BeaconBlockDeneb) ToConsensus() (*eth.BeaconBlockDeneb, error) {
	if b == nil {
		return nil, errNilValue
	}
	if b.Body == nil {
		return nil, NewDecodeError(errNilValue, "Body")
	}
	if b.Body.Eth1Data == nil {
		return nil, NewDecodeError(errNilValue, "Body.Eth1Data")
	}
	if b.Body.SyncAggregate == nil {
		return nil, NewDecodeError(errNilValue, "Body.SyncAggregate")
	}
	if b.Body.ExecutionPayload == nil {
		return nil, NewDecodeError(errNilValue, "Body.ExecutionPayload")
	}

	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(b.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(b.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(b.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(b.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(b.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(b.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ProposerSlashings")
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.AttesterSlashings")
	}
	atts, err := AttsToConsensus(b.Body.Attestations)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Attestations")
	}
	deposits, err := DepositsToConsensus(b.Body.Deposits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Deposits")
	}
	exits, err := ExitsToConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.VoluntaryExits")
	}
	syncCommitteeBits, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(b.Body.ExecutionPayload.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(b.Body.ExecutionPayload.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(b.Body.ExecutionPayload.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(b.Body.ExecutionPayload.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(b.Body.ExecutionPayload.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(b.Body.ExecutionPayload.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Body.ExecutionPayload.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(b.Body.ExecutionPayload.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExtraData")
	}
	payloadBaseFeePerGas, err := Uint256ToSSZBytes(b.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(b.Body.ExecutionPayload.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockHash")
	}
	err = VerifyMaxLength(b.Body.ExecutionPayload.Transactions, fieldparams.MaxTxsPerPayloadLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.Transactions")
	}
	txs := make([][]byte, len(b.Body.ExecutionPayload.Transactions))
	for i, tx := range b.Body.ExecutionPayload.Transactions {
		txs[i], err = DecodeHexWithMaxLength(tx, fieldparams.MaxBytesPerTxLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Transactions[%d]", i))
		}
	}
	err = VerifyMaxLength(b.Body.ExecutionPayload.Withdrawals, fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.Withdrawals")
	}
	withdrawals := make([]*enginev1.Withdrawal, len(b.Body.ExecutionPayload.Withdrawals))
	for i, w := range b.Body.ExecutionPayload.Withdrawals {
		withdrawalIndex, err := strconv.ParseUint(w.WithdrawalIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Withdrawals[%d].WithdrawalIndex", i))
		}
		validatorIndex, err := strconv.ParseUint(w.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Withdrawals[%d].ValidatorIndex", i))
		}
		address, err := DecodeHexWithLength(w.ExecutionAddress, common.AddressLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Withdrawals[%d].ExecutionAddress", i))
		}
		amount, err := strconv.ParseUint(w.Amount, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Withdrawals[%d].Amount", i))
		}
		withdrawals[i] = &enginev1.Withdrawal{
			Index:          withdrawalIndex,
			ValidatorIndex: primitives.ValidatorIndex(validatorIndex),
			Address:        address,
			Amount:         amount,
		}
	}

	payloadBlobGasUsed, err := strconv.ParseUint(b.Body.ExecutionPayload.BlobGasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlobGasUsed")
	}
	payloadExcessBlobGas, err := strconv.ParseUint(b.Body.ExecutionPayload.ExcessBlobGas, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExcessBlobGas")
	}
	blsChanges, err := BlsChangesToConsensus(b.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, NewDecodeError(err, "Body.BlsToExecutionChanges")
	}
	err = VerifyMaxLength(b.Body.BlobKzgCommitments, fieldparams.MaxBlobCommitmentsPerBlock)
	if err != nil {
		return nil, NewDecodeError(err, "Body.BlobKzgCommitments")
	}
	blobKzgCommitments := make([][]byte, len(b.Body.BlobKzgCommitments))
	for i, b := range b.Body.BlobKzgCommitments {
		kzg, err := DecodeHexWithLength(b, fieldparams.BLSPubkeyLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.BlobKzgCommitments[%d]", i))
		}
		blobKzgCommitments[i] = kzg
	}
	return &eth.BeaconBlockDeneb{
		Slot:          primitives.Slot(slot),
		ProposerIndex: primitives.ValidatorIndex(proposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body: &eth.BeaconBlockBodyDeneb{
			RandaoReveal: randaoReveal,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  depositRoot,
				DepositCount: depositCount,
				BlockHash:    blockHash,
			},
			Graffiti:          graffiti,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &eth.SyncAggregate{
				SyncCommitteeBits:      syncCommitteeBits,
				SyncCommitteeSignature: syncCommitteeSig,
			},
			ExecutionPayload: &enginev1.ExecutionPayloadDeneb{
				ParentHash:    payloadParentHash,
				FeeRecipient:  payloadFeeRecipient,
				StateRoot:     payloadStateRoot,
				ReceiptsRoot:  payloadReceiptsRoot,
				LogsBloom:     payloadLogsBloom,
				PrevRandao:    payloadPrevRandao,
				BlockNumber:   payloadBlockNumber,
				GasLimit:      payloadGasLimit,
				GasUsed:       payloadGasUsed,
				Timestamp:     payloadTimestamp,
				ExtraData:     payloadExtraData,
				BaseFeePerGas: payloadBaseFeePerGas,
				BlockHash:     payloadBlockHash,
				Transactions:  txs,
				Withdrawals:   withdrawals,
				BlobGasUsed:   payloadBlobGasUsed,
				ExcessBlobGas: payloadExcessBlobGas,
			},
			BlsToExecutionChanges: blsChanges,
			BlobKzgCommitments:    blobKzgCommitments,
		},
	}, nil
}

func (s *BlobSidecar) ToConsensus() (*eth.DeprecatedBlobSidecar, error) {
	if s == nil {
		return nil, errNilValue
	}
	blockRoot, err := DecodeHexWithLength(s.BlockRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "BlockRoot")
	}
	index, err := strconv.ParseUint(s.Index, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Index")
	}
	slot, err := strconv.ParseUint(s.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	blockParentRoot, err := DecodeHexWithLength(s.BlockParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "BlockParentRoot")
	}
	proposerIndex, err := strconv.ParseUint(s.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	blob, err := DecodeHexWithLength(s.Blob, fieldparams.BlobLength)
	if err != nil {
		return nil, NewDecodeError(err, "Blob")
	}
	kzgCommitment, err := DecodeHexWithLength(s.KzgCommitment, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, "KzgCommitment")
	}
	kzgProof, err := DecodeHexWithLength(s.KzgProof, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, "KzgProof")
	}
	bsc := &eth.DeprecatedBlobSidecar{
		BlockRoot:       blockRoot,
		Index:           index,
		Slot:            primitives.Slot(slot),
		BlockParentRoot: blockParentRoot,
		ProposerIndex:   primitives.ValidatorIndex(proposerIndex),
		Blob:            blob,
		KzgCommitment:   kzgCommitment,
		KzgProof:        kzgProof,
	}
	return bsc, nil
}

func (b *SignedBeaconBlockDeneb) ToConsensus() (*eth.SignedBeaconBlockDeneb, error) {
	if b == nil {
		return nil, errNilValue
	}

	sig, err := DecodeHexWithLength(b.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	block, err := b.Message.ToConsensus()
	if err != nil {
		return nil, NewDecodeError(err, "Message")
	}
	return &eth.SignedBeaconBlockDeneb{
		Block:     block,
		Signature: sig,
	}, nil
}

func (s *SignedBlobSidecar) ToConsensus() (*eth.SignedBlobSidecar, error) {
	if s == nil {
		return nil, errNilValue
	}
	if s.Message == nil {
		return nil, NewDecodeError(errNilValue, "Message")
	}

	blobSig, err := DecodeHexWithLength(s.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	blockRoot, err := DecodeHexWithLength(s.Message.BlockRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Message.BlockRoot")
	}
	index, err := strconv.ParseUint(s.Message.Index, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Message.Index")
	}
	slot, err := strconv.ParseUint(s.Message.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Message.Slot")
	}
	blockParentRoot, err := DecodeHexWithLength(s.Message.BlockParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Message.BlockParentRoot")
	}
	proposerIndex, err := strconv.ParseUint(s.Message.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Message.ProposerIndex")
	}
	blob, err := DecodeHexWithLength(s.Message.Blob, fieldparams.BlobLength)
	if err != nil {
		return nil, NewDecodeError(err, "Message.Blob")
	}
	kzgCommitment, err := DecodeHexWithLength(s.Message.KzgCommitment, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, "Message.KzgCommitment")
	}
	kzgProof, err := DecodeHexWithLength(s.Message.KzgProof, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, "Message.KzgProof")
	}
	bsc := &eth.DeprecatedBlobSidecar{
		BlockRoot:       blockRoot,
		Index:           index,
		Slot:            primitives.Slot(slot),
		BlockParentRoot: blockParentRoot,
		ProposerIndex:   primitives.ValidatorIndex(proposerIndex),
		Blob:            blob,
		KzgCommitment:   kzgCommitment,
		KzgProof:        kzgProof,
	}
	return &eth.SignedBlobSidecar{
		Message:   bsc,
		Signature: blobSig,
	}, nil
}

func (b *SignedBlindedBeaconBlockDeneb) ToConsensus() (*eth.SignedBlindedBeaconBlockDeneb, error) {
	if b == nil {
		return nil, errNilValue
	}

	sig, err := DecodeHexWithLength(b.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	blindedBlock, err := b.Message.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.SignedBlindedBeaconBlockDeneb{
		Message:   blindedBlock,
		Signature: sig,
	}, nil
}

func (b *BlindedBeaconBlockDeneb) ToConsensus() (*eth.BlindedBeaconBlockDeneb, error) {
	if b == nil {
		return nil, errNilValue
	}
	if b.Body == nil {
		return nil, NewDecodeError(errNilValue, "Body")
	}
	if b.Body.Eth1Data == nil {
		return nil, NewDecodeError(errNilValue, "Body.Eth1Data")
	}
	if b.Body.SyncAggregate == nil {
		return nil, NewDecodeError(errNilValue, "Body.SyncAggregate")
	}
	if b.Body.ExecutionPayloadHeader == nil {
		return nil, NewDecodeError(errNilValue, "Body.ExecutionPayloadHeader")
	}

	slot, err := strconv.ParseUint(b.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(b.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(b.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(b.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(b.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(b.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(b.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(b.Body.Eth1Data.BlockHash, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(b.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ProposerSlashings")
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, NewDecodeError(err, "Body.AttesterSlashings")
	}
	atts, err := AttsToConsensus(b.Body.Attestations)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Attestations")
	}
	deposits, err := DepositsToConsensus(b.Body.Deposits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Deposits")
	}
	exits, err := ExitsToConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, NewDecodeError(err, "Body.VoluntaryExits")
	}
	syncCommitteeBits, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(b.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(b.Body.ExecutionPayloadHeader.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ExtraData")
	}
	payloadBaseFeePerGas, err := Uint256ToSSZBytes(b.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockHash")
	}
	payloadTxsRoot, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.TransactionsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.TransactionsRoot")
	}
	payloadWithdrawalsRoot, err := DecodeHexWithLength(b.Body.ExecutionPayloadHeader.WithdrawalsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.WithdrawalsRoot")
	}

	payloadBlobGasUsed, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.BlobGasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlobGasUsed")
	}
	payloadExcessBlobGas, err := strconv.ParseUint(b.Body.ExecutionPayloadHeader.ExcessBlobGas, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExcessBlobGas")
	}

	blsChanges, err := BlsChangesToConsensus(b.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, NewDecodeError(err, "Body.BlsToExecutionChanges")
	}
	err = VerifyMaxLength(b.Body.BlobKzgCommitments, fieldparams.MaxBlobCommitmentsPerBlock)
	if err != nil {
		return nil, NewDecodeError(err, "Body.BlobKzgCommitments")
	}
	blobKzgCommitments := make([][]byte, len(b.Body.BlobKzgCommitments))
	for i, b := range b.Body.BlobKzgCommitments {
		kzg, err := DecodeHexWithLength(b, fieldparams.BLSPubkeyLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.BlobKzgCommitments[%d]", i))
		}
		blobKzgCommitments[i] = kzg
	}

	return &eth.BlindedBeaconBlockDeneb{
		Slot:          primitives.Slot(slot),
		ProposerIndex: primitives.ValidatorIndex(proposerIndex),
		ParentRoot:    parentRoot,
		StateRoot:     stateRoot,
		Body: &eth.BlindedBeaconBlockBodyDeneb{
			RandaoReveal: randaoReveal,
			Eth1Data: &eth.Eth1Data{
				DepositRoot:  depositRoot,
				DepositCount: depositCount,
				BlockHash:    blockHash,
			},
			Graffiti:          graffiti,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &eth.SyncAggregate{
				SyncCommitteeBits:      syncCommitteeBits,
				SyncCommitteeSignature: syncCommitteeSig,
			},
			ExecutionPayloadHeader: &enginev1.ExecutionPayloadHeaderDeneb{
				ParentHash:       payloadParentHash,
				FeeRecipient:     payloadFeeRecipient,
				StateRoot:        payloadStateRoot,
				ReceiptsRoot:     payloadReceiptsRoot,
				LogsBloom:        payloadLogsBloom,
				PrevRandao:       payloadPrevRandao,
				BlockNumber:      payloadBlockNumber,
				GasLimit:         payloadGasLimit,
				GasUsed:          payloadGasUsed,
				Timestamp:        payloadTimestamp,
				ExtraData:        payloadExtraData,
				BaseFeePerGas:    payloadBaseFeePerGas,
				BlockHash:        payloadBlockHash,
				TransactionsRoot: payloadTxsRoot,
				WithdrawalsRoot:  payloadWithdrawalsRoot,
				BlobGasUsed:      payloadBlobGasUsed,
				ExcessBlobGas:    payloadExcessBlobGas,
			},
			BlsToExecutionChanges: blsChanges,
			BlobKzgCommitments:    blobKzgCommitments,
		},
	}, nil
}

func (s *SignedBlindedBlobSidecar) ToConsensus() (*eth.SignedBlindedBlobSidecar, error) {
	if s == nil {
		return nil, errNilValue
	}

	blobSig, err := DecodeHexWithLength(s.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	bsc, err := s.Message.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.SignedBlindedBlobSidecar{
		Message:   bsc,
		Signature: blobSig,
	}, nil
}

func (s *BlindedBlobSidecar) ToConsensus() (*eth.BlindedBlobSidecar, error) {
	if s == nil {
		return nil, errNilValue
	}
	blockRoot, err := DecodeHexWithLength(s.BlockRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "BlockRoot")
	}
	index, err := strconv.ParseUint(s.Index, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Index")
	}
	denebSlot, err := strconv.ParseUint(s.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	blockParentRoot, err := DecodeHexWithLength(s.BlockParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "BlockParentRoot")
	}
	proposerIndex, err := strconv.ParseUint(s.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	blobRoot, err := DecodeHexWithLength(s.BlobRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "BlobRoot")
	}
	kzgCommitment, err := DecodeHexWithLength(s.KzgCommitment, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, "KzgCommitment")
	}
	kzgProof, err := DecodeHexWithLength(s.KzgProof, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, "KzgProof")
	}
	bsc := &eth.BlindedBlobSidecar{
		BlockRoot:       blockRoot,
		Index:           index,
		Slot:            primitives.Slot(denebSlot),
		BlockParentRoot: blockParentRoot,
		ProposerIndex:   primitives.ValidatorIndex(proposerIndex),
		BlobRoot:        blobRoot,
		KzgCommitment:   kzgCommitment,
		KzgProof:        kzgProof,
	}
	return bsc, nil
}

func BeaconBlockHeaderFromConsensus(h *eth.BeaconBlockHeader) *BeaconBlockHeader {
	return &BeaconBlockHeader{
		Slot:          strconv.FormatUint(uint64(h.Slot), 10),
		ProposerIndex: strconv.FormatUint(uint64(h.ProposerIndex), 10),
		ParentRoot:    hexutil.Encode(h.ParentRoot),
		StateRoot:     hexutil.Encode(h.StateRoot),
		BodyRoot:      hexutil.Encode(h.BodyRoot),
	}
}

func BeaconBlockFromConsensus(b *eth.BeaconBlock) (*BeaconBlock, error) {
	proposerSlashings := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	attesterSlashings := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	atts, err := AttsFromConsensus(b.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsFromConsensus(b.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsFromConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	e1d, err := Eth1DataFromConsensus(b.Body.Eth1Data)
	if err != nil {
		return nil, err
	}

	return &BeaconBlock{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBody{
			RandaoReveal:      hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:          e1d,
			Graffiti:          hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
		},
	}, nil
}

func SignedBeaconBlockFromConsensus(b *eth.SignedBeaconBlock) (*SignedBeaconBlock, error) {
	block, err := BeaconBlockFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &SignedBeaconBlock{
		Message:   block,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BeaconBlockAltairFromConsensus(b *eth.BeaconBlockAltair) (*BeaconBlockAltair, error) {
	proposerSlashings := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	attesterSlashings := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	atts, err := AttsFromConsensus(b.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsFromConsensus(b.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsFromConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	e1d, err := Eth1DataFromConsensus(b.Body.Eth1Data)
	if err != nil {
		return nil, err
	}

	return &BeaconBlockAltair{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyAltair{
			RandaoReveal:      hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:          e1d,
			Graffiti:          hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &SyncAggregate{
				SyncCommitteeBits:      hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeBits),
				SyncCommitteeSignature: hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeSignature),
			},
		},
	}, nil
}

func SignedBeaconBlockAltairFromConsensus(b *eth.SignedBeaconBlockAltair) (*SignedBeaconBlockAltair, error) {
	block, err := BeaconBlockAltairFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &SignedBeaconBlockAltair{
		Message:   block,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BlindedBeaconBlockBellatrixFromConsensus(b *eth.BlindedBeaconBlockBellatrix) (*BlindedBeaconBlockBellatrix, error) {
	proposerSlashings := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	attesterSlashings := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	atts, err := AttsFromConsensus(b.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsFromConsensus(b.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsFromConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	e1d, err := Eth1DataFromConsensus(b.Body.Eth1Data)
	if err != nil {
		return nil, err
	}
	payload, err := ExecutionPayloadHeaderFromConsensus(b.Body.ExecutionPayloadHeader)
	if err != nil {
		return nil, err
	}

	return &BlindedBeaconBlockBellatrix{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BlindedBeaconBlockBodyBellatrix{
			RandaoReveal:      hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:          e1d,
			Graffiti:          hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &SyncAggregate{
				SyncCommitteeBits:      hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeBits),
				SyncCommitteeSignature: hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeSignature),
			},
			ExecutionPayloadHeader: payload,
		},
	}, nil
}

func SignedBlindedBeaconBlockBellatrixFromConsensus(b *eth.SignedBlindedBeaconBlockBellatrix) (*SignedBlindedBeaconBlockBellatrix, error) {
	blindedBlock, err := BlindedBeaconBlockBellatrixFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &SignedBlindedBeaconBlockBellatrix{
		Message:   blindedBlock,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BeaconBlockBellatrixFromConsensus(b *eth.BeaconBlockBellatrix) (*BeaconBlockBellatrix, error) {
	proposerSlashings := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	attesterSlashings := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	atts, err := AttsFromConsensus(b.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsFromConsensus(b.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsFromConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	baseFeePerGas, err := sszBytesToUint256String(b.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, err
	}
	transactions := make([]string, len(b.Body.ExecutionPayload.Transactions))
	for i, tx := range b.Body.ExecutionPayload.Transactions {
		transactions[i] = hexutil.Encode(tx)
	}
	e1d, err := Eth1DataFromConsensus(b.Body.Eth1Data)
	if err != nil {
		return nil, err
	}

	return &BeaconBlockBellatrix{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyBellatrix{
			RandaoReveal:      hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:          e1d,
			Graffiti:          hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &SyncAggregate{
				SyncCommitteeBits:      hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeBits),
				SyncCommitteeSignature: hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeSignature),
			},
			ExecutionPayload: &ExecutionPayload{
				ParentHash:    hexutil.Encode(b.Body.ExecutionPayload.ParentHash),
				FeeRecipient:  hexutil.Encode(b.Body.ExecutionPayload.FeeRecipient),
				StateRoot:     hexutil.Encode(b.Body.ExecutionPayload.StateRoot),
				ReceiptsRoot:  hexutil.Encode(b.Body.ExecutionPayload.ReceiptsRoot),
				LogsBloom:     hexutil.Encode(b.Body.ExecutionPayload.LogsBloom),
				PrevRandao:    hexutil.Encode(b.Body.ExecutionPayload.PrevRandao),
				BlockNumber:   fmt.Sprintf("%d", b.Body.ExecutionPayload.BlockNumber),
				GasLimit:      fmt.Sprintf("%d", b.Body.ExecutionPayload.GasLimit),
				GasUsed:       fmt.Sprintf("%d", b.Body.ExecutionPayload.GasUsed),
				Timestamp:     fmt.Sprintf("%d", b.Body.ExecutionPayload.Timestamp),
				ExtraData:     hexutil.Encode(b.Body.ExecutionPayload.ExtraData),
				BaseFeePerGas: baseFeePerGas,
				BlockHash:     hexutil.Encode(b.Body.ExecutionPayload.BlockHash),
				Transactions:  transactions,
			},
		},
	}, nil
}

func SignedBeaconBlockBellatrixFromConsensus(b *eth.SignedBeaconBlockBellatrix) (*SignedBeaconBlockBellatrix, error) {
	block, err := BeaconBlockBellatrixFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &SignedBeaconBlockBellatrix{
		Message:   block,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BlindedBeaconBlockCapellaFromConsensus(b *eth.BlindedBeaconBlockCapella) (*BlindedBeaconBlockCapella, error) {
	proposerSlashings := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	attesterSlashings := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	atts, err := AttsFromConsensus(b.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsFromConsensus(b.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsFromConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	blsChanges, err := BlsChangesFromConsensus(b.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, err
	}
	e1d, err := Eth1DataFromConsensus(b.Body.Eth1Data)
	if err != nil {
		return nil, err
	}
	payload, err := ExecutionPayloadHeaderCapellaFromConsensus(b.Body.ExecutionPayloadHeader)
	if err != nil {
		return nil, err
	}

	return &BlindedBeaconBlockCapella{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BlindedBeaconBlockBodyCapella{
			RandaoReveal:      hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:          e1d,
			Graffiti:          hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &SyncAggregate{
				SyncCommitteeBits:      hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeBits),
				SyncCommitteeSignature: hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeSignature),
			},
			ExecutionPayloadHeader: payload,
			BlsToExecutionChanges:  blsChanges, // new in capella
		},
	}, nil
}

func SignedBlindedBeaconBlockCapellaFromConsensus(b *eth.SignedBlindedBeaconBlockCapella) (*SignedBlindedBeaconBlockCapella, error) {
	blindedBlock, err := BlindedBeaconBlockCapellaFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &SignedBlindedBeaconBlockCapella{
		Message:   blindedBlock,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BeaconBlockCapellaFromConsensus(b *eth.BeaconBlockCapella) (*BeaconBlockCapella, error) {
	proposerSlashings := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	attesterSlashings := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	atts, err := AttsFromConsensus(b.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsFromConsensus(b.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsFromConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	baseFeePerGas, err := sszBytesToUint256String(b.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, err
	}
	transactions := make([]string, len(b.Body.ExecutionPayload.Transactions))
	for i, tx := range b.Body.ExecutionPayload.Transactions {
		transactions[i] = hexutil.Encode(tx)
	}
	withdrawals := make([]*Withdrawal, len(b.Body.ExecutionPayload.Withdrawals))
	for i, w := range b.Body.ExecutionPayload.Withdrawals {
		withdrawals[i] = &Withdrawal{
			WithdrawalIndex:  fmt.Sprintf("%d", w.Index),
			ValidatorIndex:   fmt.Sprintf("%d", w.ValidatorIndex),
			ExecutionAddress: hexutil.Encode(w.Address),
			Amount:           fmt.Sprintf("%d", w.Amount),
		}
	}
	blsChanges, err := BlsChangesFromConsensus(b.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, err
	}
	e1d, err := Eth1DataFromConsensus(b.Body.Eth1Data)
	if err != nil {
		return nil, err
	}

	return &BeaconBlockCapella{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyCapella{
			RandaoReveal:      hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:          e1d,
			Graffiti:          hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &SyncAggregate{
				SyncCommitteeBits:      hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeBits),
				SyncCommitteeSignature: hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeSignature),
			},
			ExecutionPayload: &ExecutionPayloadCapella{
				ParentHash:    hexutil.Encode(b.Body.ExecutionPayload.ParentHash),
				FeeRecipient:  hexutil.Encode(b.Body.ExecutionPayload.FeeRecipient),
				StateRoot:     hexutil.Encode(b.Body.ExecutionPayload.StateRoot),
				ReceiptsRoot:  hexutil.Encode(b.Body.ExecutionPayload.ReceiptsRoot),
				LogsBloom:     hexutil.Encode(b.Body.ExecutionPayload.LogsBloom),
				PrevRandao:    hexutil.Encode(b.Body.ExecutionPayload.PrevRandao),
				BlockNumber:   fmt.Sprintf("%d", b.Body.ExecutionPayload.BlockNumber),
				GasLimit:      fmt.Sprintf("%d", b.Body.ExecutionPayload.GasLimit),
				GasUsed:       fmt.Sprintf("%d", b.Body.ExecutionPayload.GasUsed),
				Timestamp:     fmt.Sprintf("%d", b.Body.ExecutionPayload.Timestamp),
				ExtraData:     hexutil.Encode(b.Body.ExecutionPayload.ExtraData),
				BaseFeePerGas: baseFeePerGas,
				BlockHash:     hexutil.Encode(b.Body.ExecutionPayload.BlockHash),
				Transactions:  transactions,
				Withdrawals:   withdrawals, // new in capella
			},
			BlsToExecutionChanges: blsChanges, // new in capella
		},
	}, nil
}

func SignedBeaconBlockCapellaFromConsensus(b *eth.SignedBeaconBlockCapella) (*SignedBeaconBlockCapella, error) {
	block, err := BeaconBlockCapellaFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &SignedBeaconBlockCapella{
		Message:   block,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BlindedBeaconBlockContentsDenebFromConsensus(b *eth.BlindedBeaconBlockAndBlobsDeneb) (*BlindedBeaconBlockContentsDeneb, error) {
	var blindedBlobSidecars []*BlindedBlobSidecar
	if len(b.Blobs) != 0 {
		blindedBlobSidecars = make([]*BlindedBlobSidecar, len(b.Blobs))
		for i, s := range b.Blobs {
			signedBlob, err := BlindedBlobSidecarFromConsensus(s)
			if err != nil {
				return nil, err
			}
			blindedBlobSidecars[i] = signedBlob
		}
	}
	blindedBlock, err := BlindedBeaconBlockDenebFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &BlindedBeaconBlockContentsDeneb{
		BlindedBlock:        blindedBlock,
		BlindedBlobSidecars: blindedBlobSidecars,
	}, nil
}

func SignedBlindedBeaconBlockContentsDenebFromConsensus(b *eth.SignedBlindedBeaconBlockAndBlobsDeneb) (*SignedBlindedBeaconBlockContentsDeneb, error) {
	var blindedBlobSidecars []*SignedBlindedBlobSidecar
	if len(b.SignedBlindedBlobSidecars) != 0 {
		blindedBlobSidecars = make([]*SignedBlindedBlobSidecar, len(b.SignedBlindedBlobSidecars))
		for i, s := range b.SignedBlindedBlobSidecars {
			signedBlob, err := SignedBlindedBlobSidecarFromConsensus(s)
			if err != nil {
				return nil, err
			}
			blindedBlobSidecars[i] = signedBlob
		}
	}
	blindedBlock, err := SignedBlindedBeaconBlockDenebFromConsensus(b.SignedBlindedBlock)
	if err != nil {
		return nil, err
	}
	return &SignedBlindedBeaconBlockContentsDeneb{
		SignedBlindedBlock:        blindedBlock,
		SignedBlindedBlobSidecars: blindedBlobSidecars,
	}, nil
}

func BeaconBlockContentsDenebFromConsensus(b *eth.BeaconBlockAndBlobsDeneb) (*BeaconBlockContentsDeneb, error) {
	var blobSidecars []*BlobSidecar
	if len(b.Blobs) != 0 {
		blobSidecars = make([]*BlobSidecar, len(b.Blobs))
		for i, s := range b.Blobs {
			blob, err := BlobSidecarFromConsensus(s)
			if err != nil {
				return nil, err
			}
			blobSidecars[i] = blob
		}
	}
	block, err := BeaconBlockDenebFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &BeaconBlockContentsDeneb{
		Block:        block,
		BlobSidecars: blobSidecars,
	}, nil
}

func SignedBeaconBlockContentsDenebFromConsensus(b *eth.SignedBeaconBlockAndBlobsDeneb) (*SignedBeaconBlockContentsDeneb, error) {
	var blobSidecars []*SignedBlobSidecar
	if len(b.Blobs) != 0 {
		blobSidecars = make([]*SignedBlobSidecar, len(b.Blobs))
		for i, s := range b.Blobs {
			blob, err := SignedBlobSidecarFromConsensus(s)
			if err != nil {
				return nil, err
			}
			blobSidecars[i] = blob
		}
	}
	block, err := SignedBeaconBlockDenebFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &SignedBeaconBlockContentsDeneb{
		SignedBlock:        block,
		SignedBlobSidecars: blobSidecars,
	}, nil
}

func BlindedBeaconBlockDenebFromConsensus(b *eth.BlindedBeaconBlockDeneb) (*BlindedBeaconBlockDeneb, error) {
	proposerSlashings := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	attesterSlashings := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	atts, err := AttsFromConsensus(b.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsFromConsensus(b.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsFromConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	blsChanges, err := BlsChangesFromConsensus(b.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, err
	}
	blobKzgCommitments := make([]string, len(b.Body.BlobKzgCommitments))
	for i := range b.Body.BlobKzgCommitments {
		blobKzgCommitments[i] = hexutil.Encode(b.Body.BlobKzgCommitments[i])
	}
	e1d, err := Eth1DataFromConsensus(b.Body.Eth1Data)
	if err != nil {
		return nil, err
	}
	payload, err := ExecutionPayloadHeaderDenebFromConsensus(b.Body.ExecutionPayloadHeader)
	if err != nil {
		return nil, err
	}

	return &BlindedBeaconBlockDeneb{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BlindedBeaconBlockBodyDeneb{
			RandaoReveal:      hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:          e1d,
			Graffiti:          hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &SyncAggregate{
				SyncCommitteeBits:      hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeBits),
				SyncCommitteeSignature: hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeSignature),
			},
			ExecutionPayloadHeader: payload,
			BlsToExecutionChanges:  blsChanges,         // new in capella
			BlobKzgCommitments:     blobKzgCommitments, // new in deneb
		},
	}, nil
}

func SignedBlindedBeaconBlockDenebFromConsensus(b *eth.SignedBlindedBeaconBlockDeneb) (*SignedBlindedBeaconBlockDeneb, error) {
	block, err := BlindedBeaconBlockDenebFromConsensus(b.Message)
	if err != nil {
		return nil, err
	}
	return &SignedBlindedBeaconBlockDeneb{
		Message:   block,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BeaconBlockDenebFromConsensus(b *eth.BeaconBlockDeneb) (*BeaconBlockDeneb, error) {
	proposerSlashings := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	attesterSlashings := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	atts, err := AttsFromConsensus(b.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsFromConsensus(b.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsFromConsensus(b.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	baseFeePerGas, err := sszBytesToUint256String(b.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, err
	}
	transactions := make([]string, len(b.Body.ExecutionPayload.Transactions))
	for i, tx := range b.Body.ExecutionPayload.Transactions {
		transactions[i] = hexutil.Encode(tx)
	}
	withdrawals := make([]*Withdrawal, len(b.Body.ExecutionPayload.Withdrawals))
	for i, w := range b.Body.ExecutionPayload.Withdrawals {
		withdrawals[i] = &Withdrawal{
			WithdrawalIndex:  fmt.Sprintf("%d", w.Index),
			ValidatorIndex:   fmt.Sprintf("%d", w.ValidatorIndex),
			ExecutionAddress: hexutil.Encode(w.Address),
			Amount:           fmt.Sprintf("%d", w.Amount),
		}
	}
	blsChanges, err := BlsChangesFromConsensus(b.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, err
	}
	blobKzgCommitments := make([]string, len(b.Body.BlobKzgCommitments))
	for i := range b.Body.BlobKzgCommitments {
		blobKzgCommitments[i] = hexutil.Encode(b.Body.BlobKzgCommitments[i])
	}
	e1d, err := Eth1DataFromConsensus(b.Body.Eth1Data)
	if err != nil {
		return nil, err
	}

	return &BeaconBlockDeneb{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyDeneb{
			RandaoReveal:      hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data:          e1d,
			Graffiti:          hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
			SyncAggregate: &SyncAggregate{
				SyncCommitteeBits:      hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeBits),
				SyncCommitteeSignature: hexutil.Encode(b.Body.SyncAggregate.SyncCommitteeSignature),
			},
			ExecutionPayload: &ExecutionPayloadDeneb{
				ParentHash:    hexutil.Encode(b.Body.ExecutionPayload.ParentHash),
				FeeRecipient:  hexutil.Encode(b.Body.ExecutionPayload.FeeRecipient),
				StateRoot:     hexutil.Encode(b.Body.ExecutionPayload.StateRoot),
				ReceiptsRoot:  hexutil.Encode(b.Body.ExecutionPayload.ReceiptsRoot),
				LogsBloom:     hexutil.Encode(b.Body.ExecutionPayload.LogsBloom),
				PrevRandao:    hexutil.Encode(b.Body.ExecutionPayload.PrevRandao),
				BlockNumber:   fmt.Sprintf("%d", b.Body.ExecutionPayload.BlockNumber),
				GasLimit:      fmt.Sprintf("%d", b.Body.ExecutionPayload.GasLimit),
				GasUsed:       fmt.Sprintf("%d", b.Body.ExecutionPayload.GasUsed),
				Timestamp:     fmt.Sprintf("%d", b.Body.ExecutionPayload.Timestamp),
				ExtraData:     hexutil.Encode(b.Body.ExecutionPayload.ExtraData),
				BaseFeePerGas: baseFeePerGas,
				BlockHash:     hexutil.Encode(b.Body.ExecutionPayload.BlockHash),
				Transactions:  transactions,
				Withdrawals:   withdrawals,
				BlobGasUsed:   fmt.Sprintf("%d", b.Body.ExecutionPayload.BlobGasUsed),   // new in deneb TODO: rename to blob
				ExcessBlobGas: fmt.Sprintf("%d", b.Body.ExecutionPayload.ExcessBlobGas), // new in deneb TODO: rename to blob
			},
			BlsToExecutionChanges: blsChanges,         // new in capella
			BlobKzgCommitments:    blobKzgCommitments, // new in deneb
		},
	}, nil
}

func SignedBeaconBlockDenebFromConsensus(b *eth.SignedBeaconBlockDeneb) (*SignedBeaconBlockDeneb, error) {
	beaconBlock, err := BeaconBlockDenebFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &SignedBeaconBlockDeneb{
		Message:   beaconBlock,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BlindedBlobSidecarFromConsensus(b *eth.BlindedBlobSidecar) (*BlindedBlobSidecar, error) {
	return &BlindedBlobSidecar{
		BlockRoot:       hexutil.Encode(b.BlockRoot),
		Index:           fmt.Sprintf("%d", b.Index),
		Slot:            fmt.Sprintf("%d", b.Slot),
		BlockParentRoot: hexutil.Encode(b.BlockParentRoot),
		ProposerIndex:   fmt.Sprintf("%d", b.ProposerIndex),
		BlobRoot:        hexutil.Encode(b.BlobRoot),
		KzgCommitment:   hexutil.Encode(b.KzgCommitment),
		KzgProof:        hexutil.Encode(b.KzgProof),
	}, nil
}

func SignedBlindedBlobSidecarFromConsensus(b *eth.SignedBlindedBlobSidecar) (*SignedBlindedBlobSidecar, error) {
	blobSidecar, err := BlindedBlobSidecarFromConsensus(b.Message)
	if err != nil {
		return nil, err
	}
	return &SignedBlindedBlobSidecar{
		Message:   blobSidecar,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func BlobSidecarFromConsensus(b *eth.DeprecatedBlobSidecar) (*BlobSidecar, error) {
	return &BlobSidecar{
		BlockRoot:       hexutil.Encode(b.BlockRoot),
		Index:           fmt.Sprintf("%d", b.Index),
		Slot:            fmt.Sprintf("%d", b.Slot),
		BlockParentRoot: hexutil.Encode(b.BlockParentRoot),
		ProposerIndex:   fmt.Sprintf("%d", b.ProposerIndex),
		Blob:            hexutil.Encode(b.Blob),
		KzgCommitment:   hexutil.Encode(b.KzgCommitment),
		KzgProof:        hexutil.Encode(b.KzgProof),
	}, nil
}

func SignedBlobSidecarFromConsensus(b *eth.SignedBlobSidecar) (*SignedBlobSidecar, error) {
	blobSidecar, err := BlobSidecarFromConsensus(b.Message)
	if err != nil {
		return nil, err
	}
	return &SignedBlobSidecar{
		Message:   blobSidecar,
		Signature: hexutil.Encode(b.Signature),
	}, nil
}

func ProposerSlashingsToConsensus(src []*ProposerSlashing) ([]*eth.ProposerSlashing, error) {
	if src == nil {
		return nil, errNilValue
	}
	err := VerifyMaxLength(src, 16)
	if err != nil {
		return nil, err
	}
	proposerSlashings := make([]*eth.ProposerSlashing, len(src))
	for i, s := range src {
		if s == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d]", i))
		}
		if s.SignedHeader1 == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d].SignedHeader1", i))
		}
		if s.SignedHeader1.Message == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d].SignedHeader1.Message", i))
		}
		if s.SignedHeader2 == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d].SignedHeader2", i))
		}
		if s.SignedHeader2.Message == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d].SignedHeader2.Message", i))
		}

		h1Sig, err := DecodeHexWithLength(s.SignedHeader1.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader1.Signature", i))
		}
		h1Slot, err := strconv.ParseUint(s.SignedHeader1.Message.Slot, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader1.Message.Slot", i))
		}
		h1ProposerIndex, err := strconv.ParseUint(s.SignedHeader1.Message.ProposerIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader1.Message.ProposerIndex", i))
		}
		h1ParentRoot, err := DecodeHexWithLength(s.SignedHeader1.Message.ParentRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader1.Message.ParentRoot", i))
		}
		h1StateRoot, err := DecodeHexWithLength(s.SignedHeader1.Message.StateRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader1.Message.StateRoot", i))
		}
		h1BodyRoot, err := DecodeHexWithLength(s.SignedHeader1.Message.BodyRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader1.Message.BodyRoot", i))
		}
		h2Sig, err := DecodeHexWithLength(s.SignedHeader2.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader2.Signature", i))
		}
		h2Slot, err := strconv.ParseUint(s.SignedHeader2.Message.Slot, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader2.Message.Slot", i))
		}
		h2ProposerIndex, err := strconv.ParseUint(s.SignedHeader2.Message.ProposerIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader2.Message.ProposerIndex", i))
		}
		h2ParentRoot, err := DecodeHexWithLength(s.SignedHeader2.Message.ParentRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader2.Message.ParentRoot", i))
		}
		h2StateRoot, err := DecodeHexWithLength(s.SignedHeader2.Message.StateRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader2.Message.StateRoot", i))
		}
		h2BodyRoot, err := DecodeHexWithLength(s.SignedHeader2.Message.BodyRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].SignedHeader2.Message.BodyRoot", i))
		}
		proposerSlashings[i] = &eth.ProposerSlashing{
			Header_1: &eth.SignedBeaconBlockHeader{
				Header: &eth.BeaconBlockHeader{
					Slot:          primitives.Slot(h1Slot),
					ProposerIndex: primitives.ValidatorIndex(h1ProposerIndex),
					ParentRoot:    h1ParentRoot,
					StateRoot:     h1StateRoot,
					BodyRoot:      h1BodyRoot,
				},
				Signature: h1Sig,
			},
			Header_2: &eth.SignedBeaconBlockHeader{
				Header: &eth.BeaconBlockHeader{
					Slot:          primitives.Slot(h2Slot),
					ProposerIndex: primitives.ValidatorIndex(h2ProposerIndex),
					ParentRoot:    h2ParentRoot,
					StateRoot:     h2StateRoot,
					BodyRoot:      h2BodyRoot,
				},
				Signature: h2Sig,
			},
		}
	}
	return proposerSlashings, nil
}

func ProposerSlashingsFromConsensus(src []*eth.ProposerSlashing) []*ProposerSlashing {
	proposerSlashings := make([]*ProposerSlashing, len(src))
	for i, s := range src {
		proposerSlashings[i] = &ProposerSlashing{
			SignedHeader1: &SignedBeaconBlockHeader{
				Message: &BeaconBlockHeader{
					Slot:          fmt.Sprintf("%d", s.Header_1.Header.Slot),
					ProposerIndex: fmt.Sprintf("%d", s.Header_1.Header.ProposerIndex),
					ParentRoot:    hexutil.Encode(s.Header_1.Header.ParentRoot),
					StateRoot:     hexutil.Encode(s.Header_1.Header.StateRoot),
					BodyRoot:      hexutil.Encode(s.Header_1.Header.BodyRoot),
				},
				Signature: hexutil.Encode(s.Header_1.Signature),
			},
			SignedHeader2: &SignedBeaconBlockHeader{
				Message: &BeaconBlockHeader{
					Slot:          fmt.Sprintf("%d", s.Header_2.Header.Slot),
					ProposerIndex: fmt.Sprintf("%d", s.Header_2.Header.ProposerIndex),
					ParentRoot:    hexutil.Encode(s.Header_2.Header.ParentRoot),
					StateRoot:     hexutil.Encode(s.Header_2.Header.StateRoot),
					BodyRoot:      hexutil.Encode(s.Header_2.Header.BodyRoot),
				},
				Signature: hexutil.Encode(s.Header_2.Signature),
			},
		}
	}
	return proposerSlashings
}

func AttesterSlashingsToConsensus(src []*AttesterSlashing) ([]*eth.AttesterSlashing, error) {
	if src == nil {
		return nil, errNilValue
	}
	err := VerifyMaxLength(src, 2)
	if err != nil {
		return nil, err
	}

	attesterSlashings := make([]*eth.AttesterSlashing, len(src))
	for i, s := range src {
		if s == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d]", i))
		}
		if s.Attestation1 == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d].Attestation1", i))
		}
		if s.Attestation2 == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d].Attestation2", i))
		}

		a1Sig, err := DecodeHexWithLength(s.Attestation1.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Attestation1.Signature", i))
		}
		err = VerifyMaxLength(s.Attestation1.AttestingIndices, 2048)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Attestation1.AttestingIndices", i))
		}
		a1AttestingIndices := make([]uint64, len(s.Attestation1.AttestingIndices))
		for j, ix := range s.Attestation1.AttestingIndices {
			attestingIndex, err := strconv.ParseUint(ix, 10, 64)
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("[%d].Attestation1.AttestingIndices[%d]", i, j))
			}
			a1AttestingIndices[j] = attestingIndex
		}
		a1Data, err := s.Attestation1.Data.ToConsensus()
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Attestation1.Data", i))
		}
		a2Sig, err := DecodeHexWithLength(s.Attestation2.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Attestation2.Signature", i))
		}
		err = VerifyMaxLength(s.Attestation2.AttestingIndices, 2048)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Attestation2.AttestingIndices", i))
		}
		a2AttestingIndices := make([]uint64, len(s.Attestation2.AttestingIndices))
		for j, ix := range s.Attestation2.AttestingIndices {
			attestingIndex, err := strconv.ParseUint(ix, 10, 64)
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("[%d].Attestation2.AttestingIndices[%d]", i, j))
			}
			a2AttestingIndices[j] = attestingIndex
		}
		a2Data, err := s.Attestation2.Data.ToConsensus()
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Attestation2.Data", i))
		}
		attesterSlashings[i] = &eth.AttesterSlashing{
			Attestation_1: &eth.IndexedAttestation{
				AttestingIndices: a1AttestingIndices,
				Data:             a1Data,
				Signature:        a1Sig,
			},
			Attestation_2: &eth.IndexedAttestation{
				AttestingIndices: a2AttestingIndices,
				Data:             a2Data,
				Signature:        a2Sig,
			},
		}
	}
	return attesterSlashings, nil
}

func AttesterSlashingsFromConsensus(src []*eth.AttesterSlashing) []*AttesterSlashing {
	attesterSlashings := make([]*AttesterSlashing, len(src))
	for i, s := range src {
		a1AttestingIndices := make([]string, len(s.Attestation_1.AttestingIndices))
		for j, ix := range s.Attestation_1.AttestingIndices {
			a1AttestingIndices[j] = fmt.Sprintf("%d", ix)
		}
		a2AttestingIndices := make([]string, len(s.Attestation_2.AttestingIndices))
		for j, ix := range s.Attestation_2.AttestingIndices {
			a2AttestingIndices[j] = fmt.Sprintf("%d", ix)
		}
		attesterSlashings[i] = &AttesterSlashing{
			Attestation1: &IndexedAttestation{
				AttestingIndices: a1AttestingIndices,
				Data: &AttestationData{
					Slot:            fmt.Sprintf("%d", s.Attestation_1.Data.Slot),
					CommitteeIndex:  fmt.Sprintf("%d", s.Attestation_1.Data.CommitteeIndex),
					BeaconBlockRoot: hexutil.Encode(s.Attestation_1.Data.BeaconBlockRoot),
					Source: &Checkpoint{
						Epoch: fmt.Sprintf("%d", s.Attestation_1.Data.Source.Epoch),
						Root:  hexutil.Encode(s.Attestation_1.Data.Source.Root),
					},
					Target: &Checkpoint{
						Epoch: fmt.Sprintf("%d", s.Attestation_1.Data.Target.Epoch),
						Root:  hexutil.Encode(s.Attestation_1.Data.Target.Root),
					},
				},
				Signature: hexutil.Encode(s.Attestation_1.Signature),
			},
			Attestation2: &IndexedAttestation{
				AttestingIndices: a2AttestingIndices,
				Data: &AttestationData{
					Slot:            fmt.Sprintf("%d", s.Attestation_2.Data.Slot),
					CommitteeIndex:  fmt.Sprintf("%d", s.Attestation_2.Data.CommitteeIndex),
					BeaconBlockRoot: hexutil.Encode(s.Attestation_2.Data.BeaconBlockRoot),
					Source: &Checkpoint{
						Epoch: fmt.Sprintf("%d", s.Attestation_2.Data.Source.Epoch),
						Root:  hexutil.Encode(s.Attestation_2.Data.Source.Root),
					},
					Target: &Checkpoint{
						Epoch: fmt.Sprintf("%d", s.Attestation_2.Data.Target.Epoch),
						Root:  hexutil.Encode(s.Attestation_2.Data.Target.Root),
					},
				},
				Signature: hexutil.Encode(s.Attestation_2.Signature),
			},
		}
	}
	return attesterSlashings
}

func AttsToConsensus(src []*Attestation) ([]*eth.Attestation, error) {
	if src == nil {
		return nil, errNilValue
	}
	err := VerifyMaxLength(src, 128)
	if err != nil {
		return nil, err
	}

	atts := make([]*eth.Attestation, len(src))
	for i, a := range src {
		atts[i], err = a.ToConsensus()
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d]", i))
		}
	}
	return atts, nil
}

func AttsFromConsensus(src []*eth.Attestation) ([]*Attestation, error) {
	atts := make([]*Attestation, len(src))
	for i, a := range src {
		atts[i] = AttestationFromConsensus(a)
	}
	return atts, nil
}

func DepositsToConsensus(src []*Deposit) ([]*eth.Deposit, error) {
	if src == nil {
		return nil, errNilValue
	}
	err := VerifyMaxLength(src, 16)
	if err != nil {
		return nil, err
	}

	deposits := make([]*eth.Deposit, len(src))
	for i, d := range src {
		if d.Data == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d].Data", i))
		}

		err = VerifyMaxLength(d.Proof, 33)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Proof", i))
		}
		proof := make([][]byte, len(d.Proof))
		for j, p := range d.Proof {
			var err error
			proof[j], err = DecodeHexWithLength(p, fieldparams.RootLength)
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("[%d].Proof[%d]", i, j))
			}
		}
		pubkey, err := DecodeHexWithLength(d.Data.Pubkey, fieldparams.BLSPubkeyLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Pubkey", i))
		}
		withdrawalCreds, err := DecodeHexWithLength(d.Data.WithdrawalCredentials, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].WithdrawalCredentials", i))
		}
		amount, err := strconv.ParseUint(d.Data.Amount, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Amount", i))
		}
		sig, err := DecodeHexWithLength(d.Data.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Signature", i))
		}
		deposits[i] = &eth.Deposit{
			Proof: proof,
			Data: &eth.Deposit_Data{
				PublicKey:             pubkey,
				WithdrawalCredentials: withdrawalCreds,
				Amount:                amount,
				Signature:             sig,
			},
		}
	}
	return deposits, nil
}

func DepositsFromConsensus(src []*eth.Deposit) ([]*Deposit, error) {
	deposits := make([]*Deposit, len(src))
	for i, d := range src {
		proof := make([]string, len(d.Proof))
		for j, p := range d.Proof {
			proof[j] = hexutil.Encode(p)
		}
		deposits[i] = &Deposit{
			Proof: proof,
			Data: &DepositData{
				Pubkey:                hexutil.Encode(d.Data.PublicKey),
				WithdrawalCredentials: hexutil.Encode(d.Data.WithdrawalCredentials),
				Amount:                fmt.Sprintf("%d", d.Data.Amount),
				Signature:             hexutil.Encode(d.Data.Signature),
			},
		}
	}
	return deposits, nil
}

func ExitsToConsensus(src []*SignedVoluntaryExit) ([]*eth.SignedVoluntaryExit, error) {
	if src == nil {
		return nil, errNilValue
	}
	err := VerifyMaxLength(src, 16)
	if err != nil {
		return nil, err
	}

	exits := make([]*eth.SignedVoluntaryExit, len(src))
	for i, e := range src {
		exits[i], err = e.ToConsensus()
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d]", i))
		}
	}
	return exits, nil
}

func ExitsFromConsensus(src []*eth.SignedVoluntaryExit) ([]*SignedVoluntaryExit, error) {
	exits := make([]*SignedVoluntaryExit, len(src))
	for i, e := range src {
		exits[i] = &SignedVoluntaryExit{
			Message: &VoluntaryExit{
				Epoch:          fmt.Sprintf("%d", e.Exit.Epoch),
				ValidatorIndex: fmt.Sprintf("%d", e.Exit.ValidatorIndex),
			},
			Signature: hexutil.Encode(e.Signature),
		}
	}
	return exits, nil
}

func BlsChangesToConsensus(src []*SignedBlsToExecutionChange) ([]*eth.SignedBLSToExecutionChange, error) {
	if src == nil {
		return nil, errNilValue
	}
	err := VerifyMaxLength(src, 16)
	if err != nil {
		return nil, err
	}

	changes := make([]*eth.SignedBLSToExecutionChange, len(src))
	for i, ch := range src {
		if ch.Message == nil {
			return nil, NewDecodeError(errNilValue, fmt.Sprintf("[%d]", i))
		}

		sig, err := DecodeHexWithLength(ch.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Signature", i))
		}
		index, err := strconv.ParseUint(ch.Message.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Message.ValidatorIndex", i))
		}
		pubkey, err := DecodeHexWithLength(ch.Message.FromBlsPubkey, fieldparams.BLSPubkeyLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Message.FromBlsPubkey", i))
		}
		address, err := DecodeHexWithLength(ch.Message.ToExecutionAddress, common.AddressLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("[%d].Message.ToExecutionAddress", i))
		}
		changes[i] = &eth.SignedBLSToExecutionChange{
			Message: &eth.BLSToExecutionChange{
				ValidatorIndex:     primitives.ValidatorIndex(index),
				FromBlsPubkey:      pubkey,
				ToExecutionAddress: address,
			},
			Signature: sig,
		}
	}
	return changes, nil
}

func BlsChangesFromConsensus(src []*eth.SignedBLSToExecutionChange) ([]*SignedBlsToExecutionChange, error) {
	changes := make([]*SignedBlsToExecutionChange, len(src))
	for i, ch := range src {
		changes[i] = &SignedBlsToExecutionChange{
			Message: &BlsToExecutionChange{
				ValidatorIndex:     fmt.Sprintf("%d", ch.Message.ValidatorIndex),
				FromBlsPubkey:      hexutil.Encode(ch.Message.FromBlsPubkey),
				ToExecutionAddress: hexutil.Encode(ch.Message.ToExecutionAddress),
			},
			Signature: hexutil.Encode(ch.Signature),
		}
	}
	return changes, nil
}

func ExecutionPayloadHeaderFromConsensus(payload *enginev1.ExecutionPayloadHeader) (*ExecutionPayloadHeader, error) {
	if payload == nil {
		return nil, errors.New("payload header is empty")
	}

	baseFeePerGas, err := sszBytesToUint256String(payload.BaseFeePerGas)
	if err != nil {
		return nil, err
	}

	return &ExecutionPayloadHeader{
		ParentHash:       hexutil.Encode(payload.ParentHash),
		FeeRecipient:     hexutil.Encode(payload.FeeRecipient),
		StateRoot:        hexutil.Encode(payload.StateRoot),
		ReceiptsRoot:     hexutil.Encode(payload.ReceiptsRoot),
		LogsBloom:        hexutil.Encode(payload.LogsBloom),
		PrevRandao:       hexutil.Encode(payload.PrevRandao),
		BlockNumber:      fmt.Sprintf("%d", payload.BlockNumber),
		GasLimit:         fmt.Sprintf("%d", payload.GasLimit),
		GasUsed:          fmt.Sprintf("%d", payload.GasUsed),
		Timestamp:        fmt.Sprintf("%d", payload.Timestamp),
		ExtraData:        hexutil.Encode(payload.ExtraData),
		BaseFeePerGas:    baseFeePerGas,
		BlockHash:        hexutil.Encode(payload.BlockHash),
		TransactionsRoot: hexutil.Encode(payload.TransactionsRoot),
	}, nil
}

func ExecutionPayloadHeaderCapellaFromConsensus(payload *enginev1.ExecutionPayloadHeaderCapella) (*ExecutionPayloadHeaderCapella, error) {
	if payload == nil {
		return nil, errors.New("payload header is empty")
	}

	baseFeePerGas, err := sszBytesToUint256String(payload.BaseFeePerGas)
	if err != nil {
		return nil, err
	}

	return &ExecutionPayloadHeaderCapella{
		ParentHash:       hexutil.Encode(payload.ParentHash),
		FeeRecipient:     hexutil.Encode(payload.FeeRecipient),
		StateRoot:        hexutil.Encode(payload.StateRoot),
		ReceiptsRoot:     hexutil.Encode(payload.ReceiptsRoot),
		LogsBloom:        hexutil.Encode(payload.LogsBloom),
		PrevRandao:       hexutil.Encode(payload.PrevRandao),
		BlockNumber:      fmt.Sprintf("%d", payload.BlockNumber),
		GasLimit:         fmt.Sprintf("%d", payload.GasLimit),
		GasUsed:          fmt.Sprintf("%d", payload.GasUsed),
		Timestamp:        fmt.Sprintf("%d", payload.Timestamp),
		ExtraData:        hexutil.Encode(payload.ExtraData),
		BaseFeePerGas:    baseFeePerGas,
		BlockHash:        hexutil.Encode(payload.BlockHash),
		TransactionsRoot: hexutil.Encode(payload.TransactionsRoot),
		WithdrawalsRoot:  hexutil.Encode(payload.WithdrawalsRoot),
	}, nil
}

func ExecutionPayloadHeaderDenebFromConsensus(payload *enginev1.ExecutionPayloadHeaderDeneb) (*ExecutionPayloadHeaderDeneb, error) {
	if payload == nil {
		return nil, errors.New("payload header is empty")
	}

	baseFeePerGas, err := sszBytesToUint256String(payload.BaseFeePerGas)
	if err != nil {
		return nil, err
	}

	return &ExecutionPayloadHeaderDeneb{
		ParentHash:       hexutil.Encode(payload.ParentHash),
		FeeRecipient:     hexutil.Encode(payload.FeeRecipient),
		StateRoot:        hexutil.Encode(payload.StateRoot),
		ReceiptsRoot:     hexutil.Encode(payload.ReceiptsRoot),
		LogsBloom:        hexutil.Encode(payload.LogsBloom),
		PrevRandao:       hexutil.Encode(payload.PrevRandao),
		BlockNumber:      fmt.Sprintf("%d", payload.BlockNumber),
		GasLimit:         fmt.Sprintf("%d", payload.GasLimit),
		GasUsed:          fmt.Sprintf("%d", payload.GasUsed),
		Timestamp:        fmt.Sprintf("%d", payload.Timestamp),
		ExtraData:        hexutil.Encode(payload.ExtraData),
		BaseFeePerGas:    baseFeePerGas,
		BlobGasUsed:      fmt.Sprintf("%d", payload.BlobGasUsed),   // new in deneb TODO: rename to blob
		ExcessBlobGas:    fmt.Sprintf("%d", payload.ExcessBlobGas), // new in deneb TODO: rename to blob
		BlockHash:        hexutil.Encode(payload.BlockHash),
		TransactionsRoot: hexutil.Encode(payload.TransactionsRoot),
		WithdrawalsRoot:  hexutil.Encode(payload.WithdrawalsRoot),
	}, nil
}

func Uint256ToSSZBytes(num string) ([]byte, error) {
	uint256, ok := new(big.Int).SetString(num, 10)
	if !ok {
		return nil, errors.New("could not parse Uint256")
	}
	if !math.IsValidUint256(uint256) {
		return nil, fmt.Errorf("%s is not a valid Uint256", num)
	}
	return bytesutil2.PadTo(bytesutil2.ReverseByteOrder(uint256.Bytes()), 32), nil
}

func sszBytesToUint256String(b []byte) (string, error) {
	bi := bytesutil2.LittleEndianBytesToBigInt(b)
	if !math.IsValidUint256(bi) {
		return "", fmt.Errorf("%s is not a valid Uint256", bi.String())
	}
	return string(bi.String()), nil
}
