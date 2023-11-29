package blob

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/lookup"
	field_params "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

// Blobs is an HTTP handler for Beacon API getBlobs.
func (s *Server) Blobs(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.Blobs")
	defer span.End()

	var sidecars []*eth.BlobSidecar
	var root []byte

	indices := parseIndices(r.URL)
	segments := strings.Split(r.URL.Path, "/")
	blockId := segments[len(segments)-1]
	switch blockId {
	case "genesis":
		http2.HandleError(w, "blobs are not supported for Phase 0 fork", http.StatusBadRequest)
		return
	case "head":
		var err error
		root, err = s.ChainInfoFetcher.HeadRoot(r.Context())
		if err != nil {
			http2.HandleError(w, errors.Wrapf(err, "could not retrieve head root").Error(), http.StatusInternalServerError)
			return
		}
	case "finalized":
		fcp := s.ChainInfoFetcher.FinalizedCheckpt()
		if fcp == nil {
			http2.HandleError(w, "received nil finalized checkpoint", http.StatusInternalServerError)
			return
		}
		root = fcp.Root
	case "justified":
		jcp := s.ChainInfoFetcher.CurrentJustifiedCheckpt()
		if jcp == nil {
			http2.HandleError(w, "received nil justified checkpoint", http.StatusInternalServerError)
			return
		}
		root = jcp.Root
	default:
		if bytesutil.IsHex([]byte(blockId)) {
			var err error
			root, err = hexutil.Decode(blockId)
			if err != nil {
				http2.HandleError(w, errors.Wrap(err, "could not decode block ID into hex").Error(), http.StatusInternalServerError)
				return
			}
		} else {
			slot, err := strconv.ParseUint(blockId, 10, 64)
			if err != nil {
				http2.HandleError(w, lookup.NewBlockIdParseError(err).Error(), http.StatusBadRequest)
				return
			}
			denebStart, err := slots.EpochStart(params.BeaconConfig().DenebForkEpoch)
			if err != nil {
				http2.HandleError(w, errors.Wrap(err, "could not calculate Deneb start slot").Error(), http.StatusInternalServerError)
				return
			}
			if primitives.Slot(slot) < denebStart {
				http2.HandleError(w, "blobs are not supported before Deneb fork", http.StatusBadRequest)
				return
			}
			ok, roots, err := s.BeaconDB.BlockRootsBySlot(ctx, primitives.Slot(slot))
			if !ok {
				http2.HandleError(w, fmt.Sprintf("no block roots at slot %d", slot), http.StatusBadRequest)
				return
			}
			if err != nil {
				http2.HandleError(w, errors.Wrap(err, "failed to get block roots by slot").Error(), http.StatusInternalServerError)
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

	for _, index := range indices {
		sidecar, err := s.BlobStorage.Get(bytesutil.ToBytes32(root), index)
		if err != nil {
			http2.HandleError(w, errors.Wrapf(err, "could not retrieve blobs for root %#x", root).Error(), http.StatusInternalServerError)
			return
		}
		sidecars = append(sidecars, sidecar.ROBlob.BlobSidecar)
	}

	ssz := http2.SszRequested(r)
	if ssz {
		sidecarResp := &eth.BlobSidecars{
			Sidecars: sidecars,
		}
		sszResp, err := sidecarResp.MarshalSSZ()
		if err != nil {
			http2.HandleError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http2.WriteSsz(w, sszResp, "blob_sidecars.ssz")
		return
	}

	http2.WriteJson(w, buildSidecarsResponse(sidecars))
}

// parseIndices filters out invalid and duplicate blob indices
func parseIndices(url *url.URL) []uint64 {
	rawIndices := url.Query()["indices"]
	indices := make([]uint64, 0, field_params.MaxBlobsPerBlock)
loop:
	for _, raw := range rawIndices {
		ix, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			continue
		}
		if ix >= field_params.MaxBlobsPerBlock {
			continue
		}
		for i := range indices {
			if ix == indices[i] {
				continue loop
			}
		}
		indices = append(indices, ix)
	}
	return indices
}

func buildSidecarsResponse(sidecars []*eth.BlobSidecar) *SidecarsResponse {
	resp := &SidecarsResponse{Data: make([]*Sidecar, len(sidecars))}
	for i, sc := range sidecars {
		proofs := make([]string, len(sc.CommitmentInclusionProof))
		for j := range sc.CommitmentInclusionProof {
			proofs[j] = hexutil.Encode(sc.CommitmentInclusionProof[j])
		}
		resp.Data[i] = &Sidecar{
			Index:                    strconv.FormatUint(sc.Index, 10),
			Blob:                     hexutil.Encode(sc.Blob),
			KZGCommitment:            hexutil.Encode(sc.KzgCommitment),
			SignedBeaconBlockHeader:  shared.SignedBeaconBlockHeaderFromConsensus(sc.SignedBlockHeader),
			KZGProof:                 hexutil.Encode(sc.KzgProof),
			CommitmentInclusionProof: proofs,
		}
	}
	return resp
}
