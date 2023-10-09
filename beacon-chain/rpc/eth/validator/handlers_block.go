package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/rewards"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ProduceBlockV3 Requests a beacon node to produce a valid block, which can then be signed by a validator. The
// returned block may be blinded or unblinded, depending on the current state of the network as
// decided by the execution and beacon nodes.
// The beacon node must return an unblinded block if it obtains the execution payload from its
// paired execution node. It must only return a blinded block if it obtains the execution payload
// header from an MEV relay.
// Metadata in the response indicates the type of block produced, and the supported types of block
// will be added to as forks progress.
func (s *Server) ProduceBlockV3(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.ProduceBlockV3")
	defer span.End()

	if shared.IsSyncing(r.Context(), w, s.SyncChecker, s.HeadFetcher, s.TimeFetcher, s.OptimisticModeFetcher) {
		return
	}
	segments := strings.Split(r.URL.Path, "/")
	rawSlot := segments[len(segments)-1]
	rawRandaoReveal := r.URL.Query().Get("randao_reveal")
	rawGraffiti := r.URL.Query().Get("graffiti")
	rawSkipRandaoVerification := r.URL.Query().Get("skip_randao_verification")

	slot, valid := shared.ValidateUint(w, "slot", rawSlot)
	if !valid {
		return
	}

	var randaoReveal []byte
	if rawSkipRandaoVerification == "true" {
		randaoReveal = primitives.PointAtInfinity
	} else {
		rr, err := shared.DecodeHexWithLength(rawRandaoReveal, fieldparams.BLSSignatureLength)
		if err != nil {
			http2.HandleError(w, errors.Wrap(err, "unable to decode randao reveal").Error(), http.StatusBadRequest)
			return
		}
		randaoReveal = rr
	}
	var graffiti []byte
	if rawGraffiti != "" {
		g, err := shared.DecodeHexWithLength(rawGraffiti, 32)
		if err != nil {
			http2.HandleError(w, errors.Wrap(err, "unable to decode graffiti").Error(), http.StatusBadRequest)
			return
		}
		graffiti = g
	}

	s.produceBlockV3(ctx, w, r, &eth.BlockRequest{
		Slot:         primitives.Slot(slot),
		RandaoReveal: randaoReveal,
		Graffiti:     graffiti,
		SkipMevBoost: false,
	})
}

func (s *Server) produceBlockV3(ctx context.Context, w http.ResponseWriter, r *http.Request, v1alpha1req *eth.BlockRequest) {
	isSSZ := http2.SszRequested(r)
	if !isSSZ {
		log.Error("Checking for SSZ failed, defaulting to JSON")
	}
	v1alpha1resp, err := s.V1Alpha1Server.GetBeaconBlock(ctx, v1alpha1req)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	consensusBlockValue, httpError := getConsensusBlockValue(ctx, s.BlockRewardFetcher, v1alpha1resp.Block)
	if httpError != nil {
		http2.WriteError(w, httpError)
		return
	}

	w.Header().Set(api.ExecutionPayloadBlindedHeader, fmt.Sprintf("%v", v1alpha1resp.IsBlinded))
	w.Header().Set(api.ExecutionPayloadValueHeader, fmt.Sprintf("%d", v1alpha1resp.PayloadValue))
	w.Header().Set(api.ConsensusBlockValueHeader, consensusBlockValue)

	phase0Block, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Phase0)
	if ok {
		// rewards aren't used in phase 0
		handleProducePhase0V3(ctx, w, isSSZ, phase0Block, v1alpha1resp.PayloadValue)
		return
	}
	altairBlock, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Altair)
	if ok {
		handleProduceAltairV3(ctx, w, isSSZ, altairBlock, v1alpha1resp.PayloadValue, consensusBlockValue)
		return
	}
	optimistic, err := s.OptimisticModeFetcher.IsOptimistic(ctx)
	if err != nil {
		http2.HandleError(w, errors.Wrap(err, "Could not determine if the node is a optimistic node").Error(), http.StatusInternalServerError)
		return
	}
	if optimistic {
		http2.HandleError(w, "The node is currently optimistic and cannot serve validators", http.StatusServiceUnavailable)
		return
	}
	blindedBellatrixBlock, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_BlindedBellatrix)
	if ok {
		handleProduceBlindedBellatrixV3(ctx, w, isSSZ, blindedBellatrixBlock, v1alpha1resp.PayloadValue, consensusBlockValue)
		return
	}
	bellatrixBlock, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Bellatrix)
	if ok {
		handleProduceBellatrixV3(ctx, w, isSSZ, bellatrixBlock, v1alpha1resp.PayloadValue, consensusBlockValue)
		return
	}
	blindedCapellaBlock, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_BlindedCapella)
	if ok {
		handleProduceBlindedCapellaV3(ctx, w, isSSZ, blindedCapellaBlock, v1alpha1resp.PayloadValue, consensusBlockValue)
		return
	}
	capellaBlock, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Capella)
	if ok {
		handleProduceCapellaV3(ctx, w, isSSZ, capellaBlock, v1alpha1resp.PayloadValue, consensusBlockValue)
		return
	}
	blindedDenebBlockContents, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_BlindedDeneb)
	if ok {
		handleProduceBlindedDenebV3(ctx, w, isSSZ, blindedDenebBlockContents, v1alpha1resp.PayloadValue, consensusBlockValue)
		return
	}
	denebBlockContents, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Deneb)
	if ok {
		handleProduceDenebV3(ctx, w, isSSZ, denebBlockContents, v1alpha1resp.PayloadValue, consensusBlockValue)
		return
	}
}

func getConsensusBlockValue(ctx context.Context, blockRewardsFetcher rewards.BlockRewardsFetcher, i interface{} /* block as argument */) (string, *http2.DefaultErrorJson) {
	var wrapper interfaces.ReadOnlySignedBeaconBlock
	var err error

	// TODO: we should not require this fake signed wrapper and fix associated functions in the future.
	switch b := i.(type) {
	case *eth.GenericBeaconBlock_Phase0:
		//ignore for phase0
		return "", nil
	case *eth.GenericBeaconBlock_Altair:
		wrapper, err = blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_Altair{Altair: &eth.SignedBeaconBlockAltair{Block: b.Altair}})
	case *eth.GenericBeaconBlock_Bellatrix:
		wrapper, err = blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_Bellatrix{Bellatrix: &eth.SignedBeaconBlockBellatrix{Block: b.Bellatrix}})
	case *eth.GenericBeaconBlock_BlindedBellatrix:
		wrapper, err = blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: &eth.SignedBlindedBeaconBlockBellatrix{Block: b.BlindedBellatrix}})
	case *eth.GenericBeaconBlock_Capella:
		wrapper, err = blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_Capella{Capella: &eth.SignedBeaconBlockCapella{Block: b.Capella}})
	case *eth.GenericBeaconBlock_BlindedCapella:
		wrapper, err = blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_BlindedCapella{BlindedCapella: &eth.SignedBlindedBeaconBlockCapella{Block: b.BlindedCapella}})
	case *eth.GenericBeaconBlock_Deneb:
		// no need for sidecar
		wrapper, err = blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_Deneb{Deneb: &eth.SignedBeaconBlockAndBlobsDeneb{Block: &eth.SignedBeaconBlockDeneb{Block: b.Deneb.Block}}})
	case *eth.GenericBeaconBlock_BlindedDeneb:
		// no need for sidecar
		wrapper, err = blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_BlindedDeneb{BlindedDeneb: &eth.SignedBlindedBeaconBlockAndBlobsDeneb{SignedBlindedBlock: &eth.SignedBlindedBeaconBlockDeneb{Message: b.BlindedDeneb.Block}}})
	default:
		return "", &http2.DefaultErrorJson{
			Message: fmt.Errorf("type %T is not supported", b).Error(),
			Code:    http.StatusInternalServerError,
		}
	}
	if err != nil {
		return "", &http2.DefaultErrorJson{
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		}
	}

	//get consensus payload value which is the same as the total from the block rewards api
	blockRewards, httpError := blockRewardsFetcher.GetBlockRewardsData(ctx, wrapper)
	if httpError != nil {
		return "", httpError
	}
	return blockRewards.Total, nil
}

func handleProducePhase0V3(
	ctx context.Context,
	w http.ResponseWriter,
	isSSZ bool,
	blk *eth.GenericBeaconBlock_Phase0,
	payloadValue uint64,
) {
	_, span := trace.StartSpan(ctx, "validator.ProduceBlockV3.internal.handleProducePhase0V3")
	defer span.End()
	if isSSZ {
		sszResp, err := blk.Phase0.MarshalSSZ()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http2.WriteSsz(w, sszResp, "phase0Block.ssz")
		return
	}
	block, err := shared.BeaconBlockFromConsensus(blk.Phase0)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonBytes, err := json.Marshal(block)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteJson(w, &ProduceBlockV3Response{
		Version:                 version.String(version.Phase0),
		ExecutionPayloadBlinded: false,
		ExecutionPayloadValue:   fmt.Sprintf("%d", payloadValue), // mev not available at this point
		ConsensusBlockValue:     "",                              // rewards not applicable before altair
		Data:                    jsonBytes,
	})
}

func handleProduceAltairV3(
	ctx context.Context,
	w http.ResponseWriter,
	isSSZ bool,
	blk *eth.GenericBeaconBlock_Altair,
	executionPayloadValue uint64,
	consensusPayloadValue string,
) {
	_, span := trace.StartSpan(ctx, "validator.ProduceBlockV3.internal.handleProduceAltairV3")
	defer span.End()

	if isSSZ {
		sszResp, err := blk.Altair.MarshalSSZ()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http2.WriteSsz(w, sszResp, "altairBlock.ssz")
		return
	}
	block, err := shared.BeaconBlockAltairFromConsensus(blk.Altair)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonBytes, err := json.Marshal(block)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteJson(w, &ProduceBlockV3Response{
		Version:                 version.String(version.Altair),
		ExecutionPayloadBlinded: false,
		ExecutionPayloadValue:   fmt.Sprintf("%d", executionPayloadValue), // mev not available at this point
		ConsensusBlockValue:     consensusPayloadValue,
		Data:                    jsonBytes,
	})
}

func handleProduceBellatrixV3(
	ctx context.Context,
	w http.ResponseWriter,
	isSSZ bool,
	blk *eth.GenericBeaconBlock_Bellatrix,
	executionPayloadValue uint64,
	consensusPayloadValue string,
) {
	_, span := trace.StartSpan(ctx, "validator.ProduceBlockV3.internal.handleProduceBellatrixV3")
	defer span.End()
	if isSSZ {
		sszResp, err := blk.Bellatrix.MarshalSSZ()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http2.WriteSsz(w, sszResp, "bellatrixBlock.ssz")
		return
	}
	block, err := shared.BeaconBlockBellatrixFromConsensus(blk.Bellatrix)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonBytes, err := json.Marshal(block)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteJson(w, &ProduceBlockV3Response{
		Version:                 version.String(version.Bellatrix),
		ExecutionPayloadBlinded: false,
		ExecutionPayloadValue:   fmt.Sprintf("%d", executionPayloadValue), // mev not available at this point
		ConsensusBlockValue:     consensusPayloadValue,
		Data:                    jsonBytes,
	})
}

func handleProduceBlindedBellatrixV3(
	ctx context.Context,
	w http.ResponseWriter,
	isSSZ bool,
	blk *eth.GenericBeaconBlock_BlindedBellatrix,
	executionPayloadValue uint64,
	consensusPayloadValue string,
) {
	_, span := trace.StartSpan(ctx, "validator.ProduceBlockV3.internal.handleProduceBlindedBellatrixV3")
	defer span.End()
	if isSSZ {
		sszResp, err := blk.BlindedBellatrix.MarshalSSZ()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http2.WriteSsz(w, sszResp, "blindedBellatrixBlock.ssz")
		return
	}
	block, err := shared.BlindedBeaconBlockBellatrixFromConsensus(blk.BlindedBellatrix)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonBytes, err := json.Marshal(block)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteJson(w, &ProduceBlockV3Response{
		Version:                 version.String(version.Bellatrix),
		ExecutionPayloadBlinded: true,
		ExecutionPayloadValue:   fmt.Sprintf("%d", executionPayloadValue),
		ConsensusBlockValue:     consensusPayloadValue,
		Data:                    jsonBytes,
	})
}

func handleProduceBlindedCapellaV3(
	ctx context.Context,
	w http.ResponseWriter,
	isSSZ bool,
	blk *eth.GenericBeaconBlock_BlindedCapella,
	executionPayloadValue uint64,
	consensusPayloadValue string,
) {
	_, span := trace.StartSpan(ctx, "validator.ProduceBlockV3.internal.handleProduceBlindedCapellaV3")
	defer span.End()
	if isSSZ {
		sszResp, err := blk.BlindedCapella.MarshalSSZ()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http2.WriteSsz(w, sszResp, "blindedCapellaBlock.ssz")
		return
	}
	block, err := shared.BlindedBeaconBlockCapellaFromConsensus(blk.BlindedCapella)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonBytes, err := json.Marshal(block)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteJson(w, &ProduceBlockV3Response{
		Version:                 version.String(version.Capella),
		ExecutionPayloadBlinded: true,
		ExecutionPayloadValue:   fmt.Sprintf("%d", executionPayloadValue),
		ConsensusBlockValue:     consensusPayloadValue,
		Data:                    jsonBytes,
	})
}

func handleProduceCapellaV3(
	ctx context.Context,
	w http.ResponseWriter,
	isSSZ bool,
	blk *eth.GenericBeaconBlock_Capella,
	executionPayloadValue uint64,
	consensusPayloadValue string,
) {
	_, span := trace.StartSpan(ctx, "validator.ProduceBlockV3.internal.handleProduceCapellaV3")
	defer span.End()
	if isSSZ {
		sszResp, err := blk.Capella.MarshalSSZ()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http2.WriteSsz(w, sszResp, "capellaBlock.ssz")
		return
	}
	block, err := shared.BeaconBlockCapellaFromConsensus(blk.Capella)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonBytes, err := json.Marshal(block)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteJson(w, &ProduceBlockV3Response{
		Version:                 version.String(version.Capella),
		ExecutionPayloadBlinded: false,
		ExecutionPayloadValue:   fmt.Sprintf("%d", executionPayloadValue), // mev not available at this point
		ConsensusBlockValue:     consensusPayloadValue,
		Data:                    jsonBytes,
	})
}

func handleProduceBlindedDenebV3(
	ctx context.Context,
	w http.ResponseWriter,
	isSSZ bool,
	blk *eth.GenericBeaconBlock_BlindedDeneb,
	executionPayloadValue uint64,
	consensusPayloadValue string,
) {
	_, span := trace.StartSpan(ctx, "validator.ProduceBlockV3.internal.handleProduceBlindedDenebV3")
	defer span.End()
	if isSSZ {
		sszResp, err := blk.BlindedDeneb.MarshalSSZ()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http2.WriteSsz(w, sszResp, "blindedDenebBlockContents.ssz")
		return
	}
	blockContents, err := shared.BlindedBeaconBlockContentsDenebFromConsensus(blk.BlindedDeneb)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonBytes, err := json.Marshal(blockContents)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteJson(w, &ProduceBlockV3Response{
		Version:                 version.String(version.Deneb),
		ExecutionPayloadBlinded: true,
		ExecutionPayloadValue:   fmt.Sprintf("%d", executionPayloadValue),
		ConsensusBlockValue:     consensusPayloadValue,
		Data:                    jsonBytes,
	})
}

func handleProduceDenebV3(
	ctx context.Context,
	w http.ResponseWriter,
	isSSZ bool,
	blk *eth.GenericBeaconBlock_Deneb,
	executionPayloadValue uint64,
	consensusBlockValue string,
) {
	_, span := trace.StartSpan(ctx, "validator.ProduceBlockV3.internal.handleProduceDenebV3")
	defer span.End()
	if isSSZ {
		sszResp, err := blk.Deneb.MarshalSSZ()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http2.WriteSsz(w, sszResp, "denebBlockContents.ssz")
		return
	}
	blockContents, err := shared.BeaconBlockContentsDenebFromConsensus(blk.Deneb)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonBytes, err := json.Marshal(blockContents)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteJson(w, &ProduceBlockV3Response{
		Version:                 version.String(version.Deneb),
		ExecutionPayloadBlinded: false,
		ExecutionPayloadValue:   fmt.Sprintf("%d", executionPayloadValue), // mev not available at this point
		ConsensusBlockValue:     consensusBlockValue,
		Data:                    jsonBytes,
	})
}
