package validator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	prysmtime "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	payloadattribute "github.com/prysmaticlabs/prysm/v5/consensus-types/payload-attribute"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
)

var (
	// payloadIDCacheMiss tracks the number of payload ID requests that aren't present in the cache.
	payloadIDCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payload_id_cache_miss",
		Help: "The number of payload id get requests that aren't present in the cache.",
	})
	// payloadIDCacheHit tracks the number of payload ID requests that are present in the cache.
	payloadIDCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payload_id_cache_hit",
		Help: "The number of payload id get requests that are present in the cache.",
	})
)

func (vs *Server) getPayloadID(ctx context.Context, blk interfaces.SignedBeaconBlock, st state.BeaconState, feeRecipient primitives.ExecutionAddress) (primitives.PayloadID, error) {
	var pid primitives.PayloadID
	slot := blk.Block().Slot()
	head := blk.Block().ParentRoot()
	payloadId, ok := vs.PayloadIDCache.PayloadID(slot, head)
	if ok && payloadId != [8]byte{} {
		payloadIDCacheHit.Inc()
		// Payload ID is cache hit. Return the cached payload ID.
		copy(pid[:], payloadId[:])
		return pid, nil
	}
	payloadIDCacheMiss.Inc()
	log.WithFields(logrus.Fields{
		"validatorIndex": blk.Block().ProposerIndex(),
		"slot":           slot,
		"headRoot":       fmt.Sprintf("%#x", head),
	}).Debug("payload ID cache miss")

	parentHash, err := vs.getParentBlockHash(ctx, st, slot)
	if err != nil {
		return pid, err
	}
	finalizedBlockHash := [32]byte{}
	justifiedBlockHash := [32]byte{}
	// Blocks before Bellatrix don't have execution payloads. Use zeros as the hash.
	if st.Version() >= version.Bellatrix {
		finalizedBlockHash = vs.FinalizationFetcher.FinalizedBlockHash()
		justifiedBlockHash = vs.FinalizationFetcher.UnrealizedJustifiedPayloadBlockHash()
	}
	f := &enginev1.ForkchoiceState{
		HeadBlockHash:      parentHash,
		SafeBlockHash:      justifiedBlockHash[:],
		FinalizedBlockHash: finalizedBlockHash[:],
	}

	attr, err := payloadAttributesForState(slot, head, st, feeRecipient)
	if err != nil {
		return pid, err
	}
	payloadID, _, err := vs.ExecutionEngineCaller.ForkchoiceUpdated(ctx, f, attr)
	if err != nil {
		return pid, errors.Wrap(err, "could not completeWithBest payload")
	}
	if payloadID == nil {
		return pid, fmt.Errorf("nil payload with block hash: %#x", parentHash)
	}
	copy(pid[:], payloadID[:])
	return pid, nil
}

func payloadAttributesForState(slot primitives.Slot, pr [32]byte, st state.BeaconState, feeRecipient primitives.ExecutionAddress) (payloadattribute.Attributer, error) {
	random, err := helpers.RandaoMix(st, prysmtime.CurrentEpoch(st))
	if err != nil {
		return nil, err
	}

	t, err := slots.ToTime(st.GenesisTime(), slot)
	if err != nil {
		return nil, err
	}
	switch st.Version() {
	case version.Deneb:
		withdrawals, err := st.ExpectedWithdrawals()
		if err != nil {
			return nil, err
		}
		return payloadattribute.New(&enginev1.PayloadAttributesV3{
			Timestamp:             uint64(t.Unix()),
			PrevRandao:            random,
			SuggestedFeeRecipient: feeRecipient[:],
			Withdrawals:           withdrawals,
			ParentBeaconBlockRoot: pr[:],
		})
	case version.Capella:
		withdrawals, err := st.ExpectedWithdrawals()
		if err != nil {
			return nil, err
		}
		return payloadattribute.New(&enginev1.PayloadAttributesV2{
			Timestamp:             uint64(t.Unix()),
			PrevRandao:            random,
			SuggestedFeeRecipient: feeRecipient[:],
			Withdrawals:           withdrawals,
		})
	case version.Bellatrix:
		return payloadattribute.New(&enginev1.PayloadAttributes{
			Timestamp:             uint64(t.Unix()),
			PrevRandao:            random,
			SuggestedFeeRecipient: feeRecipient[:],
		})
	default:
		return nil, errors.New("unknown beacon state version")
	}
}

// warnIfFeeRecipientDiffers logs a warning if the fee recipient in the included payload does not
// match the requested one.
func warnIfFeeRecipientDiffers(payload interfaces.ExecutionData, feeRecipient primitives.ExecutionAddress) {
	// Warn if the fee recipient is not the value we expect.
	if payload != nil && !bytes.Equal(payload.FeeRecipient(), feeRecipient[:]) {
		logrus.WithFields(logrus.Fields{
			"wantedFeeRecipient": fmt.Sprintf("%#x", feeRecipient),
			"received":           fmt.Sprintf("%#x", payload.FeeRecipient()),
		}).Warn("Fee recipient address from execution client is not what was expected. " +
			"It is possible someone has compromised your client to try and take your transaction fees")
	}
}

// This returns the valid terminal block hash with an existence bool value.
//
// Spec code:
// def get_terminal_pow_block(pow_chain: Dict[Hash32, PowBlock]) -> Optional[PowBlock]:
//
//	if TERMINAL_BLOCK_HASH != Hash32():
//	    # Terminal block hash override takes precedence over terminal total difficulty
//	    if TERMINAL_BLOCK_HASH in pow_chain:
//	        return pow_chain[TERMINAL_BLOCK_HASH]
//	    else:
//	        return None
//
//	return get_pow_block_at_terminal_total_difficulty(pow_chain)
func (vs *Server) getTerminalBlockHashIfExists(ctx context.Context, transitionTime uint64) ([]byte, bool, error) {
	terminalBlockHash := params.BeaconConfig().TerminalBlockHash
	// Terminal block hash override takes precedence over terminal total difficulty.
	if params.BeaconConfig().TerminalBlockHash != params.BeaconConfig().ZeroHash {
		exists, _, err := vs.Eth1BlockFetcher.BlockExists(ctx, terminalBlockHash)
		if err != nil {
			return nil, false, err
		}
		if !exists {
			return nil, false, nil
		}

		return terminalBlockHash.Bytes(), true, nil
	}

	return vs.ExecutionEngineCaller.GetTerminalBlockHash(ctx, transitionTime)
}

var errActivationNotReached = errors.New("activation epoch not reached")
var errNoTerminalBlockHash = errors.New("no terminal block hash")

// getParentBlockHash retrieves the parent block hash of the block at the given slot.
// The function's behavior varies depending on the state version and whether the merge has been completed.
//
// For states of version Capella or later, the block hash is directly retrieved from the state's latest execution payload header.
//
// If the merge transition has been completed, the parent block hash is also retrieved from the state's latest execution payload header.
//
// If the activation epoch has not been reached, an errActivationNotReached error is returned.
//
// Otherwise, the terminal block hash is fetched based on the slot's time, and an error is returned if it doesn't exist.
func (vs *Server) getParentBlockHash(ctx context.Context, st state.BeaconState, slot primitives.Slot) ([]byte, error) {
	if st.Version() >= version.Capella {
		return getParentBlockHashPostCapella(st)
	}

	mergeComplete, err := blocks.IsMergeTransitionComplete(st)
	if err != nil {
		return nil, err
	}
	if mergeComplete {
		return getParentBlockHashPostMerge(st)
	}

	if activationEpochNotReached(slot) {
		return nil, errActivationNotReached
	}

	return getParentBlockHashPreMerge(ctx, vs, st, slot)
}

// getParentBlockHashPostCapella retrieves the parent block hash for states of version Capella or later.
func getParentBlockHashPostCapella(st state.BeaconState) ([]byte, error) {
	header, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, errors.Wrap(err, "could not get post capella payload header")
	}
	return header.BlockHash(), nil
}

// getParentBlockHashPostMerge retrieves the parent block hash after the merge has completed.
func getParentBlockHashPostMerge(st state.BeaconState) ([]byte, error) {
	header, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, errors.Wrap(err, "could not get post merge payload header")
	}
	return header.BlockHash(), nil
}

// getParentBlockHashPreMerge retrieves the parent block hash before the merge has completed.
func getParentBlockHashPreMerge(ctx context.Context, vs *Server, st state.BeaconState, slot primitives.Slot) ([]byte, error) {
	t, err := slots.ToTime(st.GenesisTime(), slot)
	if err != nil {
		return nil, err
	}

	parentHash, hasTerminalBlock, err := vs.getTerminalBlockHashIfExists(ctx, uint64(t.Unix()))
	if err != nil {
		return nil, err
	}
	if !hasTerminalBlock {
		return nil, errNoTerminalBlockHash
	}
	return parentHash, nil
}

// activationEpochNotReached returns true if activation epoch has not been reach.
// Which satisfy the following conditions in spec:
//
//	  is_terminal_block_hash_set = TERMINAL_BLOCK_HASH != Hash32()
//	  is_activation_epoch_reached = get_current_epoch(state) >= TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH
//	  if is_terminal_block_hash_set and not is_activation_epoch_reached:
//		return True
func activationEpochNotReached(slot primitives.Slot) bool {
	terminalBlockHashSet := bytesutil.ToBytes32(params.BeaconConfig().TerminalBlockHash.Bytes()) != [32]byte{}
	if terminalBlockHashSet {
		return params.BeaconConfig().TerminalBlockHashActivationEpoch > slots.ToEpoch(slot)
	}
	return false
}

func emptyPayload() *enginev1.ExecutionPayload {
	return &enginev1.ExecutionPayload{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		ExtraData:     make([]byte, 0),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
	}
}

func emptyPayloadCapella() *enginev1.ExecutionPayloadCapella {
	return &enginev1.ExecutionPayloadCapella{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		ExtraData:     make([]byte, 0),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
		Withdrawals:   make([]*enginev1.Withdrawal, 0),
	}
}

func emptyPayloadDeneb() *enginev1.ExecutionPayloadDeneb {
	return &enginev1.ExecutionPayloadDeneb{
		ParentHash:    make([]byte, fieldparams.RootLength),
		FeeRecipient:  make([]byte, fieldparams.FeeRecipientLength),
		StateRoot:     make([]byte, fieldparams.RootLength),
		ReceiptsRoot:  make([]byte, fieldparams.RootLength),
		LogsBloom:     make([]byte, fieldparams.LogsBloomLength),
		PrevRandao:    make([]byte, fieldparams.RootLength),
		ExtraData:     make([]byte, 0),
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
		Transactions:  make([][]byte, 0),
		Withdrawals:   make([]*enginev1.Withdrawal, 0),
	}
}
