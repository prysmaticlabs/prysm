package beacon

import (
	"fmt"
	"net/http"

	"github.com/OffchainLabs/prysm/v7/api/server/structs"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/rpc/eth/shared"
	"github.com/OffchainLabs/prysm/v7/monitoring/tracing/trace"
	"github.com/OffchainLabs/prysm/v7/network/httputil"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// GetDecoupledBlockSummary serves a block's post-state FG values. Loaded by root, a reorg can't mix blocks.
func (s *Server) GetDecoupledBlockSummary(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "beacon.GetDecoupledBlockSummary")
	defer span.End()

	blockId := r.PathValue("block_id")
	if blockId == "" {
		httputil.HandleError(w, "block_id is required in URL params", http.StatusBadRequest)
		return
	}
	blk, err := s.Blocker.Block(ctx, []byte(blockId))
	if !shared.WriteBlockFetchError(w, blk, err) {
		return
	}
	if blk.Version() < version.Decoupled {
		httputil.HandleError(w, "Block is not a decoupled block", http.StatusBadRequest)
		return
	}
	root, err := blk.Block().HashTreeRoot()
	if err != nil {
		httputil.HandleError(w, "Could not get block root: "+err.Error(), http.StatusInternalServerError)
		return
	}
	st, err := s.StateGen.StateByRoot(ctx, root)
	if err != nil || st == nil || st.IsNil() {
		httputil.HandleError(w, "State for block is unavailable", http.StatusNotFound)
		return
	}
	height, err := st.Height()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	entry, err := st.Entry()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	targetParticipation, err := st.TargetParticipation()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	progress, err := st.Progress()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	justifiedHeight, err := st.JustifiedHeight()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	justified, err := st.Justified()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	finalizedHeight, err := st.FinalizedHeight()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	finalized, err := st.Finalized()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pending, err := st.PendingFinality()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	entryOut := structs.CheckpointDecoupledFromConsensus(entry)
	// Zero entry root means this block, it is filled next slot.
	deferred := entryOut != nil && entryOut.Root == hexutil.Encode(make([]byte, 32))
	if deferred {
		entryOut.Root = hexutil.Encode(root[:])
	}
	rows := make([]*structs.DecoupledPendingFinalitySummary, len(pending))
	for i, p := range pending {
		rows[i] = &structs.DecoupledPendingFinalitySummary{
			Height:        fmt.Sprintf("%d", p.Height),
			Target:        structs.CheckpointDecoupledFromConsensus(p.Target),
			Participation: fmt.Sprintf("%d", p.Participation.Count()),
		}
	}
	httputil.WriteJson(w, &structs.GetDecoupledBlockSummaryResponse{Data: &structs.DecoupledBlockSummary{
		Slot:                fmt.Sprintf("%d", blk.Block().Slot()),
		Root:                hexutil.Encode(root[:]),
		Height:              fmt.Sprintf("%d", height),
		Entry:               entryOut,
		EntryDeferred:       deferred,
		TargetParticipation: fmt.Sprintf("%d", targetParticipation.Count()),
		Progress:            fmt.Sprintf("%d", progress.Count()),
		ValidatorCount:      fmt.Sprintf("%d", targetParticipation.Len()),
		JustifiedHeight:     fmt.Sprintf("%d", justifiedHeight),
		Justified:           structs.CheckpointDecoupledFromConsensus(justified),
		FinalizedHeight:     fmt.Sprintf("%d", finalizedHeight),
		Finalized:           structs.CheckpointDecoupledFromConsensus(finalized),
		PendingFinality:     rows,
	}})
}
