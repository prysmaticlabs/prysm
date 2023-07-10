package blob

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	field_params "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// Blobs is an HTTP handler for Beacon API getBlobs.
func (s *Server) Blobs(w http.ResponseWriter, r *http.Request) {
	var sidecars []*eth.BlobSidecar
	var root []byte

	indices := parseIndices(r.URL)
	segments := strings.Split(r.URL.Path, "/")
	blockId := segments[len(segments)-1]
	switch blockId {
	case "genesis":
		errJson := &http2.DefaultErrorJson{
			Message: "blobs are not supported for Phase 0 fork",
			Code:    http.StatusBadRequest,
		}
		http2.WriteError(w, errJson)
		return
	case "head":
		var err error
		root, err = s.ChainInfoFetcher.HeadRoot(r.Context())
		if err != nil {
			errJson := &http2.DefaultErrorJson{
				Message: errors.Wrapf(err, "could not retrieve head root").Error(),
				Code:    http.StatusInternalServerError,
			}
			http2.WriteError(w, errJson)
			return
		}
	case "finalized":
		fcp := s.ChainInfoFetcher.FinalizedCheckpt()
		if fcp == nil {
			errJson := &http2.DefaultErrorJson{
				Message: "received nil finalized checkpoint",
				Code:    http.StatusInternalServerError,
			}
			http2.WriteError(w, errJson)
			return
		}
		root = fcp.Root
	case "justified":
		jcp := s.ChainInfoFetcher.CurrentJustifiedCheckpt()
		if jcp == nil {
			errJson := &http2.DefaultErrorJson{
				Message: "received nil justified checkpoint",
				Code:    http.StatusInternalServerError,
			}
			http2.WriteError(w, errJson)
			return
		}
		root = jcp.Root
	default:
		if bytesutil.IsHex([]byte(blockId)) {
			var err error
			root, err = hexutil.Decode(blockId)
			if err != nil {
				errJson := &http2.DefaultErrorJson{
					Message: errors.Wrap(err, "could not decode block ID into hex").Error(),
					Code:    http.StatusInternalServerError,
				}
				http2.WriteError(w, errJson)
				return
			}
		} else {
			slot, err := strconv.ParseUint(blockId, 10, 64)
			if err != nil {
				errJson := &http2.DefaultErrorJson{
					Message: lookup.NewBlockIdParseError(err).Error(),
					Code:    http.StatusBadRequest,
				}
				http2.WriteError(w, errJson)
				return
			}
			denebStart, err := slots.EpochStart(params.BeaconConfig().DenebForkEpoch)
			if err != nil {
				errJson := &http2.DefaultErrorJson{
					Message: errors.Wrap(err, "could not calculate Deneb start slot").Error(),
					Code:    http.StatusInternalServerError,
				}
				http2.WriteError(w, errJson)
				return
			}
			if primitives.Slot(slot) < denebStart {
				errJson := &http2.DefaultErrorJson{
					Message: "blobs are not supported before Deneb fork",
					Code:    http.StatusBadRequest,
				}
				http2.WriteError(w, errJson)
				return
			}
			sidecars, err = s.BeaconDB.BlobSidecarsBySlot(r.Context(), primitives.Slot(slot), indices...)
			if err != nil {
				errJson := &http2.DefaultErrorJson{
					Message: errors.Wrapf(err, "could not retrieve blobs for slot %d", slot).Error(),
					Code:    http.StatusInternalServerError,
				}
				http2.WriteError(w, errJson)
				return
			}
			http2.WriteJson(w, buildSidecardsResponse(sidecars))
			return
		}
	}

	var err error
	sidecars, err = s.BeaconDB.BlobSidecarsByRoot(r.Context(), bytesutil.ToBytes32(root), indices...)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: errors.Wrapf(err, "could not retrieve blobs for root %#x", root).Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	ssz, err := http2.SszRequested(r)
	if err != nil {
		errJson := &http2.DefaultErrorJson{
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		}
		http2.WriteError(w, errJson)
		return
	}

	if ssz {
		v2sidecars, err := migration.V1Alpha1BlobSidecarsToV2(sidecars)
		if err != nil {
			errJson := &http2.DefaultErrorJson{
				Message: err.Error(),
				Code:    http.StatusInternalServerError,
			}
			http2.WriteError(w, errJson)
			return
		}
		sidecarResp := &ethpb.BlobSidecars{
			Sidecars: v2sidecars,
		}
		sszResp, err := sidecarResp.MarshalSSZ()
		if err != nil {
			errJson := &http2.DefaultErrorJson{
				Message: err.Error(),
				Code:    http.StatusInternalServerError,
			}
			http2.WriteError(w, errJson)
			return
		}
		http2.WriteSsz(w, sszResp, "blob_sidecars.ssz")
		return
	}

	http2.WriteJson(w, buildSidecardsResponse(sidecars))
}

// parseIndices filters out invalid and duplicate blob indices
func parseIndices(url *url.URL) []uint64 {
	query := url.Query()
	helpers.NormalizeQueryValues(query)
	rawIndices := query["indices"]
	indices := make([]uint64, 0, field_params.MaxBlobsPerBlock)
loop:
	for _, raw := range rawIndices {
		ix, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			continue
		}
		for i := range indices {
			if ix == indices[i] || ix >= field_params.MaxBlobsPerBlock {
				continue loop
			}
		}
		indices = append(indices, ix)
	}
	return indices
}

func buildSidecardsResponse(sidecars []*eth.BlobSidecar) *SidecarsResponse {
	resp := &SidecarsResponse{Data: make([]*Sidecar, len(sidecars))}
	for i, sc := range sidecars {
		resp.Data[i] = &Sidecar{
			BlockRoot:       hexutil.Encode(sc.BlockRoot),
			Index:           strconv.FormatUint(sc.Index, 10),
			Slot:            strconv.FormatUint(uint64(sc.Slot), 10),
			BlockParentRoot: hexutil.Encode(sc.BlockParentRoot),
			ProposerIndex:   strconv.FormatUint(uint64(sc.ProposerIndex), 10),
			Blob:            hexutil.Encode(sc.Blob),
			KZGCommitment:   hexutil.Encode(sc.KzgCommitment),
			KZGProof:        hexutil.Encode(sc.KzgProof),
		}
	}
	return resp
}
