package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
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
	isSSZ, err := http2.SszRequested(r)
	if err != nil {
		log.WithError(err).Error("Checking for SSZ failed, defaulting to JSON")
		isSSZ = false
	}
	v1alpha1resp, err := s.V1Alpha1Server.GetBeaconBlock(ctx, v1alpha1req)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wrapper, err := convertGenericBlockToReadOnlySignedBeaconBlock(v1alpha1resp.Block)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	consensusPayload := ""
	if wrapper != nil {
		//get consensus payload value which is the same as the total from the block rewards api
		blockRewards, httpError := s.BlockRewardFetcher.GetBlockRewardsData(ctx, wrapper)
		if httpError != nil {
			http2.WriteError(w, httpError)
			return
		}
		consensusPayload = blockRewards.Total
	}

	w.Header().Set(api.ExecutionPayloadBlindedHeader, fmt.Sprintf("%v", v1alpha1resp.IsBlinded))
	w.Header().Set(api.ExecutionPayloadValueHeader, fmt.Sprintf("%d", v1alpha1resp.PayloadValue))
	w.Header().Set(api.ConsensusPayloadValueHeader, consensusPayload)

	phase0Block, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Phase0)
	if ok {
		// rewards aren't used in phase 0
		handleProducePhase0V3(ctx, w, isSSZ, phase0Block, v1alpha1resp.PayloadValue)
		return
	}
	altairBlock, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Altair)
	if ok {
		handleProduceAltairV3(ctx, w, isSSZ, altairBlock, v1alpha1resp.PayloadValue, consensusPayload)
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
		handleProduceBlindedBellatrixV3(ctx, w, isSSZ, blindedBellatrixBlock, v1alpha1resp.PayloadValue, consensusPayload)
		return
	}
	bellatrixBlock, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Bellatrix)
	if ok {
		handleProduceBellatrixV3(ctx, w, isSSZ, bellatrixBlock, v1alpha1resp.PayloadValue, consensusPayload)
		return
	}
	blindedCapellaBlock, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_BlindedCapella)
	if ok {
		handleProduceBlindedCapellaV3(ctx, w, isSSZ, blindedCapellaBlock, v1alpha1resp.PayloadValue, consensusPayload)
		return
	}
	capellaBlock, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Capella)
	if ok {
		handleProduceCapellaV3(ctx, w, isSSZ, capellaBlock, v1alpha1resp.PayloadValue, consensusPayload)
		return
	}
	blindedDenebBlockContents, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_BlindedDeneb)
	if ok {
		handleProduceBlindedDenebV3(ctx, w, isSSZ, blindedDenebBlockContents, v1alpha1resp.PayloadValue, consensusPayload)
		return
	}
	denebBlockContents, ok := v1alpha1resp.Block.(*eth.GenericBeaconBlock_Deneb)
	if ok {
		handleProduceDenebV3(ctx, w, isSSZ, denebBlockContents, v1alpha1resp.PayloadValue, consensusPayload)
		return
	}
}

// convertGenericBlockToReadOnlySignedBeaconBlock will create a fake wrapper object to call the reward functions as some require a signed block even if not using a signature.
// TODO: we should remove this and update the associated functions in another PR
func convertGenericBlockToReadOnlySignedBeaconBlock(i interface{}) (interfaces.ReadOnlySignedBeaconBlock, error) {
	switch b := i.(type) {
	case *eth.GenericBeaconBlock_Phase0:
		//ignore for phase0
		return nil, nil
	case *eth.GenericBeaconBlock_Altair:
		return blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_Altair{Altair: &eth.SignedBeaconBlockAltair{Block: b.Altair}})
	case *eth.GenericBeaconBlock_Bellatrix:
		return blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_Bellatrix{Bellatrix: &eth.SignedBeaconBlockBellatrix{Block: b.Bellatrix}})
	case *eth.GenericBeaconBlock_BlindedBellatrix:
		return blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_BlindedBellatrix{BlindedBellatrix: &eth.SignedBlindedBeaconBlockBellatrix{Block: b.BlindedBellatrix}})
	case *eth.GenericBeaconBlock_Capella:
		return blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_Capella{Capella: &eth.SignedBeaconBlockCapella{Block: b.Capella}})
	case *eth.GenericBeaconBlock_BlindedCapella:
		return blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_BlindedCapella{BlindedCapella: &eth.SignedBlindedBeaconBlockCapella{Block: b.BlindedCapella}})
	case *eth.GenericBeaconBlock_Deneb:
		// no need for sidecar
		return blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_Deneb{Deneb: &eth.SignedBeaconBlockAndBlobsDeneb{Block: &eth.SignedBeaconBlockDeneb{Block: b.Deneb.Block}}})
	case *eth.GenericBeaconBlock_BlindedDeneb:
		// no need for sidecar
		return blocks.NewSignedBeaconBlock(&eth.GenericSignedBeaconBlock_BlindedDeneb{BlindedDeneb: &eth.SignedBlindedBeaconBlockAndBlobsDeneb{SignedBlindedBlock: &eth.SignedBlindedBeaconBlockDeneb{Message: b.BlindedDeneb.Block}}})
	default:
		return nil, fmt.Errorf("type %T is not supported", b)
	}
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
		ConsensusPayloadValue:   "",                              // rewards not applicable before altair
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
		ConsensusPayloadValue:   consensusPayloadValue,
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
		ConsensusPayloadValue:   consensusPayloadValue,
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
		ConsensusPayloadValue:   consensusPayloadValue,
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
		ConsensusPayloadValue:   consensusPayloadValue,
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
		ConsensusPayloadValue:   consensusPayloadValue,
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
		ConsensusPayloadValue:   consensusPayloadValue,
		Data:                    jsonBytes,
	})
}

func handleProduceDenebV3(
	ctx context.Context,
	w http.ResponseWriter,
	isSSZ bool,
	blk *eth.GenericBeaconBlock_Deneb,
	executionPayloadValue uint64,
	consensusPayloadValue string,
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
		ConsensusPayloadValue:   consensusPayloadValue,
		Data:                    jsonBytes,
	})
}
