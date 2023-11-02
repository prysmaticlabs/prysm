package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/api"
	corehelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	consensus_types "github.com/prysmaticlabs/prysm/v4/consensus-types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

const (
	broadcastValidationQueryParam               = "broadcast_validation"
	broadcastValidationConsensus                = "consensus"
	broadcastValidationConsensusAndEquivocation = "consensus_and_equivocation"
)

type handled bool

// GetBlock retrieves block details for given block ID.
//
// DEPRECATED: please use GetBlockV2 instead
func (s *Server) GetBlock(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetBlock")
	defer span.End()

	blockId := mux.Vars(r)["block_id"]
	if blockId == "" {
		http2.HandleError(w, "block_id is required in URL params", http.StatusBadRequest)
		return
	}
	blk, err := s.Blocker.Block(ctx, []byte(blockId))
	if !shared.WriteBlockFetchError(w, blk, err) {
		return
	}

	if http2.SszRequested(r) {
		s.getBlockSSZ(ctx, w, blk)
	} else {
		s.getBlock(ctx, w, blk)
	}
}

// getBlock returns the JSON-serialized version of the beacon block for given block ID.
func (s *Server) getBlock(ctx context.Context, w http.ResponseWriter, blk interfaces.ReadOnlySignedBeaconBlock) {
	v2Resp, err := s.getBlockPhase0(ctx, blk)
	if err != nil {
		http2.HandleError(w, "Could not get block: "+err.Error(), http.StatusInternalServerError)
		return
	}
	resp := &GetBlockResponse{Data: v2Resp.Data}
	http2.WriteJson(w, resp)
}

// getBlockSSZ returns the SSZ-serialized version of the becaon block for given block ID.
func (s *Server) getBlockSSZ(ctx context.Context, w http.ResponseWriter, blk interfaces.ReadOnlySignedBeaconBlock) {
	resp, err := s.getBlockPhase0SSZ(ctx, blk)
	if err != nil {
		http2.HandleError(w, "Could not get block: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http2.WriteSsz(w, resp, "beacon_block.ssz")
}

// GetBlockV2 retrieves block details for given block ID.
func (s *Server) GetBlockV2(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetBlockV2")
	defer span.End()

	blockId := mux.Vars(r)["block_id"]
	if blockId == "" {
		http2.HandleError(w, "block_id is required in URL params", http.StatusBadRequest)
		return
	}
	blk, err := s.Blocker.Block(ctx, []byte(blockId))
	if !shared.WriteBlockFetchError(w, blk, err) {
		return
	}

	if http2.SszRequested(r) {
		s.getBlockSSZV2(ctx, w, blk)
	} else {
		s.getBlockV2(ctx, w, blk)
	}
}

// getBlockV2 returns the JSON-serialized version of the beacon block for given block ID.
func (s *Server) getBlockV2(ctx context.Context, w http.ResponseWriter, blk interfaces.ReadOnlySignedBeaconBlock) {
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		http2.HandleError(w, "Could not get block root "+err.Error(), http.StatusInternalServerError)
		return
	}
	finalized := s.FinalizationFetcher.IsFinalized(ctx, blkRoot)

	getBlockHandler := func(get func(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error)) handled {
		result, err := get(ctx, blk)
		if result != nil {
			result.Finalized = finalized
			w.Header().Set(api.VersionHeader, result.Version)
			http2.WriteJson(w, result)
			return true
		}
		// ErrUnsupportedField means that we have another block type
		if !errors.Is(err, consensus_types.ErrUnsupportedField) {
			http2.HandleError(w, "Could not get signed beacon block: "+err.Error(), http.StatusInternalServerError)
			return true
		}
		return false
	}

	if getBlockHandler(s.getBlockDeneb) {
		return
	}
	if getBlockHandler(s.getBlockCapella) {
		return
	}
	if getBlockHandler(s.getBlockBellatrix) {
		return
	}
	if getBlockHandler(s.getBlockAltair) {
		return
	}
	if getBlockHandler(s.getBlockPhase0) {
		return
	}
	http2.HandleError(w, fmt.Sprintf("Unknown block type %T", blk), http.StatusInternalServerError)
}

// getBlockSSZV2 returns the SSZ-serialized version of the beacon block for given block ID.
func (s *Server) getBlockSSZV2(ctx context.Context, w http.ResponseWriter, blk interfaces.ReadOnlySignedBeaconBlock) {
	getBlockHandler := func(get func(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock) ([]byte, error), ver string) handled {
		result, err := get(ctx, blk)
		if result != nil {
			w.Header().Set(api.VersionHeader, ver)
			http2.WriteSsz(w, result, "beacon_block.ssz")
			return true
		}
		// ErrUnsupportedField means that we have another block type
		if !errors.Is(err, consensus_types.ErrUnsupportedField) {
			http2.HandleError(w, "Could not get signed beacon block: "+err.Error(), http.StatusInternalServerError)
			return true
		}
		return false
	}

	if getBlockHandler(s.getBlockDenebSSZ, version.String(version.Deneb)) {
		return
	}
	if getBlockHandler(s.getBlockCapellaSSZ, version.String(version.Capella)) {
		return
	}
	if getBlockHandler(s.getBlockBellatrixSSZ, version.String(version.Bellatrix)) {
		return
	}
	if getBlockHandler(s.getBlockAltairSSZ, version.String(version.Altair)) {
		return
	}
	if getBlockHandler(s.getBlockPhase0SSZ, version.String(version.Phase0)) {
		return
	}
	http2.HandleError(w, fmt.Sprintf("Unknown block type %T", blk), http.StatusInternalServerError)
}

// GetBlindedBlock retrieves blinded block for given block id.
func (s *Server) GetBlindedBlock(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetBlindedBlock")
	defer span.End()

	blockId := mux.Vars(r)["block_id"]
	if blockId == "" {
		http2.HandleError(w, "block_id is required in URL params", http.StatusBadRequest)
		return
	}
	blk, err := s.Blocker.Block(ctx, []byte(blockId))
	if !shared.WriteBlockFetchError(w, blk, err) {
		return
	}

	if http2.SszRequested(r) {
		s.getBlindedBlockSSZ(ctx, w, blk)
	} else {
		s.getBlindedBlock(ctx, w, blk)
	}
}

// getBlindedBlock returns the JSON-serialized version of the blinded beacon block for given block id.
func (s *Server) getBlindedBlock(ctx context.Context, w http.ResponseWriter, blk interfaces.ReadOnlySignedBeaconBlock) {
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		http2.HandleError(w, "Could not get block root "+err.Error(), http.StatusInternalServerError)
		return
	}
	finalized := s.FinalizationFetcher.IsFinalized(ctx, blkRoot)

	getBlockHandler := func(get func(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error)) handled {
		result, err := get(ctx, blk)
		if result != nil {
			result.Finalized = finalized
			w.Header().Set(api.VersionHeader, result.Version)
			http2.WriteJson(w, result)
			return true
		}
		// ErrUnsupportedField means that we have another block type
		if !errors.Is(err, consensus_types.ErrUnsupportedField) {
			http2.HandleError(w, "Could not get signed beacon block: "+err.Error(), http.StatusInternalServerError)
			return true
		}
		return false
	}

	if getBlockHandler(s.getBlockPhase0) {
		return
	}
	if getBlockHandler(s.getBlockAltair) {
		return
	}
	if getBlockHandler(s.getBlindedBlockBellatrix) {
		return
	}
	if getBlockHandler(s.getBlindedBlockCapella) {
		return
	}
	if getBlockHandler(s.getBlindedBlockDeneb) {
		return
	}
	http2.HandleError(w, fmt.Sprintf("Unknown block type %T", blk), http.StatusInternalServerError)
}

// getBlindedBlockSSZ returns the SSZ-serialized version of the blinded beacon block for given block id.
func (s *Server) getBlindedBlockSSZ(ctx context.Context, w http.ResponseWriter, blk interfaces.ReadOnlySignedBeaconBlock) {
	getBlockHandler := func(get func(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock) ([]byte, error), ver string) handled {
		result, err := get(ctx, blk)
		if result != nil {
			w.Header().Set(api.VersionHeader, ver)
			http2.WriteSsz(w, result, "beacon_block.ssz")
			return true
		}
		// ErrUnsupportedField means that we have another block type
		if !errors.Is(err, consensus_types.ErrUnsupportedField) {
			http2.HandleError(w, "Could not get signed beacon block: "+err.Error(), http.StatusInternalServerError)
			return true
		}
		return false
	}

	if getBlockHandler(s.getBlockPhase0SSZ, version.String(version.Phase0)) {
		return
	}
	if getBlockHandler(s.getBlockAltairSSZ, version.String(version.Altair)) {
		return
	}
	if getBlockHandler(s.getBlindedBlockBellatrixSSZ, version.String(version.Bellatrix)) {
		return
	}
	if getBlockHandler(s.getBlindedBlockCapellaSSZ, version.String(version.Capella)) {
		return
	}
	if getBlockHandler(s.getBlindedBlockDenebSSZ, version.String(version.Deneb)) {
		return
	}
	http2.HandleError(w, fmt.Sprintf("Unknown block type %T", blk), http.StatusInternalServerError)
}

func (*Server) getBlockPhase0(_ context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error) {
	consensusBlk, err := blk.PbPhase0Block()
	if err != nil {
		return nil, err
	}
	if consensusBlk == nil {
		return nil, errNilBlock
	}
	respBlk, err := shared.SignedBeaconBlockFromConsensus(consensusBlk)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(respBlk.Message)
	if err != nil {
		return nil, err
	}
	return &GetBlockV2Response{
		Version:             version.String(version.Phase0),
		ExecutionOptimistic: false,
		Data: &SignedBlock{
			Message:   jsonBytes,
			Signature: respBlk.Signature,
		},
	}, nil
}

func (*Server) getBlockAltair(_ context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error) {
	consensusBlk, err := blk.PbAltairBlock()
	if err != nil {
		return nil, err
	}
	if consensusBlk == nil {
		return nil, errNilBlock
	}
	respBlk, err := shared.SignedBeaconBlockAltairFromConsensus(consensusBlk)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(respBlk.Message)
	if err != nil {
		return nil, err
	}
	return &GetBlockV2Response{
		Version:             version.String(version.Altair),
		ExecutionOptimistic: false,
		Data: &SignedBlock{
			Message:   jsonBytes,
			Signature: respBlk.Signature,
		},
	}, nil
}

func (s *Server) getBlockBellatrix(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error) {
	consensusBlk, err := blk.PbBellatrixBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			blindedConsensusBlk, err := blk.PbBlindedBellatrixBlock()
			if err != nil {
				return nil, err
			}
			if blindedConsensusBlk == nil {
				return nil, errNilBlock
			}
			fullBlk, err := s.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
			if err != nil {
				return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
			}
			consensusBlk, err = fullBlk.PbBellatrixBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if consensusBlk == nil {
		return nil, errNilBlock
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	respBlk, err := shared.SignedBeaconBlockBellatrixFromConsensus(consensusBlk)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(respBlk.Message)
	if err != nil {
		return nil, err
	}
	return &GetBlockV2Response{
		Version:             version.String(version.Bellatrix),
		ExecutionOptimistic: isOptimistic,
		Data: &SignedBlock{
			Message:   jsonBytes,
			Signature: respBlk.Signature,
		},
	}, nil
}

func (s *Server) getBlockCapella(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error) {
	consensusBlk, err := blk.PbCapellaBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			blindedConsensusBlk, err := blk.PbBlindedCapellaBlock()
			if err != nil {
				return nil, err
			}
			if blindedConsensusBlk == nil {
				return nil, errNilBlock
			}
			fullBlk, err := s.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
			if err != nil {
				return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
			}
			consensusBlk, err = fullBlk.PbCapellaBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if consensusBlk == nil {
		return nil, errNilBlock
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	respBlk, err := shared.SignedBeaconBlockCapellaFromConsensus(consensusBlk)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(respBlk.Message)
	if err != nil {
		return nil, err
	}
	return &GetBlockV2Response{
		Version:             version.String(version.Capella),
		ExecutionOptimistic: isOptimistic,
		Data: &SignedBlock{
			Message:   jsonBytes,
			Signature: respBlk.Signature,
		},
	}, nil
}

func (s *Server) getBlockDeneb(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error) {
	consensusBlk, err := blk.PbDenebBlock()
	if err != nil {
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			blindedConsensusBlk, err := blk.PbBlindedDenebBlock()
			if err != nil {
				return nil, err
			}
			if blindedConsensusBlk == nil {
				return nil, errNilBlock
			}
			fullBlk, err := s.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
			if err != nil {
				return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
			}
			consensusBlk, err = fullBlk.PbDenebBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if consensusBlk == nil {
		return nil, errNilBlock
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	respBlk, err := shared.SignedBeaconBlockDenebFromConsensus(consensusBlk)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(respBlk.Message)
	if err != nil {
		return nil, err
	}
	return &GetBlockV2Response{
		Version:             version.String(version.Deneb),
		ExecutionOptimistic: isOptimistic,
		Data: &SignedBlock{
			Message:   jsonBytes,
			Signature: respBlk.Signature,
		},
	}, nil
}

func (*Server) getBlockPhase0SSZ(_ context.Context, blk interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	consensusBlk, err := blk.PbPhase0Block()
	if err != nil {
		return nil, err
	}
	if consensusBlk == nil {
		return nil, errNilBlock
	}
	sszData, err := consensusBlk.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return sszData, nil
}

func (*Server) getBlockAltairSSZ(_ context.Context, blk interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	consensusBlk, err := blk.PbAltairBlock()
	if err != nil {
		return nil, err
	}
	if consensusBlk == nil {
		return nil, errNilBlock
	}
	sszData, err := consensusBlk.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return sszData, nil
}

func (s *Server) getBlockBellatrixSSZ(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	consensusBlk, err := blk.PbBellatrixBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			blindedConsensusBlk, err := blk.PbBlindedBellatrixBlock()
			if err != nil {
				return nil, err
			}
			if blindedConsensusBlk == nil {
				return nil, errNilBlock
			}
			fullBlk, err := s.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
			if err != nil {
				return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
			}
			consensusBlk, err = fullBlk.PbBellatrixBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if consensusBlk == nil {
		return nil, errNilBlock
	}
	sszData, err := consensusBlk.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return sszData, nil
}

func (s *Server) getBlockCapellaSSZ(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	consensusBlk, err := blk.PbCapellaBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			blindedConsensusBlk, err := blk.PbBlindedCapellaBlock()
			if err != nil {
				return nil, err
			}
			if blindedConsensusBlk == nil {
				return nil, errNilBlock
			}
			fullBlk, err := s.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
			if err != nil {
				return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
			}
			consensusBlk, err = fullBlk.PbCapellaBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if consensusBlk == nil {
		return nil, errNilBlock
	}
	sszData, err := consensusBlk.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return sszData, nil
}

func (s *Server) getBlockDenebSSZ(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	consensusBlk, err := blk.PbDenebBlock()
	if err != nil {
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			blindedConsensusBlk, err := blk.PbBlindedDenebBlock()
			if err != nil {
				return nil, err
			}
			if blindedConsensusBlk == nil {
				return nil, errNilBlock
			}
			fullBlk, err := s.ExecutionPayloadReconstructor.ReconstructFullBlock(ctx, blk)
			if err != nil {
				return nil, errors.Wrapf(err, "could not reconstruct full execution payload to create signed beacon block")
			}
			consensusBlk, err = fullBlk.PbDenebBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if consensusBlk == nil {
		return nil, errNilBlock
	}
	sszData, err := consensusBlk.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return sszData, nil
}

func (s *Server) getBlindedBlockBellatrix(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error) {
	blindedConsensusBlk, err := blk.PbBlindedBellatrixBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			consensusBlk, err := blk.PbBellatrixBlock()
			if err != nil {
				return nil, err
			}
			if consensusBlk == nil {
				return nil, errNilBlock
			}
			blkInterface, err := blk.ToBlinded()
			if err != nil {
				return nil, errors.Wrapf(err, "could not convert block to blinded block")
			}
			blindedConsensusBlk, err = blkInterface.PbBlindedBellatrixBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if blindedConsensusBlk == nil {
		return nil, errNilBlock
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	respBlk, err := shared.SignedBlindedBeaconBlockBellatrixFromConsensus(blindedConsensusBlk)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(respBlk.Message)
	if err != nil {
		return nil, err
	}
	return &GetBlockV2Response{
		Version:             version.String(version.Bellatrix),
		ExecutionOptimistic: isOptimistic,
		Data: &SignedBlock{
			Message:   jsonBytes,
			Signature: respBlk.Signature,
		},
	}, nil
}

func (s *Server) getBlindedBlockCapella(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error) {
	blindedConsensusBlk, err := blk.PbBlindedCapellaBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			consensusBlk, err := blk.PbCapellaBlock()
			if err != nil {
				return nil, err
			}
			if consensusBlk == nil {
				return nil, errNilBlock
			}
			blkInterface, err := blk.ToBlinded()
			if err != nil {
				return nil, errors.Wrapf(err, "could not convert block to blinded block")
			}
			blindedConsensusBlk, err = blkInterface.PbBlindedCapellaBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if blindedConsensusBlk == nil {
		return nil, errNilBlock
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	respBlk, err := shared.SignedBlindedBeaconBlockCapellaFromConsensus(blindedConsensusBlk)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(respBlk.Message)
	if err != nil {
		return nil, err
	}
	return &GetBlockV2Response{
		Version:             version.String(version.Capella),
		ExecutionOptimistic: isOptimistic,
		Data: &SignedBlock{
			Message:   jsonBytes,
			Signature: respBlk.Signature,
		},
	}, nil
}

func (s *Server) getBlindedBlockDeneb(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) (*GetBlockV2Response, error) {
	blindedConsensusBlk, err := blk.PbBlindedDenebBlock()
	if err != nil {
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			consensusBlk, err := blk.PbDenebBlock()
			if err != nil {
				return nil, err
			}
			if consensusBlk == nil {
				return nil, errNilBlock
			}
			blkInterface, err := blk.ToBlinded()
			if err != nil {
				return nil, errors.Wrapf(err, "could not convert block to blinded block")
			}
			blindedConsensusBlk, err = blkInterface.PbBlindedDenebBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if blindedConsensusBlk == nil {
		return nil, errNilBlock
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get block root")
	}
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		return nil, errors.Wrapf(err, "could not check if block is optimistic")
	}
	respBlk, err := shared.SignedBlindedBeaconBlockDenebFromConsensus(blindedConsensusBlk)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(respBlk.Message)
	if err != nil {
		return nil, err
	}
	return &GetBlockV2Response{
		Version:             version.String(version.Deneb),
		ExecutionOptimistic: isOptimistic,
		Data: &SignedBlock{
			Message:   jsonBytes,
			Signature: respBlk.Signature,
		},
	}, nil
}

func (*Server) getBlindedBlockBellatrixSSZ(_ context.Context, blk interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	blindedConsensusBlk, err := blk.PbBlindedBellatrixBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			consensusBlk, err := blk.PbBellatrixBlock()
			if err != nil {
				return nil, err
			}
			if consensusBlk == nil {
				return nil, errNilBlock
			}
			blkInterface, err := blk.ToBlinded()
			if err != nil {
				return nil, errors.Wrapf(err, "could not convert block to blinded block")
			}
			blindedConsensusBlk, err = blkInterface.PbBlindedBellatrixBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if blindedConsensusBlk == nil {
		return nil, errNilBlock
	}
	sszData, err := blindedConsensusBlk.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return sszData, nil
}

func (*Server) getBlindedBlockCapellaSSZ(_ context.Context, blk interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	blindedConsensusBlk, err := blk.PbBlindedCapellaBlock()
	if err != nil {
		// ErrUnsupportedField means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			consensusBlk, err := blk.PbCapellaBlock()
			if err != nil {
				return nil, err
			}
			if consensusBlk == nil {
				return nil, errNilBlock
			}
			blkInterface, err := blk.ToBlinded()
			if err != nil {
				return nil, errors.Wrapf(err, "could not convert block to blinded block")
			}
			blindedConsensusBlk, err = blkInterface.PbBlindedCapellaBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if blindedConsensusBlk == nil {
		return nil, errNilBlock
	}
	sszData, err := blindedConsensusBlk.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return sszData, nil
}

func (*Server) getBlindedBlockDenebSSZ(_ context.Context, blk interfaces.ReadOnlySignedBeaconBlock) ([]byte, error) {
	blindedConsensusBlk, err := blk.PbBlindedDenebBlock()
	if err != nil {
		// ErrUnsupportedGetter means that we have another block type
		if errors.Is(err, consensus_types.ErrUnsupportedField) {
			consensusBlk, err := blk.PbDenebBlock()
			if err != nil {
				return nil, err
			}
			if consensusBlk == nil {
				return nil, errNilBlock
			}
			blkInterface, err := blk.ToBlinded()
			if err != nil {
				return nil, errors.Wrapf(err, "could not convert block to blinded block")
			}
			blindedConsensusBlk, err = blkInterface.PbBlindedDenebBlock()
			if err != nil {
				return nil, errors.Wrapf(err, "could not get signed beacon block")
			}
		} else {
			return nil, err
		}
	}

	if blindedConsensusBlk == nil {
		return nil, errNilBlock
	}
	sszData, err := blindedConsensusBlk.MarshalSSZ()
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal block into SSZ")
	}
	return sszData, nil
}

// GetBlockAttestations retrieves attestation included in requested block.
func (s *Server) GetBlockAttestations(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetBlockAttestations")
	defer span.End()

	blockId := mux.Vars(r)["block_id"]
	if blockId == "" {
		http2.HandleError(w, "block_id is required in URL params", http.StatusBadRequest)
		return
	}
	blk, err := s.Blocker.Block(ctx, []byte(blockId))
	if !shared.WriteBlockFetchError(w, blk, err) {
		return
	}

	consensusAtts := blk.Block().Body().Attestations()
	atts := make([]*shared.Attestation, len(consensusAtts))
	for i, att := range consensusAtts {
		atts[i] = shared.AttestationFromConsensus(att)
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		http2.HandleError(w, "Could not get block root: "+err.Error(), http.StatusInternalServerError)
		return
	}
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, root)
	if err != nil {
		http2.HandleError(w, "Could not check if block is optimistic: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := &GetBlockAttestationsResponse{
		Data:                atts,
		ExecutionOptimistic: isOptimistic,
		Finalized:           s.FinalizationFetcher.IsFinalized(ctx, root),
	}
	http2.WriteJson(w, resp)
}

// PublishBlindedBlock instructs the beacon node to use the components of the `SignedBlindedBeaconBlock` to construct
// and publish a SignedBeaconBlock by swapping out the transactions_root for the corresponding full list of `transactions`.
// The beacon node should broadcast a newly constructed SignedBeaconBlock to the beacon network, to be included in the
// beacon chain. The beacon node is not required to validate the signed BeaconBlock, and a successful response (20X)
// only indicates that the broadcast has been successful. The beacon node is expected to integrate the new block into
// its state, and therefore validate the block internally, however blocks which fail the validation are still broadcast
// but a different status code is returned (202). Pre-Bellatrix, this endpoint will accept a SignedBeaconBlock. After
// Deneb, this additionally instructs the beacon node to broadcast all given signed blobs.
func (s *Server) PublishBlindedBlock(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.PublishBlindedBlock")
	defer span.End()
	if shared.IsSyncing(r.Context(), w, s.SyncChecker, s.HeadFetcher, s.TimeFetcher, s.OptimisticModeFetcher) {
		return
	}
	isSSZ := http2.SszRequested(r)
	if isSSZ {
		s.publishBlindedBlockSSZ(ctx, w, r)
	} else {
		s.publishBlindedBlock(ctx, w, r)
	}
}

// PublishBlindedBlockV2 instructs the beacon node to use the components of the `SignedBlindedBeaconBlock` to construct and publish a
// `SignedBeaconBlock` by swapping out the `transactions_root` for the corresponding full list of `transactions`.
// The beacon node should broadcast a newly constructed `SignedBeaconBlock` to the beacon network,
// to be included in the beacon chain. The beacon node is not required to validate the signed
// `BeaconBlock`, and a successful response (20X) only indicates that the broadcast has been
// successful. The beacon node is expected to integrate the new block into its state, and
// therefore validate the block internally, however blocks which fail the validation are still
// broadcast but a different status code is returned (202). Pre-Bellatrix, this endpoint will accept
// a `SignedBeaconBlock`. After Deneb, this additionally instructs the beacon node to broadcast all given signed blobs.
// The broadcast behaviour may be adjusted via the `broadcast_validation`
// query parameter.
func (s *Server) PublishBlindedBlockV2(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.PublishBlindedBlockV2")
	defer span.End()
	if shared.IsSyncing(r.Context(), w, s.SyncChecker, s.HeadFetcher, s.TimeFetcher, s.OptimisticModeFetcher) {
		return
	}
	isSSZ := http2.SszRequested(r)
	if isSSZ {
		s.publishBlindedBlockSSZ(ctx, w, r)
	} else {
		s.publishBlindedBlock(ctx, w, r)
	}
}

func (s *Server) publishBlindedBlockSSZ(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http2.HandleError(w, "Could not read request body: "+err.Error(), http.StatusInternalServerError)
		return
	}
	denebBlockContents := &eth.SignedBlindedBeaconBlockAndBlobsDeneb{}
	if err := denebBlockContents.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_BlindedDeneb{
				BlindedDeneb: denebBlockContents,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}
	capellaBlock := &eth.SignedBlindedBeaconBlockCapella{}
	if err := capellaBlock.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_BlindedCapella{
				BlindedCapella: capellaBlock,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}
	bellatrixBlock := &eth.SignedBlindedBeaconBlockBellatrix{}
	if err := bellatrixBlock.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_BlindedBellatrix{
				BlindedBellatrix: bellatrixBlock,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}

	// blinded is not supported before bellatrix hardfork
	altairBlock := &eth.SignedBeaconBlockAltair{}
	if err := altairBlock.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Altair{
				Altair: altairBlock,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}
	phase0Block := &eth.SignedBeaconBlock{}
	if err := phase0Block.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Phase0{
				Phase0: phase0Block,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}
	http2.HandleError(w, "Body does not represent a valid block type", http.StatusBadRequest)
}

func (s *Server) publishBlindedBlock(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http2.HandleError(w, "Could not read request body", http.StatusInternalServerError)
		return
	}
	versionHeader := r.Header.Get(api.VersionHeader)
	var blockVersionError string
	var denebBlockContents *shared.SignedBlindedBeaconBlockContentsDeneb
	if err = unmarshalStrict(body, &denebBlockContents); err == nil {
		consensusBlock, err := denebBlockContents.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Deneb) {
			blockVersionError = fmt.Sprintf("could not decode %s request body into consensus block: %v", version.String(version.Deneb), err.Error())
		}
	}

	var capellaBlock *shared.SignedBlindedBeaconBlockCapella
	if err = unmarshalStrict(body, &capellaBlock); err == nil {
		consensusBlock, err := capellaBlock.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Capella) {
			blockVersionError = fmt.Sprintf("could not decode %s request body into consensus block: %v", version.String(version.Capella), err.Error())
		}
	}

	var bellatrixBlock *shared.SignedBlindedBeaconBlockBellatrix
	if err = unmarshalStrict(body, &bellatrixBlock); err == nil {
		consensusBlock, err := bellatrixBlock.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Bellatrix) {
			blockVersionError = fmt.Sprintf("could not decode %s request body into consensus block: %v", version.String(version.Bellatrix), err.Error())
		}
	}
	var altairBlock *shared.SignedBeaconBlockAltair
	if err = unmarshalStrict(body, &altairBlock); err == nil {
		consensusBlock, err := altairBlock.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Altair) {
			blockVersionError = fmt.Sprintf("could not decode %s request body into consensus block: %v", version.String(version.Altair), err.Error())
		}
	}
	var phase0Block *shared.SignedBeaconBlock
	if err = unmarshalStrict(body, &phase0Block); err == nil {
		consensusBlock, err := phase0Block.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Phase0) {
			blockVersionError = fmt.Sprintf("could not decode %s request body into consensus block: %v", version.String(version.Phase0), err.Error())
		}
	}
	if versionHeader == "" {
		blockVersionError = fmt.Sprintf("please add the api header %s to see specific type errors", api.VersionHeader)
	}
	http2.HandleError(w, "Body does not represent a valid block type: "+blockVersionError, http.StatusBadRequest)
}

// PublishBlock instructs the beacon node to broadcast a newly signed beacon block to the beacon network,
// to be included in the beacon chain. A success response (20x) indicates that the block
// passed gossip validation and was successfully broadcast onto the network.
// The beacon node is also expected to integrate the block into state, but may broadcast it
// before doing so, so as to aid timely delivery of the block. Should the block fail full
// validation, a separate success response code (202) is used to indicate that the block was
// successfully broadcast but failed integration. After Deneb, this additionally instructs the
// beacon node to broadcast all given signed blobs.
func (s *Server) PublishBlock(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.PublishBlock")
	defer span.End()
	if shared.IsSyncing(r.Context(), w, s.SyncChecker, s.HeadFetcher, s.TimeFetcher, s.OptimisticModeFetcher) {
		return
	}
	isSSZ := http2.SszRequested(r)
	if isSSZ {
		s.publishBlockSSZ(ctx, w, r)
	} else {
		s.publishBlock(ctx, w, r)
	}
}

// PublishBlockV2 instructs the beacon node to broadcast a newly signed beacon block to the beacon network,
// to be included in the beacon chain. A success response (20x) indicates that the block
// passed gossip validation and was successfully broadcast onto the network.
// The beacon node is also expected to integrate the block into the state, but may broadcast it
// before doing so, so as to aid timely delivery of the block. Should the block fail full
// validation, a separate success response code (202) is used to indicate that the block was
// successfully broadcast but failed integration. After Deneb, this additionally instructs the beacon node to
// broadcast all given signed blobs. The broadcast behaviour may be adjusted via the
// `broadcast_validation` query parameter.
func (s *Server) PublishBlockV2(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.PublishBlockV2")
	defer span.End()
	if shared.IsSyncing(r.Context(), w, s.SyncChecker, s.HeadFetcher, s.TimeFetcher, s.OptimisticModeFetcher) {
		return
	}
	isSSZ := http2.SszRequested(r)
	if isSSZ {
		s.publishBlockSSZ(ctx, w, r)
	} else {
		s.publishBlock(ctx, w, r)
	}
}

func (s *Server) publishBlockSSZ(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http2.HandleError(w, "Could not read request body", http.StatusInternalServerError)
		return
	}
	denebBlockContents := &eth.SignedBeaconBlockAndBlobsDeneb{}
	if err := denebBlockContents.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Deneb{
				Deneb: denebBlockContents,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}
	capellaBlock := &eth.SignedBeaconBlockCapella{}
	if err := capellaBlock.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Capella{
				Capella: capellaBlock,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}
	bellatrixBlock := &eth.SignedBeaconBlockBellatrix{}
	if err := bellatrixBlock.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Bellatrix{
				Bellatrix: bellatrixBlock,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}
	altairBlock := &eth.SignedBeaconBlockAltair{}
	if err := altairBlock.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Altair{
				Altair: altairBlock,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}
	phase0Block := &eth.SignedBeaconBlock{}
	if err := phase0Block.UnmarshalSSZ(body); err == nil {
		genericBlock := &eth.GenericSignedBeaconBlock{
			Block: &eth.GenericSignedBeaconBlock_Phase0{
				Phase0: phase0Block,
			},
		}
		if err = s.validateBroadcast(ctx, r, genericBlock); err != nil {
			http2.HandleError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.proposeBlock(ctx, w, genericBlock)
		return
	}
	http2.HandleError(w, "Body does not represent a valid block type", http.StatusBadRequest)
}

func (s *Server) publishBlock(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http2.HandleError(w, "Could not read request body", http.StatusInternalServerError)
		return
	}
	versionHeader := r.Header.Get(api.VersionHeader)
	var blockVersionError string
	var denebBlockContents *shared.SignedBeaconBlockContentsDeneb
	if err = unmarshalStrict(body, &denebBlockContents); err == nil {
		consensusBlock, err := denebBlockContents.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Deneb) {
			blockVersionError = fmt.Sprintf(": could not decode %s request body into consensus block: %v", version.String(version.Deneb), err.Error())
		}
	}
	var capellaBlock *shared.SignedBeaconBlockCapella
	if err = unmarshalStrict(body, &capellaBlock); err == nil {
		consensusBlock, err := capellaBlock.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Capella) {
			blockVersionError = fmt.Sprintf(": could not decode %s request body into consensus block: %v", version.String(version.Capella), err.Error())
		}
	}
	var bellatrixBlock *shared.SignedBeaconBlockBellatrix
	if err = unmarshalStrict(body, &bellatrixBlock); err == nil {
		consensusBlock, err := bellatrixBlock.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Bellatrix) {
			blockVersionError = fmt.Sprintf(": could not decode %s request body into consensus block: %v", version.String(version.Bellatrix), err.Error())
		}
	}
	var altairBlock *shared.SignedBeaconBlockAltair
	if err = unmarshalStrict(body, &altairBlock); err == nil {
		consensusBlock, err := altairBlock.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Altair) {
			blockVersionError = fmt.Sprintf(": could not decode %s request body into consensus block: %v", version.String(version.Altair), err.Error())
		}
	}
	var phase0Block *shared.SignedBeaconBlock
	if err = unmarshalStrict(body, &phase0Block); err == nil {
		consensusBlock, err := phase0Block.ToGeneric()
		if err == nil {
			if err = s.validateBroadcast(ctx, r, consensusBlock); err != nil {
				http2.HandleError(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.proposeBlock(ctx, w, consensusBlock)
			return
		}
		if versionHeader == version.String(version.Phase0) {
			blockVersionError = fmt.Sprintf(": Could not decode %s request body into consensus block: %v", version.String(version.Phase0), err.Error())
		}
	}
	if versionHeader == "" {
		blockVersionError = fmt.Sprintf(": please add the api header %s to see specific type errors", api.VersionHeader)
	}
	http2.HandleError(w, "Body does not represent a valid block type"+blockVersionError, http.StatusBadRequest)
}

func (bs *Server) proposeBlock(ctx context.Context, w http.ResponseWriter, blk *eth.GenericSignedBeaconBlock) {
	_, err := bs.V1Alpha1ValidatorServer.ProposeBeaconBlock(ctx, blk)
	if err != nil {
		http2.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func unmarshalStrict(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func (s *Server) validateBroadcast(ctx context.Context, r *http.Request, blk *eth.GenericSignedBeaconBlock) error {
	switch r.URL.Query().Get(broadcastValidationQueryParam) {
	case broadcastValidationConsensus:
		b, err := blocks.NewSignedBeaconBlock(blk.Block)
		if err != nil {
			return errors.Wrapf(err, "could not create signed beacon block")
		}
		if err = s.validateConsensus(ctx, b); err != nil {
			return errors.Wrap(err, "consensus validation failed")
		}
	case broadcastValidationConsensusAndEquivocation:
		b, err := blocks.NewSignedBeaconBlock(blk.Block)
		if err != nil {
			return errors.Wrapf(err, "could not create signed beacon block")
		}
		if err = s.validateConsensus(r.Context(), b); err != nil {
			return errors.Wrap(err, "consensus validation failed")
		}
		if err = s.validateEquivocation(b.Block()); err != nil {
			return errors.Wrap(err, "equivocation validation failed")
		}
	default:
		return nil
	}
	return nil
}

func (s *Server) validateConsensus(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock) error {
	parentBlockRoot := blk.Block().ParentRoot()
	parentBlock, err := s.Blocker.Block(ctx, parentBlockRoot[:])
	if err != nil {
		return errors.Wrap(err, "could not get parent block")
	}
	parentStateRoot := parentBlock.Block().StateRoot()
	parentState, err := s.Stater.State(ctx, parentStateRoot[:])
	if err != nil {
		return errors.Wrap(err, "could not get parent state")
	}
	_, err = transition.ExecuteStateTransition(ctx, parentState, blk)
	if err != nil {
		return errors.Wrap(err, "could not execute state transition")
	}
	return nil
}

func (s *Server) validateEquivocation(blk interfaces.ReadOnlyBeaconBlock) error {
	if s.ForkchoiceFetcher.HighestReceivedBlockSlot() == blk.Slot() {
		return fmt.Errorf("block for slot %d already exists in fork choice", blk.Slot())
	}
	return nil
}

// GetBlockRoot retrieves the root of a block.
func (s *Server) GetBlockRoot(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetBlockRoot")
	defer span.End()

	var err error
	var root []byte
	blockID := mux.Vars(r)["block_id"]
	if blockID == "" {
		http2.HandleError(w, "block_id is required in URL params", http.StatusBadRequest)
		return
	}
	switch blockID {
	case "head":
		root, err = s.ChainInfoFetcher.HeadRoot(ctx)
		if err != nil {
			http2.HandleError(w, "Could not retrieve head root: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if root == nil {
			http2.HandleError(w, "No head root was found", http.StatusNotFound)
			return
		}
	case "finalized":
		finalized := s.ChainInfoFetcher.FinalizedCheckpt()
		root = finalized.Root
	case "genesis":
		blk, err := s.BeaconDB.GenesisBlock(ctx)
		if err != nil {
			http2.HandleError(w, "Could not retrieve genesis block: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := blocks.BeaconBlockIsNil(blk); err != nil {
			http2.HandleError(w, "Could not find genesis block: "+err.Error(), http.StatusNotFound)
			return
		}
		blkRoot, err := blk.Block().HashTreeRoot()
		if err != nil {
			http2.HandleError(w, "Could not hash genesis block: "+err.Error(), http.StatusInternalServerError)
			return
		}
		root = blkRoot[:]
	default:
		isHex := strings.HasPrefix(blockID, "0x")
		if isHex {
			blockIDBytes, err := hexutil.Decode(blockID)
			if err != nil {
				http2.HandleError(w, "Could not decode block ID into bytes: "+err.Error(), http.StatusBadRequest)
				return
			}
			if len(blockIDBytes) != fieldparams.RootLength {
				http2.HandleError(w, fmt.Sprintf("Block ID has length %d instead of %d", len(blockIDBytes), fieldparams.RootLength), http.StatusBadRequest)
				return
			}
			blockID32 := bytesutil.ToBytes32(blockIDBytes)
			blk, err := s.BeaconDB.Block(ctx, blockID32)
			if err != nil {
				http2.HandleError(w, fmt.Sprintf("Could not retrieve block for block root %#x: %v", blockID, err), http.StatusInternalServerError)
				return
			}
			if err := blocks.BeaconBlockIsNil(blk); err != nil {
				http2.HandleError(w, "Could not find block: "+err.Error(), http.StatusNotFound)
				return
			}
			root = blockIDBytes
		} else {
			slot, err := strconv.ParseUint(blockID, 10, 64)
			if err != nil {
				http2.HandleError(w, "Could not parse block ID: "+err.Error(), http.StatusBadRequest)
				return
			}
			hasRoots, roots, err := s.BeaconDB.BlockRootsBySlot(ctx, primitives.Slot(slot))
			if err != nil {
				http2.HandleError(w, fmt.Sprintf("Could not retrieve blocks for slot %d: %v", slot, err), http.StatusInternalServerError)
				return
			}

			if !hasRoots {
				http2.HandleError(w, "Could not find any blocks with given slot", http.StatusNotFound)
				return
			}
			root = roots[0][:]
			if len(roots) == 1 {
				break
			}
			for _, blockRoot := range roots {
				canonical, err := s.ChainInfoFetcher.IsCanonical(ctx, blockRoot)
				if err != nil {
					http2.HandleError(w, "Could not determine if block root is canonical: "+err.Error(), http.StatusInternalServerError)
					return
				}
				if canonical {
					root = blockRoot[:]
					break
				}
			}
		}
	}

	b32Root := bytesutil.ToBytes32(root)
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, b32Root)
	if err != nil {
		http2.HandleError(w, "Could not check if block is optimistic: "+err.Error(), http.StatusInternalServerError)
		return
	}
	response := &BlockRootResponse{
		Data: &struct {
			Root string `json:"root"`
		}{
			Root: hexutil.Encode(root),
		},
		ExecutionOptimistic: isOptimistic,
		Finalized:           s.FinalizationFetcher.IsFinalized(ctx, b32Root),
	}
	http2.WriteJson(w, response)
}

// GetStateFork returns Fork object for state with given 'stateId'.
func (s *Server) GetStateFork(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetStateFork")
	defer span.End()
	stateId := mux.Vars(r)["state_id"]
	if stateId == "" {
		http2.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}
	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}
	fork := st.Fork()
	isOptimistic, err := helpers.IsOptimistic(ctx, []byte(stateId), s.OptimisticModeFetcher, s.Stater, s.ChainInfoFetcher, s.BeaconDB)
	if err != nil {
		http2.HandleError(w, "Could not check optimistic status"+err.Error(), http.StatusInternalServerError)
		return
	}
	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		http2.HandleError(w, errors.Wrap(err, "Could not calculate root of latest block header: ").Error(), http.StatusInternalServerError)
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)
	response := &GetStateForkResponse{
		Data: &shared.Fork{
			PreviousVersion: hexutil.Encode(fork.PreviousVersion),
			CurrentVersion:  hexutil.Encode(fork.CurrentVersion),
			Epoch:           fmt.Sprintf("%d", fork.Epoch),
		},
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
	}
	http2.WriteJson(w, response)
}

// GetCommittees retrieves the committees for the given state at the given epoch.
// If the requested slot and index are defined, only those committees are returned.
func (s *Server) GetCommittees(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetCommittees")
	defer span.End()

	stateId := mux.Vars(r)["state_id"]
	if stateId == "" {
		http2.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}

	ok, rawEpoch, e := shared.UintFromQuery(w, r, "epoch")
	if !ok {
		return
	}
	ok, rawIndex, i := shared.UintFromQuery(w, r, "index")
	if !ok {
		return
	}
	ok, rawSlot, sl := shared.UintFromQuery(w, r, "slot")
	if !ok {
		return
	}

	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}

	epoch := slots.ToEpoch(st.Slot())
	if rawEpoch != "" {
		epoch = primitives.Epoch(e)
	}
	activeCount, err := corehelpers.ActiveValidatorCount(ctx, st, epoch)
	if err != nil {
		http2.HandleError(w, "Could not get active validator count: "+err.Error(), http.StatusInternalServerError)
		return
	}

	startSlot, err := slots.EpochStart(epoch)
	if err != nil {
		http2.HandleError(w, "Could not get epoch start slot: "+err.Error(), http.StatusInternalServerError)
		return
	}
	endSlot, err := slots.EpochEnd(epoch)
	if err != nil {
		http2.HandleError(w, "Could not get epoch end slot: "+err.Error(), http.StatusInternalServerError)
		return
	}
	committeesPerSlot := corehelpers.SlotCommitteeCount(activeCount)
	committees := make([]*shared.Committee, 0)
	for slot := startSlot; slot <= endSlot; slot++ {
		if rawSlot != "" && slot != primitives.Slot(sl) {
			continue
		}
		for index := primitives.CommitteeIndex(0); index < primitives.CommitteeIndex(committeesPerSlot); index++ {
			if rawIndex != "" && index != primitives.CommitteeIndex(i) {
				continue
			}
			committee, err := corehelpers.BeaconCommitteeFromState(ctx, st, slot, index)
			if err != nil {
				http2.HandleError(w, "Could not get committee: "+err.Error(), http.StatusInternalServerError)
				return
			}
			var validators []string
			for _, v := range committee {
				validators = append(validators, strconv.FormatUint(uint64(v), 10))
			}
			committeeContainer := &shared.Committee{
				Index:      strconv.FormatUint(uint64(index), 10),
				Slot:       strconv.FormatUint(uint64(slot), 10),
				Validators: validators,
			}
			committees = append(committees, committeeContainer)
		}
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, []byte(stateId), s.OptimisticModeFetcher, s.Stater, s.ChainInfoFetcher, s.BeaconDB)
	if err != nil {
		http2.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		http2.HandleError(w, "Could not calculate root of latest block header: "+err.Error(), http.StatusInternalServerError)
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)
	http2.WriteJson(w, &GetCommitteesResponse{Data: committees, ExecutionOptimistic: isOptimistic, Finalized: isFinalized})
}

// GetDepositContract retrieves deposit contract address and genesis fork version.
func (*Server) GetDepositContract(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "beacon.GetDepositContract")
	defer span.End()

	http2.WriteJson(w, &DepositContractResponse{
		Data: &struct {
			ChainId string `json:"chain_id"`
			Address string `json:"address"`
		}{
			ChainId: strconv.FormatUint(params.BeaconConfig().DepositChainID, 10),
			Address: params.BeaconConfig().DepositContractAddress,
		},
	})
}

// GetBlockHeaders retrieves block headers matching given query. By default it will fetch current head slot blocks.
func (s *Server) GetBlockHeaders(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetBlockHeaders")
	defer span.End()

	rawSlot := r.URL.Query().Get("slot")
	rawParentRoot := r.URL.Query().Get("parent_root")

	var err error
	var blks []interfaces.ReadOnlySignedBeaconBlock
	var blkRoots [][32]byte

	if rawParentRoot != "" {
		parentRoot, valid := shared.ValidateHex(w, "Parent Root", rawParentRoot, fieldparams.RootLength)
		if !valid {
			return
		}
		blks, blkRoots, err = s.BeaconDB.Blocks(ctx, filters.NewFilter().SetParentRoot(parentRoot))
		if err != nil {
			http2.HandleError(w, errors.Wrapf(err, "Could not retrieve blocks for parent root %s", parentRoot).Error(), http.StatusInternalServerError)
			return
		}
	} else {
		slot := uint64(s.ChainInfoFetcher.HeadSlot())
		if rawSlot != "" {
			var valid bool
			slot, valid = shared.ValidateUint(w, "Slot", rawSlot)
			if !valid {
				return
			}
		}
		blks, err = s.BeaconDB.BlocksBySlot(ctx, primitives.Slot(slot))
		if err != nil {
			http2.HandleError(w, errors.Wrapf(err, "Could not retrieve blocks for slot %d", slot).Error(), http.StatusInternalServerError)
			return
		}
		_, blkRoots, err = s.BeaconDB.BlockRootsBySlot(ctx, primitives.Slot(slot))
		if err != nil {
			http2.HandleError(w, errors.Wrapf(err, "Could not retrieve blocks for slot %d", slot).Error(), http.StatusInternalServerError)
			return
		}
	}

	isOptimistic := false
	isFinalized := true
	blkHdrs := make([]*shared.SignedBeaconBlockHeaderContainer, len(blks))
	for i, bl := range blks {
		v1alpha1Header, err := bl.Header()
		if err != nil {
			http2.HandleError(w, errors.Wrapf(err, "Could not get block header from block").Error(), http.StatusInternalServerError)
			return
		}
		headerRoot, err := v1alpha1Header.Header.HashTreeRoot()
		if err != nil {
			http2.HandleError(w, errors.Wrapf(err, "Could not hash block header").Error(), http.StatusInternalServerError)
			return
		}
		canonical, err := s.ChainInfoFetcher.IsCanonical(ctx, blkRoots[i])
		if err != nil {
			http2.HandleError(w, errors.Wrapf(err, "Could not determine if block root is canonical").Error(), http.StatusInternalServerError)
			return
		}
		if !isOptimistic {
			isOptimistic, err = s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, blkRoots[i])
			if err != nil {
				http2.HandleError(w, errors.Wrapf(err, "Could not check if block is optimistic").Error(), http.StatusInternalServerError)
				return
			}
		}
		if isFinalized {
			isFinalized = s.FinalizationFetcher.IsFinalized(ctx, blkRoots[i])
		}
		blkHdrs[i] = &shared.SignedBeaconBlockHeaderContainer{
			Header: &shared.SignedBeaconBlockHeader{
				Message:   shared.BeaconBlockHeaderFromConsensus(v1alpha1Header.Header),
				Signature: hexutil.Encode(v1alpha1Header.Signature),
			},
			Root:      hexutil.Encode(headerRoot[:]),
			Canonical: canonical,
		}
	}

	response := &GetBlockHeadersResponse{
		Data:                blkHdrs,
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
	}
	http2.WriteJson(w, response)
}

// GetBlockHeader retrieves block header for given block id.
func (s *Server) GetBlockHeader(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetBlockHeader")
	defer span.End()

	blockID := mux.Vars(r)["block_id"]
	if blockID == "" {
		http2.HandleError(w, "block_id is required in URL params", http.StatusBadRequest)
		return
	}

	blk, err := s.Blocker.Block(ctx, []byte(blockID))
	ok := shared.WriteBlockFetchError(w, blk, err)
	if !ok {
		return
	}
	blockHeader, err := blk.Header()
	if err != nil {
		http2.HandleError(w, "Could not get block header: %s"+err.Error(), http.StatusInternalServerError)
		return
	}
	headerRoot, err := blockHeader.Header.HashTreeRoot()
	if err != nil {
		http2.HandleError(w, "Could not hash block header: %s"+err.Error(), http.StatusInternalServerError)
		return
	}
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		http2.HandleError(w, "Could not hash block: %s"+err.Error(), http.StatusInternalServerError)
		return
	}
	canonical, err := s.ChainInfoFetcher.IsCanonical(ctx, blkRoot)
	if err != nil {
		http2.HandleError(w, "Could not determine if block root is canonical: %s"+err.Error(), http.StatusInternalServerError)
		return
	}
	isOptimistic, err := s.OptimisticModeFetcher.IsOptimisticForRoot(ctx, blkRoot)
	if err != nil {
		http2.HandleError(w, "Could not check if block is optimistic: %s"+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := &GetBlockHeaderResponse{
		Data: &shared.SignedBeaconBlockHeaderContainer{
			Root:      hexutil.Encode(headerRoot[:]),
			Canonical: canonical,
			Header: &shared.SignedBeaconBlockHeader{
				Message:   shared.BeaconBlockHeaderFromConsensus(blockHeader.Header),
				Signature: hexutil.Encode(blockHeader.Signature),
			},
		},
		ExecutionOptimistic: isOptimistic,
		Finalized:           s.FinalizationFetcher.IsFinalized(ctx, blkRoot),
	}
	http2.WriteJson(w, resp)
}

// GetFinalityCheckpoints returns finality checkpoints for state with given 'stateId'. In case finality is
// not yet achieved, checkpoint should return epoch 0 and ZERO_HASH as root.
func (s *Server) GetFinalityCheckpoints(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetFinalityCheckpoints")
	defer span.End()

	stateId := mux.Vars(r)["state_id"]
	if stateId == "" {
		http2.HandleError(w, "state_id is required in URL params", http.StatusBadRequest)
		return
	}

	st, err := s.Stater.State(ctx, []byte(stateId))
	if err != nil {
		shared.WriteStateFetchError(w, err)
		return
	}
	isOptimistic, err := helpers.IsOptimistic(ctx, []byte(stateId), s.OptimisticModeFetcher, s.Stater, s.ChainInfoFetcher, s.BeaconDB)
	if err != nil {
		http2.HandleError(w, "Could not check optimistic status: "+err.Error(), http.StatusInternalServerError)
		return
	}
	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		http2.HandleError(w, "Could not calculate root of latest block header: "+err.Error(), http.StatusInternalServerError)
		return
	}
	isFinalized := s.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	pj := st.PreviousJustifiedCheckpoint()
	cj := st.CurrentJustifiedCheckpoint()
	f := st.FinalizedCheckpoint()
	resp := &GetFinalityCheckpointsResponse{
		Data: &FinalityCheckpoints{
			PreviousJustified: &shared.Checkpoint{
				Epoch: strconv.FormatUint(uint64(pj.Epoch), 10),
				Root:  hexutil.Encode(pj.Root),
			},
			CurrentJustified: &shared.Checkpoint{
				Epoch: strconv.FormatUint(uint64(cj.Epoch), 10),
				Root:  hexutil.Encode(cj.Root),
			},
			Finalized: &shared.Checkpoint{
				Epoch: strconv.FormatUint(uint64(f.Epoch), 10),
				Root:  hexutil.Encode(f.Root),
			},
		},
		ExecutionOptimistic: isOptimistic,
		Finalized:           isFinalized,
	}
	http2.WriteJson(w, resp)
}

// GetGenesis retrieves details of the chain's genesis which can be used to identify chain.
func (s *Server) GetGenesis(w http.ResponseWriter, r *http.Request) {
	_, span := trace.StartSpan(r.Context(), "beacon.GetGenesis")
	defer span.End()

	genesisTime := s.GenesisTimeFetcher.GenesisTime()
	if genesisTime.IsZero() {
		http2.HandleError(w, "Chain genesis info is not yet known", http.StatusNotFound)
		return
	}
	validatorsRoot := s.ChainInfoFetcher.GenesisValidatorsRoot()
	if bytes.Equal(validatorsRoot[:], params.BeaconConfig().ZeroHash[:]) {
		http2.HandleError(w, "Chain genesis info is not yet known", http.StatusNotFound)
		return
	}
	forkVersion := params.BeaconConfig().GenesisForkVersion

	resp := &GetGenesisResponse{
		Data: &Genesis{
			GenesisTime:           strconv.FormatUint(uint64(genesisTime.Unix()), 10),
			GenesisValidatorsRoot: hexutil.Encode(validatorsRoot[:]),
			GenesisForkVersion:    hexutil.Encode(forkVersion),
		},
	}
	http2.WriteJson(w, resp)
}
