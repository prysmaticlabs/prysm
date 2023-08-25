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

func (phase0SignedBeaconBlock *SignedBeaconBlock) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := DecodeHexWithLength(phase0SignedBeaconBlock.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}

	bl, err := phase0SignedBeaconBlock.Message.ToConsensus()
	if err != nil {
		return nil, err
	}

	block := &eth.SignedBeaconBlock{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Phase0{Phase0: block}}, nil
}

func (phase0BeaconBlock *BeaconBlock) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := phase0BeaconBlock.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Phase0{Phase0: block}}, nil
}

func (phase0BeaconBlock *BeaconBlock) ToConsensus() (*eth.BeaconBlock, error) {
	slot, err := strconv.ParseUint(phase0BeaconBlock.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(phase0BeaconBlock.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(phase0BeaconBlock.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(phase0BeaconBlock.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(phase0BeaconBlock.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(phase0BeaconBlock.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "could not decode phase0BeaconBlock.Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(phase0BeaconBlock.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(phase0BeaconBlock.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(phase0BeaconBlock.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(phase0BeaconBlock.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(phase0BeaconBlock.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := AttsToConsensus(phase0BeaconBlock.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsToConsensus(phase0BeaconBlock.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsToConsensus(phase0BeaconBlock.Body.VoluntaryExits)
	if err != nil {
		return nil, err
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

func (altairSignedBeaconBlock *SignedBeaconBlockAltair) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := DecodeHexWithLength(altairSignedBeaconBlock.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	bl, err := altairSignedBeaconBlock.Message.ToConsensus()
	if err != nil {
		return nil, err
	}
	block := &eth.SignedBeaconBlockAltair{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Altair{Altair: block}}, nil
}

func (altairBeaconBlock *BeaconBlockAltair) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := altairBeaconBlock.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Altair{Altair: block}}, nil
}

func (altairBeaconBlock *BeaconBlockAltair) ToConsensus() (*eth.BeaconBlockAltair, error) {
	slot, err := strconv.ParseUint(altairBeaconBlock.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(altairBeaconBlock.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(altairBeaconBlock.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(altairBeaconBlock.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(altairBeaconBlock.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(altairBeaconBlock.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(altairBeaconBlock.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(altairBeaconBlock.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(altairBeaconBlock.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(altairBeaconBlock.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(altairBeaconBlock.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := AttsToConsensus(altairBeaconBlock.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsToConsensus(altairBeaconBlock.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsToConsensus(altairBeaconBlock.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := DecodeHexWithLength(altairBeaconBlock.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(altairBeaconBlock.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
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

func (bellatrixSignedBeaconBlock *SignedBeaconBlockBellatrix) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := DecodeHexWithLength(bellatrixSignedBeaconBlock.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, ".Signature")
	}
	bl, err := bellatrixSignedBeaconBlock.Message.ToConsensus()
	if err != nil {
		return nil, err
	}
	block := &eth.SignedBeaconBlockBellatrix{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Bellatrix{Bellatrix: block}}, nil
}

func (bellatrixBeaconBlock *BeaconBlockBellatrix) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := bellatrixBeaconBlock.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Bellatrix{Bellatrix: block}}, nil
}

func (bellatrixBeaconBlock *BeaconBlockBellatrix) ToConsensus() (*eth.BeaconBlockBellatrix, error) {
	slot, err := strconv.ParseUint(bellatrixBeaconBlock.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(bellatrixBeaconBlock.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(bellatrixBeaconBlock.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(bellatrixBeaconBlock.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(bellatrixBeaconBlock.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(bellatrixBeaconBlock.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(bellatrixBeaconBlock.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := AttsToConsensus(bellatrixBeaconBlock.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsToConsensus(bellatrixBeaconBlock.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsToConsensus(bellatrixBeaconBlock.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.ExecutionPayload.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.ExecutionPayload.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.ExecutionPayload.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.ExecutionPayload.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.ExecutionPayload.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.ExecutionPayload.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(bellatrixBeaconBlock.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(bellatrixBeaconBlock.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(bellatrixBeaconBlock.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(bellatrixBeaconBlock.Body.ExecutionPayload.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(bellatrixBeaconBlock.Body.ExecutionPayload.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToSSZBytes(bellatrixBeaconBlock.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(bellatrixBeaconBlock.Body.ExecutionPayload.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockHash")
	}
	err = VerifyMaxLength("signedBlock.Body.ExecutionPayload.Transactions", len(bellatrixBeaconBlock.Body.ExecutionPayload.Transactions), fieldparams.MaxTxsPerPayloadLength)
	if err != nil {
		return nil, err
	}
	payloadTxs := make([][]byte, len(bellatrixBeaconBlock.Body.ExecutionPayload.Transactions))
	for i, tx := range bellatrixBeaconBlock.Body.ExecutionPayload.Transactions {
		payloadTxs[i], err = DecodeHexWithMaxLength(tx, fieldparams.MaxBytesPerTxLength)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode bellatrixBeaconBlock.Body.ExecutionPayload.Transactions[%d]", i)
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

func (bellatrixSignedBlindedBeaconBlock *SignedBlindedBeaconBlockBellatrix) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := DecodeHexWithLength(bellatrixSignedBlindedBeaconBlock.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixSignedBlindedBeaconBlock.Signature")
	}
	bl, err := bellatrixSignedBlindedBeaconBlock.Message.ToConsensus()
	if err != nil {
		return nil, err
	}
	block := &eth.SignedBlindedBeaconBlockBellatrix{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (bellatrixBlindedBeaconBlock *BlindedBeaconBlockBellatrix) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := bellatrixBlindedBeaconBlock.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (bellatrixBlindedBeaconBlock *BlindedBeaconBlockBellatrix) ToConsensus() (*eth.BlindedBeaconBlockBellatrix, error) {
	slot, err := strconv.ParseUint(bellatrixBlindedBeaconBlock.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Slot")
	}
	proposerIndex, err := strconv.ParseUint(bellatrixBlindedBeaconBlock.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(bellatrixBlindedBeaconBlock.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(bellatrixBlindedBeaconBlock.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(bellatrixBlindedBeaconBlock.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := AttsToConsensus(bellatrixBlindedBeaconBlock.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsToConsensus(bellatrixBlindedBeaconBlock.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsToConsensus(bellatrixBlindedBeaconBlock.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.ParentHash, common.HashLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.BlockNumber, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.GasLimit, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.GasUsed, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.Timestamp, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToSSZBytes(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.BlockHash, common.HashLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.BlockHash")
	}
	payloadTxsRoot, err := DecodeHexWithLength(bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.TransactionsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode bellatrixBlindedBeaconBlock.Body.ExecutionPayloadHeader.TransactionsRoot")
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

func (capellaSignedBeaconBlock *SignedBeaconBlockCapella) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := DecodeHexWithLength(capellaSignedBeaconBlock.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	bl, err := capellaSignedBeaconBlock.Message.ToConsensus()
	if err != nil {
		return nil, err
	}
	block := &eth.SignedBeaconBlockCapella{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Capella{Capella: block}}, nil
}

func (capellaBeaconBlock *BeaconBlockCapella) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := capellaBeaconBlock.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Capella{Capella: block}}, nil
}

func (capellaBeaconBlock *BeaconBlockCapella) ToConsensus() (*eth.BeaconBlockCapella, error) {
	slot, err := strconv.ParseUint(capellaBeaconBlock.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(capellaBeaconBlock.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(capellaBeaconBlock.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(capellaBeaconBlock.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(capellaBeaconBlock.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(capellaBeaconBlock.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(capellaBeaconBlock.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(capellaBeaconBlock.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(capellaBeaconBlock.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(capellaBeaconBlock.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(capellaBeaconBlock.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := AttsToConsensus(capellaBeaconBlock.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsToConsensus(capellaBeaconBlock.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsToConsensus(capellaBeaconBlock.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := DecodeHexWithLength(capellaBeaconBlock.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(capellaBeaconBlock.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(capellaBeaconBlock.Body.ExecutionPayload.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(capellaBeaconBlock.Body.ExecutionPayload.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(capellaBeaconBlock.Body.ExecutionPayload.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(capellaBeaconBlock.Body.ExecutionPayload.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(capellaBeaconBlock.Body.ExecutionPayload.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(capellaBeaconBlock.Body.ExecutionPayload.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(capellaBeaconBlock.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(capellaBeaconBlock.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(capellaBeaconBlock.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(capellaBeaconBlock.Body.ExecutionPayload.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(capellaBeaconBlock.Body.ExecutionPayload.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToSSZBytes(capellaBeaconBlock.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(capellaBeaconBlock.Body.ExecutionPayload.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockHash")
	}
	err = VerifyMaxLength("signedBlock.Body.ExecutionPayload.Transactions", len(capellaBeaconBlock.Body.ExecutionPayload.Transactions), fieldparams.MaxTxsPerPayloadLength)
	if err != nil {
		return nil, err
	}
	payloadTxs := make([][]byte, len(capellaBeaconBlock.Body.ExecutionPayload.Transactions))
	for i, tx := range capellaBeaconBlock.Body.ExecutionPayload.Transactions {
		payloadTxs[i], err = DecodeHexWithMaxLength(tx, fieldparams.MaxBytesPerTxLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ExecutionPayload.Transactions[%d]", i))
		}
	}
	err = VerifyMaxLength("signedBlock.Body.ExecutionPayload.Withdrawals", len(capellaBeaconBlock.Body.ExecutionPayload.Withdrawals), fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return nil, err
	}
	withdrawals := make([]*enginev1.Withdrawal, len(capellaBeaconBlock.Body.ExecutionPayload.Withdrawals))
	for i, w := range capellaBeaconBlock.Body.ExecutionPayload.Withdrawals {
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
	blsChanges, err := BlsChangesToConsensus(capellaBeaconBlock.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, err
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

func (capellaSignedBlindedBeaconBlock *SignedBlindedBeaconBlockCapella) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	sig, err := DecodeHexWithLength(capellaSignedBlindedBeaconBlock.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	bl, err := capellaSignedBlindedBeaconBlock.Message.ToConsensus()
	if err != nil {
		return nil, err
	}
	block := &eth.SignedBlindedBeaconBlockCapella{
		Block:     bl,
		Signature: sig,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_BlindedCapella{BlindedCapella: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (capellaBlindedBeaconBLock *BlindedBeaconBlockCapella) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := capellaBlindedBeaconBLock.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_BlindedCapella{BlindedCapella: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (capellaBlindedBeaconBLock *BlindedBeaconBlockCapella) ToConsensus() (*eth.BlindedBeaconBlockCapella, error) {
	slot, err := strconv.ParseUint(capellaBlindedBeaconBLock.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(capellaBlindedBeaconBLock.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(capellaBlindedBeaconBLock.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(capellaBlindedBeaconBLock.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(capellaBlindedBeaconBLock.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(capellaBlindedBeaconBLock.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(capellaBlindedBeaconBLock.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := AttsToConsensus(capellaBlindedBeaconBLock.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsToConsensus(capellaBlindedBeaconBLock.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsToConsensus(capellaBlindedBeaconBLock.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToSSZBytes(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithMaxLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockHash")
	}
	payloadTxsRoot, err := DecodeHexWithMaxLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.TransactionsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.TransactionsRoot")
	}
	payloadWithdrawalsRoot, err := DecodeHexWithMaxLength(capellaBlindedBeaconBLock.Body.ExecutionPayloadHeader.WithdrawalsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.WithdrawalsRoot")
	}
	blsChanges, err := BlsChangesToConsensus(capellaBlindedBeaconBLock.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, err
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

func (denebSignedBeaconBlockContents *SignedBeaconBlockContentsDeneb) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	var signedBlobSidecars []*eth.SignedBlobSidecar
	if len(denebSignedBeaconBlockContents.SignedBlobSidecars) != 0 {
		err := VerifyMaxLength("denebSignedBeaconBlockContents.SignedBlobSidecars", len(denebSignedBeaconBlockContents.SignedBlobSidecars), fieldparams.MaxBlobsPerBlock)
		if err != nil {
			return nil, err
		}
		signedBlobSidecars = make([]*eth.SignedBlobSidecar, len(denebSignedBeaconBlockContents.SignedBlobSidecars))
		for i := range denebSignedBeaconBlockContents.SignedBlobSidecars {
			signedBlob, err := denebSignedBeaconBlockContents.SignedBlobSidecars[i].ToConsensus(i)
			if err != nil {
				return nil, err
			}
			signedBlobSidecars[i] = signedBlob
		}
	}
	signedDenebBlock, err := denebSignedBeaconBlockContents.SignedBlock.ToConsensus()
	if err != nil {
		return nil, err
	}
	block := &eth.SignedBeaconBlockAndBlobsDeneb{
		Block: signedDenebBlock,
		Blobs: signedBlobSidecars,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_Deneb{Deneb: block}}, nil
}

func (denebSignedBeaconBlockContents *SignedBeaconBlockContentsDeneb) ToUnsigned() *BeaconBlockContentsDeneb {
	var blobSidecars []*BlobSidecar
	if len(denebSignedBeaconBlockContents.SignedBlobSidecars) != 0 {
		blobSidecars = make([]*BlobSidecar, len(denebSignedBeaconBlockContents.SignedBlobSidecars))
		for i, s := range denebSignedBeaconBlockContents.SignedBlobSidecars {
			blobSidecars[i] = s.Message
		}
	}
	return &BeaconBlockContentsDeneb{
		Block:        denebSignedBeaconBlockContents.SignedBlock.Message,
		BlobSidecars: blobSidecars,
	}
}

func (denebBeaconBlockContents *BeaconBlockContentsDeneb) ToGeneric() (*eth.GenericBeaconBlock, error) {
	block, err := denebBeaconBlockContents.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_Deneb{Deneb: block}}, nil
}

func (denebBeaconBlockContents *BeaconBlockContentsDeneb) ToConsensus() (*eth.BeaconBlockAndBlobsDeneb, error) {
	var blobSidecars []*eth.BlobSidecar
	if len(denebBeaconBlockContents.BlobSidecars) != 0 {
		err := VerifyMaxLength("denebBeaconBlockContents.BlobSidecars", len(denebBeaconBlockContents.BlobSidecars), fieldparams.MaxBlobsPerBlock)
		if err != nil {
			return nil, err
		}
		blobSidecars = make([]*eth.BlobSidecar, len(denebBeaconBlockContents.BlobSidecars))
		for i := range denebBeaconBlockContents.BlobSidecars {
			blob, err := denebBeaconBlockContents.BlobSidecars[i].ToConsensus(i)
			if err != nil {
				return nil, err
			}
			blobSidecars[i] = blob
		}
	}
	denebBlock, err := denebBeaconBlockContents.Block.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.BeaconBlockAndBlobsDeneb{
		Block: denebBlock,
		Blobs: blobSidecars,
	}, nil
}

func (denebSignedBlindedBeaconBlockContents *SignedBlindedBeaconBlockContentsDeneb) ToGeneric() (*eth.GenericSignedBeaconBlock, error) {
	var signedBlindedBlobSidecars []*eth.SignedBlindedBlobSidecar
	if len(denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars) != 0 {
		err := VerifyMaxLength("denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars", len(denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars), fieldparams.MaxBlobsPerBlock)
		if err != nil {
			return nil, err
		}
		signedBlindedBlobSidecars = make([]*eth.SignedBlindedBlobSidecar, len(denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars))
		for i := range denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars {
			signedBlob, err := denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars[i].ToConensus(i)
			if err != nil {
				return nil, err
			}
			signedBlindedBlobSidecars[i] = signedBlob
		}
	}
	signedBlindedBlock, err := denebSignedBlindedBeaconBlockContents.SignedBlindedBlock.ToConsensus()
	if err != nil {
		return nil, err
	}
	block := &eth.SignedBlindedBeaconBlockAndBlobsDeneb{
		Block: signedBlindedBlock,
		Blobs: signedBlindedBlobSidecars,
	}
	return &eth.GenericSignedBeaconBlock{Block: &eth.GenericSignedBeaconBlock_BlindedDeneb{BlindedDeneb: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (denebSignedBlindedBeaconBlockContents *SignedBlindedBeaconBlockContentsDeneb) ToUnsigned() *BlindedBeaconBlockContentsDeneb {
	var blobSidecars []*BlindedBlobSidecar
	if len(denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars) != 0 {
		blobSidecars = make([]*BlindedBlobSidecar, len(denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars))
		for i := range denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars {
			blobSidecars[i] = denebSignedBlindedBeaconBlockContents.SignedBlindedBlobSidecars[i].Message
		}
	}
	return &BlindedBeaconBlockContentsDeneb{
		BlindedBlock:        denebSignedBlindedBeaconBlockContents.SignedBlindedBlock.Message,
		BlindedBlobSidecars: blobSidecars,
	}
}

func (denebBlindedBeaconBlockContents *BlindedBeaconBlockContentsDeneb) ToGeneric() (*eth.GenericBeaconBlock, error) {
	var blindedBlobSidecars []*eth.BlindedBlobSidecar
	if len(denebBlindedBeaconBlockContents.BlindedBlobSidecars) != 0 {
		err := VerifyMaxLength("denebBlindedBeaconBlockContents.BlindedBlobSidecars", len(denebBlindedBeaconBlockContents.BlindedBlobSidecars), fieldparams.MaxBlobsPerBlock)
		if err != nil {
			return nil, err
		}
		blindedBlobSidecars = make([]*eth.BlindedBlobSidecar, len(denebBlindedBeaconBlockContents.BlindedBlobSidecars))
		for i := range denebBlindedBeaconBlockContents.BlindedBlobSidecars {
			blob, err := denebBlindedBeaconBlockContents.BlindedBlobSidecars[i].ToConsensus(i)
			if err != nil {
				return nil, err
			}
			blindedBlobSidecars[i] = blob
		}
	}
	blindedBlock, err := denebBlindedBeaconBlockContents.BlindedBlock.ToConsensus()
	if err != nil {
		return nil, err
	}
	block := &eth.BlindedBeaconBlockAndBlobsDeneb{
		Block: blindedBlock,
		Blobs: blindedBlobSidecars,
	}
	return &eth.GenericBeaconBlock{Block: &eth.GenericBeaconBlock_BlindedDeneb{BlindedDeneb: block}, IsBlinded: true, PayloadValue: 0 /* can't get payload value from blinded block */}, nil
}

func (denebBeaconBlock *BeaconBlockDeneb) ToConsensus() (*eth.BeaconBlockDeneb, error) {
	slot, err := strconv.ParseUint(denebBeaconBlock.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(denebBeaconBlock.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(denebBeaconBlock.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(denebBeaconBlock.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(denebBeaconBlock.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(denebBeaconBlock.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(denebBeaconBlock.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(denebBeaconBlock.Body.Eth1Data.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(denebBeaconBlock.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(denebBeaconBlock.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(denebBeaconBlock.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := AttsToConsensus(denebBeaconBlock.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsToConsensus(denebBeaconBlock.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsToConsensus(denebBeaconBlock.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := DecodeHexWithLength(denebBeaconBlock.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(denebBeaconBlock.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(denebBeaconBlock.Body.ExecutionPayload.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(denebBeaconBlock.Body.ExecutionPayload.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(denebBeaconBlock.Body.ExecutionPayload.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(denebBeaconBlock.Body.ExecutionPayload.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(denebBeaconBlock.Body.ExecutionPayload.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(denebBeaconBlock.Body.ExecutionPayload.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(denebBeaconBlock.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(denebBeaconBlock.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(denebBeaconBlock.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(denebBeaconBlock.Body.ExecutionPayload.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(denebBeaconBlock.Body.ExecutionPayload.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToSSZBytes(denebBeaconBlock.Body.ExecutionPayload.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(denebBeaconBlock.Body.ExecutionPayload.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlockHash")
	}
	err = VerifyMaxLength("denebBeaconBlock.Body.ExecutionPayload.Transactions", len(denebBeaconBlock.Body.ExecutionPayload.Transactions), fieldparams.MaxTxsPerPayloadLength)
	if err != nil {
		return nil, err
	}
	txs := make([][]byte, len(denebBeaconBlock.Body.ExecutionPayload.Transactions))
	for i, tx := range denebBeaconBlock.Body.ExecutionPayload.Transactions {
		txs[i], err = DecodeHexWithMaxLength(tx, fieldparams.MaxBytesPerTxLength)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode denebBeaconBlock.Body.ExecutionPayload.Transactions[%d]", i)
		}
	}
	err = VerifyMaxLength("denebBeaconBlock.Body.ExecutionPayload.Withdrawals", len(denebBeaconBlock.Body.ExecutionPayload.Withdrawals), fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return nil, err
	}
	withdrawals := make([]*enginev1.Withdrawal, len(denebBeaconBlock.Body.ExecutionPayload.Withdrawals))
	for i, w := range denebBeaconBlock.Body.ExecutionPayload.Withdrawals {
		withdrawalIndex, err := strconv.ParseUint(w.WithdrawalIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode denebBeaconBlock.Body.ExecutionPayload.Withdrawals[%d].WithdrawalIndex", i)
		}
		validatorIndex, err := strconv.ParseUint(w.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode denebBeaconBlock.Body.ExecutionPayload.Withdrawals[%d].ValidatorIndex", i)
		}
		address, err := DecodeHexWithLength(w.ExecutionAddress, common.AddressLength)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode denebBeaconBlock.Body.ExecutionPayload.Withdrawals[%d].ExecutionAddress", i)
		}
		amount, err := strconv.ParseUint(w.Amount, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode denebBeaconBlock.Body.ExecutionPayload.Withdrawals[%d].Amount", i)
		}
		withdrawals[i] = &enginev1.Withdrawal{
			Index:          withdrawalIndex,
			ValidatorIndex: primitives.ValidatorIndex(validatorIndex),
			Address:        address,
			Amount:         amount,
		}
	}

	payloadBlobGasUsed, err := strconv.ParseUint(denebBeaconBlock.Body.ExecutionPayload.BlobGasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlobGasUsed")
	}
	payloadExcessBlobGas, err := strconv.ParseUint(denebBeaconBlock.Body.ExecutionPayload.ExcessBlobGas, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExcessBlobGas")
	}
	blsChanges, err := BlsChangesToConsensus(denebBeaconBlock.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, err
	}
	err = VerifyMaxLength("denebBeaconBlock.Body.BlobKzgCommitments", len(denebBeaconBlock.Body.BlobKzgCommitments), 4096)
	if err != nil {
		return nil, err
	}
	blobKzgCommitments := make([][]byte, len(denebBeaconBlock.Body.BlobKzgCommitments))
	for i, b := range denebBeaconBlock.Body.BlobKzgCommitments {
		kzg, err := DecodeHexWithLength(b, fieldparams.BLSPubkeyLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("blob kzg commitment at index %d", i))
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

func (blobSidecar *BlobSidecar) ToConsensus(i int) (*eth.BlobSidecar, error) {
	if blobSidecar == nil {
		return nil, fmt.Errorf("blobsidecar message was empty at index %d", i)
	}
	blockRoot, err := DecodeHexWithLength(blobSidecar.BlockRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blobSidecar.BlockRoot at index %d", i))
	}
	index, err := strconv.ParseUint(blobSidecar.Index, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blobSidecar.Index at index %d", i))
	}
	slot, err := strconv.ParseUint(blobSidecar.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blobSidecar.Index at index %d", i))
	}
	blockParentRoot, err := DecodeHexWithLength(blobSidecar.BlockParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blobSidecar.BlockParentRoot at index %d", i))
	}
	proposerIndex, err := strconv.ParseUint(blobSidecar.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blobSidecar.ProposerIndex at index %d", i))
	}
	blob, err := DecodeHexWithLength(blobSidecar.Blob, fieldparams.BlobLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blobSidecar.Blob at index %d", i))
	}
	kzgCommitment, err := DecodeHexWithLength(blobSidecar.KzgCommitment, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blobSidecar.KzgCommitment at index %d", i))
	}
	kzgProof, err := DecodeHexWithLength(blobSidecar.KzgProof, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blobSidecar.KzgProof at index %d", i))
	}
	bsc := &eth.BlobSidecar{
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

func (denebSignedBeaconBlock *SignedBeaconBlockDeneb) ToConsensus() (*eth.SignedBeaconBlockDeneb, error) {
	sig, err := DecodeHexWithLength(denebSignedBeaconBlock.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	block, err := denebSignedBeaconBlock.Message.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.SignedBeaconBlockDeneb{
		Block:     block,
		Signature: sig,
	}, nil
}

func (signedBlob *SignedBlobSidecar) ToConsensus(i int) (*eth.SignedBlobSidecar, error) {
	blobSig, err := DecodeHexWithLength(signedBlob.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Signature")
	}
	if signedBlob.Message == nil {
		return nil, fmt.Errorf("blobsidecar message was empty at index %d", i)
	}
	blockRoot, err := DecodeHexWithLength(signedBlob.Message.BlockRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("signedBlob.Message.BlockRoot at index %d", i))
	}
	index, err := strconv.ParseUint(signedBlob.Message.Index, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("signedBlob.Message.Index at index %d", i))
	}
	slot, err := strconv.ParseUint(signedBlob.Message.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("signedBlob.Message.Index at index %d", i))
	}
	blockParentRoot, err := DecodeHexWithLength(signedBlob.Message.BlockParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("signedBlob.Message.BlockParentRoot at index %d", i))
	}
	proposerIndex, err := strconv.ParseUint(signedBlob.Message.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("signedBlob.Message.ProposerIndex at index %d", i))
	}
	blob, err := DecodeHexWithLength(signedBlob.Message.Blob, fieldparams.BlobLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("signedBlob.Message.Blob at index %d", i))
	}
	kzgCommitment, err := DecodeHexWithLength(signedBlob.Message.KzgCommitment, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("signedBlob.Message.KzgCommitment at index %d", i))
	}
	kzgProof, err := DecodeHexWithLength(signedBlob.Message.KzgProof, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("signedBlob.Message.KzgProof at index %d", i))
	}
	bsc := &eth.BlobSidecar{
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

func (denebSignedBlindedBeaconBlock *SignedBlindedBeaconBlockDeneb) ToConsensus() (*eth.SignedBlindedBeaconBlockDeneb, error) {
	if denebSignedBlindedBeaconBlock == nil {
		return nil, errors.New("signed blinded block is empty")
	}
	sig, err := DecodeHexWithLength(denebSignedBlindedBeaconBlock.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, ".Signature")
	}
	blindedBlock, err := denebSignedBlindedBeaconBlock.Message.ToConsensus()
	if err != nil {
		return nil, err
	}
	return &eth.SignedBlindedBeaconBlockDeneb{
		Block:     blindedBlock,
		Signature: sig,
	}, nil
}

func (denebBlindedBeaconBlock *BlindedBeaconBlockDeneb) ToConsensus() (*eth.BlindedBeaconBlockDeneb, error) {
	if denebBlindedBeaconBlock == nil {
		return nil, errors.New("blinded block is empty")
	}
	slot, err := strconv.ParseUint(denebBlindedBeaconBlock.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Slot")
	}
	proposerIndex, err := strconv.ParseUint(denebBlindedBeaconBlock.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "ProposerIndex")
	}
	parentRoot, err := DecodeHexWithLength(denebBlindedBeaconBlock.ParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "ParentRoot")
	}
	stateRoot, err := DecodeHexWithLength(denebBlindedBeaconBlock.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "StateRoot")
	}
	randaoReveal, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.RandaoReveal, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.RandaoReveal")
	}
	depositRoot, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.Eth1Data.DepositRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositRoot")
	}
	depositCount, err := strconv.ParseUint(denebBlindedBeaconBlock.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.DepositCount")
	}
	blockHash, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.Eth1Data.BlockHash, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Eth1Data.BlockHash")
	}
	graffiti, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.Graffiti, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.Graffiti")
	}
	proposerSlashings, err := ProposerSlashingsToConsensus(denebBlindedBeaconBlock.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsToConsensus(denebBlindedBeaconBlock.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
	atts, err := AttsToConsensus(denebBlindedBeaconBlock.Body.Attestations)
	if err != nil {
		return nil, err
	}
	deposits, err := DepositsToConsensus(denebBlindedBeaconBlock.Body.Deposits)
	if err != nil {
		return nil, err
	}
	exits, err := ExitsToConsensus(denebBlindedBeaconBlock.Body.VoluntaryExits)
	if err != nil {
		return nil, err
	}
	syncCommitteeBits, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.SyncAggregate.SyncCommitteeBits, fieldparams.SyncAggregateSyncCommitteeBytesLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeBits")
	}
	syncCommitteeSig, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.SyncAggregate.SyncCommitteeSignature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.SyncAggregate.SyncCommitteeSignature")
	}
	payloadParentHash, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.ParentHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ParentHash")
	}
	payloadFeeRecipient, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.FeeRecipient, fieldparams.FeeRecipientLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.FeeRecipient")
	}
	payloadStateRoot, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.StateRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.StateRoot")
	}
	payloadReceiptsRoot, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.ReceiptsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ReceiptsRoot")
	}
	payloadLogsBloom, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.LogsBloom, fieldparams.LogsBloomLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.LogsBloom")
	}
	payloadPrevRandao, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.PrevRandao, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.PrevRandao")
	}
	payloadBlockNumber, err := strconv.ParseUint(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.BlockNumber, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockNumber")
	}
	payloadGasLimit, err := strconv.ParseUint(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.GasLimit, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasLimit")
	}
	payloadGasUsed, err := strconv.ParseUint(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.GasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.GasUsed")
	}
	payloadTimestamp, err := strconv.ParseUint(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.Timestamp, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.Timestamp")
	}
	payloadExtraData, err := DecodeHexWithMaxLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.ExtraData, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.ExtraData")
	}
	payloadBaseFeePerGas, err := uint256ToSSZBytes(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BaseFeePerGas")
	}
	payloadBlockHash, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.BlockHash, common.HashLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.BlockHash")
	}
	payloadTxsRoot, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.TransactionsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.TransactionsRoot")
	}
	payloadWithdrawalsRoot, err := DecodeHexWithLength(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.WithdrawalsRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayloadHeader.WithdrawalsRoot")
	}

	payloadBlobGasUsed, err := strconv.ParseUint(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.BlobGasUsed, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.BlobGasUsed")
	}
	payloadExcessBlobGas, err := strconv.ParseUint(denebBlindedBeaconBlock.Body.ExecutionPayloadHeader.ExcessBlobGas, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, "Body.ExecutionPayload.ExcessBlobGas")
	}

	blsChanges, err := BlsChangesToConsensus(denebBlindedBeaconBlock.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, err
	}
	err = VerifyMaxLength("denebBlindedBeaconBlock.Body.BlobKzgCommitments", len(denebBlindedBeaconBlock.Body.BlobKzgCommitments), 4096)
	if err != nil {
		return nil, err
	}
	blobKzgCommitments := make([][]byte, len(denebBlindedBeaconBlock.Body.BlobKzgCommitments))
	for i, b := range denebBlindedBeaconBlock.Body.BlobKzgCommitments {
		kzg, err := DecodeHexWithLength(b, fieldparams.BLSPubkeyLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("blob kzg commitment at index %d", i))
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

func (signedBlindedBlob *SignedBlindedBlobSidecar) ToConensus(i int) (*eth.SignedBlindedBlobSidecar, error) {
	blobSig, err := DecodeHexWithLength(signedBlindedBlob.Signature, fieldparams.BLSSignatureLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not decode signedBlindedBlob.Signature")
	}
	if signedBlindedBlob.Message == nil {
		return nil, fmt.Errorf("blobsidecar message was empty at index %d", i)
	}
	bsc, err := signedBlindedBlob.Message.ToConsensus(i)
	if err != nil {
		return nil, err
	}
	return &eth.SignedBlindedBlobSidecar{
		Message:   bsc,
		Signature: blobSig,
	}, nil
}

func (blindedBlob *BlindedBlobSidecar) ToConsensus(i int) (*eth.BlindedBlobSidecar, error) {
	if blindedBlob == nil {
		return nil, fmt.Errorf("blobsidecar message was empty at index %d", i)
	}
	blockRoot, err := DecodeHexWithLength(blindedBlob.BlockRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blindedBlob.BlockRoot at index %d", i))
	}
	index, err := strconv.ParseUint(blindedBlob.Index, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blindedBlob.Index at index %d", i))
	}
	denebSlot, err := strconv.ParseUint(blindedBlob.Slot, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blindedBlob.Index at index %d", i))
	}
	blockParentRoot, err := DecodeHexWithLength(blindedBlob.BlockParentRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blindedBlob.BlockParentRoot at index %d", i))
	}
	proposerIndex, err := strconv.ParseUint(blindedBlob.ProposerIndex, 10, 64)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blindedBlob.ProposerIndex at index %d", i))
	}
	blobRoot, err := DecodeHexWithLength(blindedBlob.BlobRoot, fieldparams.RootLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blindedBlob.BlobRoot at index %d", i))
	}
	kzgCommitment, err := DecodeHexWithLength(blindedBlob.KzgCommitment, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blindedBlob.KzgCommitment at index %d", i))
	}
	kzgProof, err := DecodeHexWithLength(blindedBlob.KzgProof, fieldparams.BLSPubkeyLength)
	if err != nil {
		return nil, NewDecodeError(err, fmt.Sprintf("blindedBlob.KzgProof at index %d", i))
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

func BeaconBlockFromConsensus(b *eth.BeaconBlock) (*BeaconBlock, error) {
	if b == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
	proposerSlashings, err := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
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
	return &BeaconBlock{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBody{
			RandaoReveal: hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data: &Eth1Data{
				DepositRoot:  hexutil.Encode(b.Body.Eth1Data.DepositRoot),
				DepositCount: fmt.Sprintf("%d", b.Body.Eth1Data.DepositCount),
				BlockHash:    hexutil.Encode(b.Body.Eth1Data.BlockHash),
			},
			Graffiti:          hexutil.Encode(b.Body.Graffiti),
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      atts,
			Deposits:          deposits,
			VoluntaryExits:    exits,
		},
	}, nil
}

func BeaconBlockAltairFromConsensus(b *eth.BeaconBlockAltair) (*BeaconBlockAltair, error) {
	if b == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
	proposerSlashings, err := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
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

	return &BeaconBlockAltair{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyAltair{
			RandaoReveal: hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data: &Eth1Data{
				DepositRoot:  hexutil.Encode(b.Body.Eth1Data.DepositRoot),
				DepositCount: fmt.Sprintf("%d", b.Body.Eth1Data.DepositCount),
				BlockHash:    hexutil.Encode(b.Body.Eth1Data.BlockHash),
			},
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

func BlindedBeaconBlockBellatrixFromConsensus(b *eth.BlindedBeaconBlockBellatrix) (*BlindedBeaconBlockBellatrix, error) {
	if b == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
	proposerSlashings, err := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
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
	baseFeePerGas, err := sszBytesToUint256String(b.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, err
	}
	return &BlindedBeaconBlockBellatrix{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BlindedBeaconBlockBodyBellatrix{
			RandaoReveal: hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data: &Eth1Data{
				DepositRoot:  hexutil.Encode(b.Body.Eth1Data.DepositRoot),
				DepositCount: fmt.Sprintf("%d", b.Body.Eth1Data.DepositCount),
				BlockHash:    hexutil.Encode(b.Body.Eth1Data.BlockHash),
			},
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
			ExecutionPayloadHeader: &ExecutionPayloadHeader{
				ParentHash:       hexutil.Encode(b.Body.ExecutionPayloadHeader.ParentHash),
				FeeRecipient:     hexutil.Encode(b.Body.ExecutionPayloadHeader.FeeRecipient),
				StateRoot:        hexutil.Encode(b.Body.ExecutionPayloadHeader.StateRoot),
				ReceiptsRoot:     hexutil.Encode(b.Body.ExecutionPayloadHeader.ReceiptsRoot),
				LogsBloom:        hexutil.Encode(b.Body.ExecutionPayloadHeader.LogsBloom),
				PrevRandao:       hexutil.Encode(b.Body.ExecutionPayloadHeader.PrevRandao),
				BlockNumber:      fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.BlockNumber),
				GasLimit:         fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.GasLimit),
				GasUsed:          fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.GasUsed),
				Timestamp:        fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.Timestamp),
				ExtraData:        hexutil.Encode(b.Body.ExecutionPayloadHeader.ExtraData),
				BaseFeePerGas:    baseFeePerGas,
				BlockHash:        hexutil.Encode(b.Body.ExecutionPayloadHeader.BlockHash),
				TransactionsRoot: hexutil.Encode(b.Body.ExecutionPayloadHeader.TransactionsRoot),
			},
		},
	}, nil
}

func BeaconBlockBellatrixFromConsensus(b *eth.BeaconBlockBellatrix) (*BeaconBlockBellatrix, error) {
	if b == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
	proposerSlashings, err := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
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
	return &BeaconBlockBellatrix{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyBellatrix{
			RandaoReveal: hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data: &Eth1Data{
				DepositRoot:  hexutil.Encode(b.Body.Eth1Data.DepositRoot),
				DepositCount: fmt.Sprintf("%d", b.Body.Eth1Data.DepositCount),
				BlockHash:    hexutil.Encode(b.Body.Eth1Data.BlockHash),
			},
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

func BlindedBeaconBlockCapellaFromConsensus(b *eth.BlindedBeaconBlockCapella) (*BlindedBeaconBlockCapella, error) {
	if b == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
	proposerSlashings, err := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
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
	baseFeePerGas, err := sszBytesToUint256String(b.Body.ExecutionPayloadHeader.BaseFeePerGas)
	if err != nil {
		return nil, err
	}
	blsChanges, err := BlsChangesFromConsensus(b.Body.BlsToExecutionChanges)
	if err != nil {
		return nil, err
	}

	return &BlindedBeaconBlockCapella{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BlindedBeaconBlockBodyCapella{
			RandaoReveal: hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data: &Eth1Data{
				DepositRoot:  hexutil.Encode(b.Body.Eth1Data.DepositRoot),
				DepositCount: fmt.Sprintf("%d", b.Body.Eth1Data.DepositCount),
				BlockHash:    hexutil.Encode(b.Body.Eth1Data.BlockHash),
			},
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
			ExecutionPayloadHeader: &ExecutionPayloadHeaderCapella{
				ParentHash:       hexutil.Encode(b.Body.ExecutionPayloadHeader.ParentHash),
				FeeRecipient:     hexutil.Encode(b.Body.ExecutionPayloadHeader.FeeRecipient),
				StateRoot:        hexutil.Encode(b.Body.ExecutionPayloadHeader.StateRoot),
				ReceiptsRoot:     hexutil.Encode(b.Body.ExecutionPayloadHeader.ReceiptsRoot),
				LogsBloom:        hexutil.Encode(b.Body.ExecutionPayloadHeader.LogsBloom),
				PrevRandao:       hexutil.Encode(b.Body.ExecutionPayloadHeader.PrevRandao),
				BlockNumber:      fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.BlockNumber),
				GasLimit:         fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.GasLimit),
				GasUsed:          fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.GasUsed),
				Timestamp:        fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.Timestamp),
				ExtraData:        hexutil.Encode(b.Body.ExecutionPayloadHeader.ExtraData),
				BaseFeePerGas:    baseFeePerGas,
				BlockHash:        hexutil.Encode(b.Body.ExecutionPayloadHeader.BlockHash),
				TransactionsRoot: hexutil.Encode(b.Body.ExecutionPayloadHeader.TransactionsRoot),
				WithdrawalsRoot:  hexutil.Encode(b.Body.ExecutionPayloadHeader.WithdrawalsRoot), // new in capella
			},
			BlsToExecutionChanges: blsChanges, // new in capella
		},
	}, nil
}

func BeaconBlockCapellaFromConsensus(b *eth.BeaconBlockCapella) (*BeaconBlockCapella, error) {
	if b == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
	proposerSlashings, err := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
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
	return &BeaconBlockCapella{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyCapella{
			RandaoReveal: hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data: &Eth1Data{
				DepositRoot:  hexutil.Encode(b.Body.Eth1Data.DepositRoot),
				DepositCount: fmt.Sprintf("%d", b.Body.Eth1Data.DepositCount),
				BlockHash:    hexutil.Encode(b.Body.Eth1Data.BlockHash),
			},
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

func BlindedBeaconBlockContentsDenebFromConsensus(b *eth.BlindedBeaconBlockAndBlobsDeneb) (*BlindedBeaconBlockContentsDeneb, error) {
	if b == nil || b.Block == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
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
	blindedBlock, err := BlindedDenebBlockFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &BlindedBeaconBlockContentsDeneb{
		BlindedBlock:        blindedBlock,
		BlindedBlobSidecars: blindedBlobSidecars,
	}, nil
}

func BeaconBlockContentsDenebFromConsensus(b *eth.BeaconBlockAndBlobsDeneb) (*BeaconBlockContentsDeneb, error) {
	if b == nil || b.Block == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
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
	block, err := DenebBlockFromConsensus(b.Block)
	if err != nil {
		return nil, err
	}
	return &BeaconBlockContentsDeneb{
		Block:        block,
		BlobSidecars: blobSidecars,
	}, nil
}

func BlindedDenebBlockFromConsensus(b *eth.BlindedBeaconBlockDeneb) (*BlindedBeaconBlockDeneb, error) {
	if b == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
	proposerSlashings, err := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
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
	baseFeePerGas, err := sszBytesToUint256String(b.Body.ExecutionPayloadHeader.BaseFeePerGas)
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

	return &BlindedBeaconBlockDeneb{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BlindedBeaconBlockBodyDeneb{
			RandaoReveal: hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data: &Eth1Data{
				DepositRoot:  hexutil.Encode(b.Body.Eth1Data.DepositRoot),
				DepositCount: fmt.Sprintf("%d", b.Body.Eth1Data.DepositCount),
				BlockHash:    hexutil.Encode(b.Body.Eth1Data.BlockHash),
			},
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
			ExecutionPayloadHeader: &ExecutionPayloadHeaderDeneb{
				ParentHash:       hexutil.Encode(b.Body.ExecutionPayloadHeader.ParentHash),
				FeeRecipient:     hexutil.Encode(b.Body.ExecutionPayloadHeader.FeeRecipient),
				StateRoot:        hexutil.Encode(b.Body.ExecutionPayloadHeader.StateRoot),
				ReceiptsRoot:     hexutil.Encode(b.Body.ExecutionPayloadHeader.ReceiptsRoot),
				LogsBloom:        hexutil.Encode(b.Body.ExecutionPayloadHeader.LogsBloom),
				PrevRandao:       hexutil.Encode(b.Body.ExecutionPayloadHeader.PrevRandao),
				BlockNumber:      fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.BlockNumber),
				GasLimit:         fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.GasLimit),
				GasUsed:          fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.GasUsed),
				Timestamp:        fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.Timestamp),
				ExtraData:        hexutil.Encode(b.Body.ExecutionPayloadHeader.ExtraData),
				BaseFeePerGas:    baseFeePerGas,
				BlockHash:        hexutil.Encode(b.Body.ExecutionPayloadHeader.BlockHash),
				TransactionsRoot: hexutil.Encode(b.Body.ExecutionPayloadHeader.TransactionsRoot),
				WithdrawalsRoot:  hexutil.Encode(b.Body.ExecutionPayloadHeader.WithdrawalsRoot),
				BlobGasUsed:      fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.BlobGasUsed),   // new in deneb TODO: rename to blob
				ExcessBlobGas:    fmt.Sprintf("%d", b.Body.ExecutionPayloadHeader.ExcessBlobGas), // new in deneb TODO: rename to blob
			},
			BlsToExecutionChanges: blsChanges,         // new in capella
			BlobKzgCommitments:    blobKzgCommitments, // new in deneb
		},
	}, nil
}

func DenebBlockFromConsensus(b *eth.BeaconBlockDeneb) (*BeaconBlockDeneb, error) {
	if b == nil {
		return nil, errors.New("block is empty, nothing to convert.")
	}
	proposerSlashings, err := ProposerSlashingsFromConsensus(b.Body.ProposerSlashings)
	if err != nil {
		return nil, err
	}
	attesterSlashings, err := AttesterSlashingsFromConsensus(b.Body.AttesterSlashings)
	if err != nil {
		return nil, err
	}
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

	return &BeaconBlockDeneb{
		Slot:          fmt.Sprintf("%d", b.Slot),
		ProposerIndex: fmt.Sprintf("%d", b.ProposerIndex),
		ParentRoot:    hexutil.Encode(b.ParentRoot),
		StateRoot:     hexutil.Encode(b.StateRoot),
		Body: &BeaconBlockBodyDeneb{
			RandaoReveal: hexutil.Encode(b.Body.RandaoReveal),
			Eth1Data: &Eth1Data{
				DepositRoot:  hexutil.Encode(b.Body.Eth1Data.DepositRoot),
				DepositCount: fmt.Sprintf("%d", b.Body.Eth1Data.DepositCount),
				BlockHash:    hexutil.Encode(b.Body.Eth1Data.BlockHash),
			},
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

func BlindedBlobSidecarFromConsensus(b *eth.BlindedBlobSidecar) (*BlindedBlobSidecar, error) {
	if b == nil {
		return nil, errors.New("BlindedBlobSidecar is empty, nothing to convert.")
	}
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

func BlobSidecarFromConsensus(b *eth.BlobSidecar) (*BlobSidecar, error) {
	if b == nil {
		return nil, errors.New("BlobSidecar is empty, nothing to convert.")
	}
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

func ProposerSlashingsToConsensus(src []*ProposerSlashing) ([]*eth.ProposerSlashing, error) {
	if src == nil {
		return nil, errors.New("nil b.Message.Body.ProposerSlashings")
	}
	err := VerifyMaxLength("b.Message.Body.ProposerSlashings", len(src), 16)
	if err != nil {
		return nil, err
	}
	proposerSlashings := make([]*eth.ProposerSlashing, len(src))
	for i, s := range src {
		h1Sig, err := DecodeHexWithLength(s.SignedHeader1.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader1.Signature", i))
		}
		h1Slot, err := strconv.ParseUint(s.SignedHeader1.Message.Slot, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader1.Message.Slot", i))
		}
		h1ProposerIndex, err := strconv.ParseUint(s.SignedHeader1.Message.ProposerIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader1.Message.ProposerIndex", i))
		}
		h1ParentRoot, err := DecodeHexWithLength(s.SignedHeader1.Message.ParentRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader1.Message.ParentRoot", i))
		}
		h1StateRoot, err := DecodeHexWithLength(s.SignedHeader1.Message.StateRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader1.Message.StateRoot", i))
		}
		h1BodyRoot, err := DecodeHexWithLength(s.SignedHeader1.Message.BodyRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader1.Message.BodyRoot", i))
		}
		h2Sig, err := DecodeHexWithLength(s.SignedHeader2.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader2.Signature", i))
		}
		h2Slot, err := strconv.ParseUint(s.SignedHeader2.Message.Slot, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader2.Message.Slot", i))
		}
		h2ProposerIndex, err := strconv.ParseUint(s.SignedHeader2.Message.ProposerIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader2.Message.ProposerIndex", i))
		}
		h2ParentRoot, err := DecodeHexWithLength(s.SignedHeader2.Message.ParentRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader2.Message.ParentRoot", i))
		}
		h2StateRoot, err := DecodeHexWithLength(s.SignedHeader2.Message.StateRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader2.Message.StateRoot", i))
		}
		h2BodyRoot, err := DecodeHexWithLength(s.SignedHeader2.Message.BodyRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.ProposerSlashings[%d].SignedHeader2.Message.BodyRoot", i))
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

func ProposerSlashingsFromConsensus(src []*eth.ProposerSlashing) ([]*ProposerSlashing, error) {
	if src == nil {
		return nil, errors.New("proposer slashings are empty, nothing to convert.")
	}
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
	return proposerSlashings, nil
}

func AttesterSlashingsToConsensus(src []*AttesterSlashing) ([]*eth.AttesterSlashing, error) {
	if src == nil {
		return nil, errors.New("nil b.Message.Body.AttesterSlashings")
	}
	err := VerifyMaxLength("b.Message.Body.AttesterSlashings", len(src), 2)
	if err != nil {
		return nil, err
	}
	attesterSlashings := make([]*eth.AttesterSlashing, len(src))
	for i, s := range src {
		a1Sig, err := DecodeHexWithLength(s.Attestation1.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation1.Signature", i))
		}
		err = VerifyMaxLength(fmt.Sprintf("s.Attestation1.AttestingIndices at index %d", i), len(s.Attestation1.AttestingIndices), 2048)
		if err != nil {
			return nil, err
		}
		a1AttestingIndices := make([]uint64, len(s.Attestation1.AttestingIndices))
		for j, ix := range s.Attestation1.AttestingIndices {
			attestingIndex, err := strconv.ParseUint(ix, 10, 64)
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation1.AttestingIndices[%d]", i, j))
			}
			a1AttestingIndices[j] = attestingIndex
		}
		a1Slot, err := strconv.ParseUint(s.Attestation1.Data.Slot, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation1.Data.Slot", i))
		}
		a1CommitteeIndex, err := strconv.ParseUint(s.Attestation1.Data.CommitteeIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation1.Data.Index", i))
		}
		a1BeaconBlockRoot, err := DecodeHexWithLength(s.Attestation1.Data.BeaconBlockRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation1.Data.BeaconBlockRoot", i))
		}
		a1SourceEpoch, err := strconv.ParseUint(s.Attestation1.Data.Source.Epoch, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation1.Data.Source.Epoch", i))
		}
		a1SourceRoot, err := DecodeHexWithLength(s.Attestation1.Data.Source.Root, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation1.Data.Source.Root", i))
		}
		a1TargetEpoch, err := strconv.ParseUint(s.Attestation1.Data.Target.Epoch, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation1.Data.Target.Epoch", i))
		}
		a1TargetRoot, err := DecodeHexWithLength(s.Attestation1.Data.Target.Root, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation1.Data.Target.Root", i))
		}
		a2Sig, err := DecodeHexWithLength(s.Attestation2.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation2.Signature", i))
		}
		err = VerifyMaxLength(fmt.Sprintf("s.Attestation2.AttestingIndices at index %d", i), len(s.Attestation2.AttestingIndices), 2048)
		if err != nil {
			return nil, err
		}
		a2AttestingIndices := make([]uint64, len(s.Attestation2.AttestingIndices))
		for j, ix := range s.Attestation2.AttestingIndices {
			attestingIndex, err := strconv.ParseUint(ix, 10, 64)
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation2.AttestingIndices[%d]", i, j))
			}
			a2AttestingIndices[j] = attestingIndex
		}
		a2Slot, err := strconv.ParseUint(s.Attestation2.Data.Slot, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation2.Data.Slot", i))
		}
		a2CommitteeIndex, err := strconv.ParseUint(s.Attestation2.Data.CommitteeIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation2.Data.Index", i))
		}
		a2BeaconBlockRoot, err := DecodeHexWithLength(s.Attestation2.Data.BeaconBlockRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation2.Data.BeaconBlockRoot", i))
		}
		a2SourceEpoch, err := strconv.ParseUint(s.Attestation2.Data.Source.Epoch, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation2.Data.Source.Epoch", i))
		}
		a2SourceRoot, err := DecodeHexWithLength(s.Attestation2.Data.Source.Root, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation2.Data.Source.Root", i))
		}
		a2TargetEpoch, err := strconv.ParseUint(s.Attestation2.Data.Target.Epoch, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation2.Data.Target.Epoch", i))
		}
		a2TargetRoot, err := DecodeHexWithLength(s.Attestation2.Data.Target.Root, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.AttesterSlashings[%d].Attestation2.Data.Target.Root", i))
		}
		attesterSlashings[i] = &eth.AttesterSlashing{
			Attestation_1: &eth.IndexedAttestation{
				AttestingIndices: a1AttestingIndices,
				Data: &eth.AttestationData{
					Slot:            primitives.Slot(a1Slot),
					CommitteeIndex:  primitives.CommitteeIndex(a1CommitteeIndex),
					BeaconBlockRoot: a1BeaconBlockRoot,
					Source: &eth.Checkpoint{
						Epoch: primitives.Epoch(a1SourceEpoch),
						Root:  a1SourceRoot,
					},
					Target: &eth.Checkpoint{
						Epoch: primitives.Epoch(a1TargetEpoch),
						Root:  a1TargetRoot,
					},
				},
				Signature: a1Sig,
			},
			Attestation_2: &eth.IndexedAttestation{
				AttestingIndices: a2AttestingIndices,
				Data: &eth.AttestationData{
					Slot:            primitives.Slot(a2Slot),
					CommitteeIndex:  primitives.CommitteeIndex(a2CommitteeIndex),
					BeaconBlockRoot: a2BeaconBlockRoot,
					Source: &eth.Checkpoint{
						Epoch: primitives.Epoch(a2SourceEpoch),
						Root:  a2SourceRoot,
					},
					Target: &eth.Checkpoint{
						Epoch: primitives.Epoch(a2TargetEpoch),
						Root:  a2TargetRoot,
					},
				},
				Signature: a2Sig,
			},
		}
	}
	return attesterSlashings, nil
}

func AttesterSlashingsFromConsensus(src []*eth.AttesterSlashing) ([]*AttesterSlashing, error) {
	if src == nil {
		return nil, errors.New("AttesterSlashings is empty, nothing to convert.")
	}
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
	return attesterSlashings, nil
}

func AttsToConsensus(src []*Attestation) ([]*eth.Attestation, error) {
	if src == nil {
		return nil, errors.New("nil b.Message.Body.Attestations")
	}
	err := VerifyMaxLength("b.Message.Body.Attestations", len(src), 128)
	if err != nil {
		return nil, err
	}
	atts := make([]*eth.Attestation, len(src))
	for i, a := range src {
		sig, err := DecodeHexWithLength(a.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Attestations[%d].Signature", i))
		}
		slot, err := strconv.ParseUint(a.Data.Slot, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Attestations[%d].Data.Slot", i))
		}
		committeeIndex, err := strconv.ParseUint(a.Data.CommitteeIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Attestations[%d].Data.Index", i))
		}
		beaconBlockRoot, err := DecodeHexWithLength(a.Data.BeaconBlockRoot, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Attestations[%d].Data.BeaconBlockRoot", i))
		}
		sourceEpoch, err := strconv.ParseUint(a.Data.Source.Epoch, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Attestations[%d].Data.Source.Epoch", i))
		}
		sourceRoot, err := DecodeHexWithLength(a.Data.Source.Root, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Attestations[%d].Data.Source.Root", i))
		}
		targetEpoch, err := strconv.ParseUint(a.Data.Target.Epoch, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Attestations[%d].Data.Target.Epoch", i))
		}
		targetRoot, err := DecodeHexWithLength(a.Data.Target.Root, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Attestations[%d].Data.Target.Root", i))
		}
		aggregateBits, err := DecodeHexWithMaxLength(a.AggregationBits, 2048)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Attestations[%d].AggregationBits", i))
		}
		atts[i] = &eth.Attestation{
			AggregationBits: aggregateBits,
			Data: &eth.AttestationData{
				Slot:            primitives.Slot(slot),
				CommitteeIndex:  primitives.CommitteeIndex(committeeIndex),
				BeaconBlockRoot: beaconBlockRoot,
				Source: &eth.Checkpoint{
					Epoch: primitives.Epoch(sourceEpoch),
					Root:  sourceRoot,
				},
				Target: &eth.Checkpoint{
					Epoch: primitives.Epoch(targetEpoch),
					Root:  targetRoot,
				},
			},
			Signature: sig,
		}
	}
	return atts, nil
}

func AttsFromConsensus(src []*eth.Attestation) ([]*Attestation, error) {
	if src == nil {
		return nil, errors.New("Attestations are empty, nothing to convert.")
	}
	atts := make([]*Attestation, len(src))
	for i, a := range src {
		atts[i] = &Attestation{
			AggregationBits: hexutil.Encode(a.AggregationBits),
			Data: &AttestationData{
				Slot:            fmt.Sprintf("%d", a.Data.Slot),
				CommitteeIndex:  fmt.Sprintf("%d", a.Data.CommitteeIndex),
				BeaconBlockRoot: hexutil.Encode(a.Data.BeaconBlockRoot),
				Source: &Checkpoint{
					Epoch: fmt.Sprintf("%d", a.Data.Source.Epoch),
					Root:  hexutil.Encode(a.Data.Source.Root),
				},
				Target: &Checkpoint{
					Epoch: fmt.Sprintf("%d", a.Data.Target.Epoch),
					Root:  hexutil.Encode(a.Data.Target.Root),
				},
			},
			Signature: hexutil.Encode(a.Signature),
		}
	}
	return atts, nil
}

func DepositsToConsensus(src []*Deposit) ([]*eth.Deposit, error) {
	if src == nil {
		return nil, errors.New("nil b.Message.Body.Deposits")
	}
	err := VerifyMaxLength("b.Message.Body.Deposits", len(src), 16)
	if err != nil {
		return nil, err
	}
	deposits := make([]*eth.Deposit, len(src))
	for i, d := range src {
		err = VerifyMaxLength("d.Proof", len(d.Proof), 33)
		if err != nil {
			return nil, err
		}
		proof := make([][]byte, len(d.Proof))
		for j, p := range d.Proof {
			var err error
			proof[j], err = DecodeHexWithLength(p, fieldparams.RootLength)
			if err != nil {
				return nil, NewDecodeError(err, fmt.Sprintf("Body.Deposits[%d].Proof[%d]", i, j))
			}
		}
		pubkey, err := DecodeHexWithLength(d.Data.Pubkey, fieldparams.BLSPubkeyLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Deposits[%d].Pubkey", i))
		}
		withdrawalCreds, err := DecodeHexWithLength(d.Data.WithdrawalCredentials, fieldparams.RootLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Deposits[%d].WithdrawalCredentials", i))
		}
		amount, err := strconv.ParseUint(d.Data.Amount, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Deposits[%d].Amount", i))
		}
		sig, err := DecodeHexWithLength(d.Data.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.Deposits[%d].Signature", i))
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
	if src == nil {
		return nil, errors.New("deposits are empty, nothing to convert.")
	}
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
		return nil, errors.New("nil b.Message.Body.VoluntaryExits")
	}
	err := VerifyMaxLength("b.Message.Body.VoluntaryExits", len(src), 16)
	if err != nil {
		return nil, err
	}
	exits := make([]*eth.SignedVoluntaryExit, len(src))
	for i, e := range src {
		sig, err := DecodeHexWithLength(e.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.VoluntaryExits[%d].Signature", i))
		}
		epoch, err := strconv.ParseUint(e.Message.Epoch, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.VoluntaryExits[%d].Epoch", i))
		}
		validatorIndex, err := strconv.ParseUint(e.Message.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.VoluntaryExits[%d].ValidatorIndex", i))
		}
		exits[i] = &eth.SignedVoluntaryExit{
			Exit: &eth.VoluntaryExit{
				Epoch:          primitives.Epoch(epoch),
				ValidatorIndex: primitives.ValidatorIndex(validatorIndex),
			},
			Signature: sig,
		}
	}
	return exits, nil
}

func ExitsFromConsensus(src []*eth.SignedVoluntaryExit) ([]*SignedVoluntaryExit, error) {
	if src == nil {
		return nil, errors.New("VoluntaryExits are empty, nothing to convert.")
	}
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
		return nil, errors.New("nil b.Message.Body.BlsToExecutionChanges")
	}
	err := VerifyMaxLength("b.Message.Body.BlsToExecutionChanges", len(src), 16)
	if err != nil {
		return nil, err
	}
	changes := make([]*eth.SignedBLSToExecutionChange, len(src))
	for i, ch := range src {
		sig, err := DecodeHexWithLength(ch.Signature, fieldparams.BLSSignatureLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.BlsToExecutionChanges[%d].Signature", i))
		}
		index, err := strconv.ParseUint(ch.Message.ValidatorIndex, 10, 64)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.BlsToExecutionChanges[%d].Message.ValidatorIndex", i))
		}
		pubkey, err := DecodeHexWithLength(ch.Message.FromBlsPubkey, fieldparams.BLSPubkeyLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.BlsToExecutionChanges[%d].Message.FromBlsPubkey", i))
		}
		address, err := DecodeHexWithLength(ch.Message.ToExecutionAddress, common.AddressLength)
		if err != nil {
			return nil, NewDecodeError(err, fmt.Sprintf("Body.BlsToExecutionChanges[%d].Message.ToExecutionAddress", i))
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
	if src == nil {
		return nil, errors.New("BlsToExecutionChanges are empty, nothing to convert.")
	}
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

func uint256ToSSZBytes(num string) ([]byte, error) {
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
	return string([]byte(bi.String())), nil
}
