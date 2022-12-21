package beacon_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	neturl "net/url"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

type abstractProduceBlockResponseJson struct {
	Version string          `json:"version" enum:"true"`
	Data    json.RawMessage `json:"data"`
}

func (c beaconApiValidatorClient) getBeaconBlock(slot types.Slot, randaoReveal []byte, graffiti []byte) (*ethpb.GenericBeaconBlock, error) {
	queryParams := neturl.Values{}
	queryParams.Add("randao_reveal", hexutil.Encode(randaoReveal))

	if len(graffiti) > 0 {
		queryParams.Add("graffiti", hexutil.Encode(graffiti))
	}

	queryUrl := buildURL(fmt.Sprintf("/eth/v2/validator/blocks/%d", slot), queryParams)

	// Since we don't know yet what the json looks like, we unmarshal into an abstract structure that has only a version
	// and a blob of data
	produceBlockResponseJson := abstractProduceBlockResponseJson{}
	if _, err := c.jsonRestHandler.GetRestJsonResponse(queryUrl, &produceBlockResponseJson); err != nil {
		return nil, errors.Wrap(err, "failed to query GET REST endpoint")
	}

	// Once we know what the consensus version is, we can go ahead and unmarshal into the specific structs unique to each version
	decoder := json.NewDecoder(bytes.NewReader(produceBlockResponseJson.Data))
	decoder.DisallowUnknownFields()

	response := &ethpb.GenericBeaconBlock{}

	switch produceBlockResponseJson.Version {
	case "phase0":
		jsonPhase0Block := apimiddleware.BeaconBlockJson{}
		if err := decoder.Decode(&jsonPhase0Block); err != nil {
			return nil, errors.Wrap(err, "failed to decode phase0 block response json")
		}

		phase0Block, err := convertRESTPhase0BlockToProto(&jsonPhase0Block)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get phase0 block")
		}
		response.Block = phase0Block

	case "altair":
		jsonAltairBlock := apimiddleware.BeaconBlockAltairJson{}
		if err := decoder.Decode(&jsonAltairBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode altair block response json")
		}

		altairBlock, err := convertRESTAltairBlockToProto(&jsonAltairBlock)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get altair block")
		}
		response.Block = altairBlock

	case "bellatrix":
		jsonBellatrixBlock := apimiddleware.BeaconBlockBellatrixJson{}
		if err := decoder.Decode(&jsonBellatrixBlock); err != nil {
			return nil, errors.Wrap(err, "failed to decode bellatrix block response json")
		}

		bellatrixBlock, err := convertRESTBellatrixBlockToProto(&jsonBellatrixBlock)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get bellatrix block")
		}
		response.Block = bellatrixBlock

	default:
		return nil, errors.Errorf("unsupported consensus version `%s`", produceBlockResponseJson.Version)
	}
	return response, nil
}

func convertRESTPhase0BlockToProto(block *apimiddleware.BeaconBlockJson) (*ethpb.GenericBeaconBlock_Phase0, error) {
	blockSlot, err := strconv.ParseUint(block.Slot, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse slot `%s`", block.Slot)
	}

	blockProposerIndex, err := strconv.ParseUint(block.ProposerIndex, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse proposer index `%s`", block.ProposerIndex)
	}

	parentRoot, err := hexutil.Decode(block.ParentRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode parent root `%s`", block.ParentRoot)
	}

	stateRoot, err := hexutil.Decode(block.StateRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode state root `%s`", block.StateRoot)
	}

	if block.Body == nil {
		return nil, errors.New("block body is nil")
	}

	randaoReveal, err := hexutil.Decode(block.Body.RandaoReveal)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode randao reveal `%s`", block.Body.RandaoReveal)
	}

	if block.Body.Eth1Data == nil {
		return nil, errors.New("eth1 data is nil")
	}

	depositRoot, err := hexutil.Decode(block.Body.Eth1Data.DepositRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode deposit root `%s`", block.Body.Eth1Data.DepositRoot)
	}

	depositCount, err := strconv.ParseUint(block.Body.Eth1Data.DepositCount, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse deposit count `%s`", block.Body.Eth1Data.DepositCount)
	}

	blockHash, err := hexutil.Decode(block.Body.Eth1Data.BlockHash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode block hash `%s`", block.Body.Eth1Data.BlockHash)
	}

	graffiti, err := hexutil.Decode(block.Body.Graffiti)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode graffiti `%s`", block.Body.Graffiti)
	}

	proposerSlashings, err := convertProposerSlashingsToProto(block.Body.ProposerSlashings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get proposer slashings")
	}

	attesterSlashings, err := convertAttesterSlashingsToProto(block.Body.AttesterSlashings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attester slashings")
	}

	attestations, err := convertAttestationsToProto(block.Body.Attestations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get attestations")
	}

	deposits, err := convertDepositsToProto(block.Body.Deposits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get deposits")
	}

	voluntaryExits, err := convertVoluntaryExitsToProto(block.Body.VoluntaryExits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get voluntary exits")
	}

	return &ethpb.GenericBeaconBlock_Phase0{
		Phase0: &ethpb.BeaconBlock{
			Slot:          types.Slot(blockSlot),
			ProposerIndex: types.ValidatorIndex(blockProposerIndex),
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			Body: &ethpb.BeaconBlockBody{
				RandaoReveal: randaoReveal,
				Eth1Data: &ethpb.Eth1Data{
					DepositRoot:  depositRoot,
					DepositCount: depositCount,
					BlockHash:    blockHash,
				},
				Graffiti:          graffiti,
				ProposerSlashings: proposerSlashings,
				AttesterSlashings: attesterSlashings,
				Attestations:      attestations,
				Deposits:          deposits,
				VoluntaryExits:    voluntaryExits,
			},
		},
	}, nil
}

func convertRESTAltairBlockToProto(block *apimiddleware.BeaconBlockAltairJson) (*ethpb.GenericBeaconBlock_Altair, error) {
	if block.Body == nil {
		return nil, errors.New("block body is nil")
	}

	// Call convertRESTPhase0BlockToProto to set the phase0 fields because all the error handling and the heavy lifting
	// has already been done
	phase0Block, err := convertRESTPhase0BlockToProto(&apimiddleware.BeaconBlockJson{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    block.ParentRoot,
		StateRoot:     block.StateRoot,
		Body: &apimiddleware.BeaconBlockBodyJson{
			RandaoReveal:      block.Body.RandaoReveal,
			Eth1Data:          block.Body.Eth1Data,
			Graffiti:          block.Body.Graffiti,
			ProposerSlashings: block.Body.ProposerSlashings,
			AttesterSlashings: block.Body.AttesterSlashings,
			Attestations:      block.Body.Attestations,
			Deposits:          block.Body.Deposits,
			VoluntaryExits:    block.Body.VoluntaryExits,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the phase0 fields of the altair block")
	}

	if block.Body.SyncAggregate == nil {
		return nil, errors.New("sync aggregate is nil")
	}

	syncCommitteeBits, err := hexutil.Decode(block.Body.SyncAggregate.SyncCommitteeBits)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode sync committee bits `%s`", block.Body.SyncAggregate.SyncCommitteeBits)
	}

	syncCommitteeSignature, err := hexutil.Decode(block.Body.SyncAggregate.SyncCommitteeSignature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode sync committee signature `%s`", block.Body.SyncAggregate.SyncCommitteeSignature)
	}

	return &ethpb.GenericBeaconBlock_Altair{
		Altair: &ethpb.BeaconBlockAltair{
			Slot:          phase0Block.Phase0.Slot,
			ProposerIndex: phase0Block.Phase0.ProposerIndex,
			ParentRoot:    phase0Block.Phase0.ParentRoot,
			StateRoot:     phase0Block.Phase0.StateRoot,
			Body: &ethpb.BeaconBlockBodyAltair{
				RandaoReveal:      phase0Block.Phase0.Body.RandaoReveal,
				Eth1Data:          phase0Block.Phase0.Body.Eth1Data,
				Graffiti:          phase0Block.Phase0.Body.Graffiti,
				ProposerSlashings: phase0Block.Phase0.Body.ProposerSlashings,
				AttesterSlashings: phase0Block.Phase0.Body.AttesterSlashings,
				Attestations:      phase0Block.Phase0.Body.Attestations,
				Deposits:          phase0Block.Phase0.Body.Deposits,
				VoluntaryExits:    phase0Block.Phase0.Body.VoluntaryExits,
				SyncAggregate: &ethpb.SyncAggregate{
					SyncCommitteeBits:      syncCommitteeBits,
					SyncCommitteeSignature: syncCommitteeSignature,
				},
			},
		},
	}, nil
}

func convertRESTBellatrixBlockToProto(block *apimiddleware.BeaconBlockBellatrixJson) (*ethpb.GenericBeaconBlock_Bellatrix, error) {
	if block.Body == nil {
		return nil, errors.New("block body is nil")
	}

	// Call convertRESTAltairBlockToProto to set the altair fields because all the error handling and the heavy lifting
	// has already been done
	altairBlock, err := convertRESTAltairBlockToProto(&apimiddleware.BeaconBlockAltairJson{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    block.ParentRoot,
		StateRoot:     block.StateRoot,
		Body: &apimiddleware.BeaconBlockBodyAltairJson{
			RandaoReveal:      block.Body.RandaoReveal,
			Eth1Data:          block.Body.Eth1Data,
			Graffiti:          block.Body.Graffiti,
			ProposerSlashings: block.Body.ProposerSlashings,
			AttesterSlashings: block.Body.AttesterSlashings,
			Attestations:      block.Body.Attestations,
			Deposits:          block.Body.Deposits,
			VoluntaryExits:    block.Body.VoluntaryExits,
			SyncAggregate:     block.Body.SyncAggregate,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the altair fields of the bellatrix block")
	}

	if block.Body.ExecutionPayload == nil {
		return nil, errors.New("execution payload is nil")
	}

	parentHash, err := hexutil.Decode(block.Body.ExecutionPayload.ParentHash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload parent hash `%s`", block.Body.ExecutionPayload.ParentHash)
	}

	feeRecipient, err := hexutil.Decode(block.Body.ExecutionPayload.FeeRecipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload fee recipient `%s`", block.Body.ExecutionPayload.FeeRecipient)
	}

	stateRoot, err := hexutil.Decode(block.Body.ExecutionPayload.StateRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload state root `%s`", block.Body.ExecutionPayload.StateRoot)
	}

	receiptsRoot, err := hexutil.Decode(block.Body.ExecutionPayload.ReceiptsRoot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload receipts root `%s`", block.Body.ExecutionPayload.ReceiptsRoot)
	}

	logsBloom, err := hexutil.Decode(block.Body.ExecutionPayload.LogsBloom)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload logs bloom `%s`", block.Body.ExecutionPayload.LogsBloom)
	}

	prevRandao, err := hexutil.Decode(block.Body.ExecutionPayload.PrevRandao)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload prev randao `%s`", block.Body.ExecutionPayload.PrevRandao)
	}

	blockNumber, err := strconv.ParseUint(block.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse execution payload block number `%s`", block.Body.ExecutionPayload.BlockNumber)
	}

	gasLimit, err := strconv.ParseUint(block.Body.ExecutionPayload.GasLimit, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse execution payload gas limit `%s`", block.Body.ExecutionPayload.GasLimit)
	}

	gasUsed, err := strconv.ParseUint(block.Body.ExecutionPayload.GasUsed, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse execution payload gas used `%s`", block.Body.ExecutionPayload.GasUsed)
	}

	timestamp, err := strconv.ParseUint(block.Body.ExecutionPayload.TimeStamp, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse execution payload timestamp `%s`", block.Body.ExecutionPayload.TimeStamp)
	}

	extraData, err := hexutil.Decode(block.Body.ExecutionPayload.ExtraData)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload extra data `%s`", block.Body.ExecutionPayload.ExtraData)
	}

	baseFeePerGas := new(big.Int)
	if _, ok := baseFeePerGas.SetString(block.Body.ExecutionPayload.BaseFeePerGas, 10); !ok {
		return nil, errors.Errorf("failed to parse execution payload base fee per gas `%s`", block.Body.ExecutionPayload.BaseFeePerGas)
	}

	blockHash, err := hexutil.Decode(block.Body.ExecutionPayload.BlockHash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode execution payload block hash `%s`", block.Body.ExecutionPayload.BlockHash)
	}

	transactions, err := convertTransactionsToProto(block.Body.ExecutionPayload.Transactions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get execution payload transactions")
	}

	return &ethpb.GenericBeaconBlock_Bellatrix{
		Bellatrix: &ethpb.BeaconBlockBellatrix{
			Slot:          altairBlock.Altair.Slot,
			ProposerIndex: altairBlock.Altair.ProposerIndex,
			ParentRoot:    altairBlock.Altair.ParentRoot,
			StateRoot:     altairBlock.Altair.StateRoot,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				RandaoReveal:      altairBlock.Altair.Body.RandaoReveal,
				Eth1Data:          altairBlock.Altair.Body.Eth1Data,
				Graffiti:          altairBlock.Altair.Body.Graffiti,
				ProposerSlashings: altairBlock.Altair.Body.ProposerSlashings,
				AttesterSlashings: altairBlock.Altair.Body.AttesterSlashings,
				Attestations:      altairBlock.Altair.Body.Attestations,
				Deposits:          altairBlock.Altair.Body.Deposits,
				VoluntaryExits:    altairBlock.Altair.Body.VoluntaryExits,
				SyncAggregate:     altairBlock.Altair.Body.SyncAggregate,
				ExecutionPayload: &enginev1.ExecutionPayload{
					ParentHash:    parentHash,
					FeeRecipient:  feeRecipient,
					StateRoot:     stateRoot,
					ReceiptsRoot:  receiptsRoot,
					LogsBloom:     logsBloom,
					PrevRandao:    prevRandao,
					BlockNumber:   blockNumber,
					GasLimit:      gasLimit,
					GasUsed:       gasUsed,
					Timestamp:     timestamp,
					ExtraData:     extraData,
					BaseFeePerGas: bytesutil.PadTo(bytesutil.BigIntToLittleEndianBytes(baseFeePerGas), 32),
					BlockHash:     blockHash,
					Transactions:  transactions,
				},
			},
		},
	}, nil
}
