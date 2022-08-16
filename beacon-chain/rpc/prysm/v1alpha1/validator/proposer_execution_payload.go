package validator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/kv"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
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

// This returns the execution payload of a given slot. The function has full awareness of pre and post merge.
// The payload is computed given the respected time of merge.
func (vs *Server) getExecutionPayload(ctx context.Context, slot types.Slot, vIdx types.ValidatorIndex, headRoot [32]byte) (*enginev1.ExecutionPayload, error) {
	proposerID, payloadId, ok := vs.ProposerSlotIndexCache.GetProposerPayloadIDs(slot, headRoot)
	if ok && proposerID == vIdx && payloadId != [8]byte{} { // Payload ID is cache hit. Return the cached payload ID.
		var pid [8]byte
		copy(pid[:], payloadId[:])
		payloadIDCacheHit.Inc()
		return vs.ExecutionEngineCaller.GetPayload(ctx, pid)
	}

	st, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, err
	}
	st, err = transition.ProcessSlotsIfPossible(ctx, st, slot)
	if err != nil {
		return nil, err
	}

	var parentHash []byte
	var hasTerminalBlock bool
	mergeComplete, err := blocks.IsMergeTransitionComplete(st)
	if err != nil {
		return nil, err
	}

	t, err := slots.ToTime(st.GenesisTime(), slot)
	if err != nil {
		return nil, err
	}
	if mergeComplete {
		header, err := st.LatestExecutionPayloadHeader()
		if err != nil {
			return nil, err
		}
		parentHash = header.BlockHash
	} else {
		if activationEpochNotReached(slot) {
			return emptyPayload(), nil
		}
		parentHash, hasTerminalBlock, err = vs.getTerminalBlockHashIfExists(ctx, uint64(t.Unix()))
		if err != nil {
			return nil, err
		}
		if !hasTerminalBlock {
			return emptyPayload(), nil
		}
	}
	payloadIDCacheMiss.Inc()

	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return nil, err
	}
	finalizedBlockHash := params.BeaconConfig().ZeroHash[:]
	finalizedRoot := bytesutil.ToBytes32(st.FinalizedCheckpoint().Root)
	if finalizedRoot != [32]byte{} { // finalized root could be zeros before the first finalized block.
		finalizedBlock, err := vs.BeaconDB.Block(ctx, bytesutil.ToBytes32(st.FinalizedCheckpoint().Root))
		if err != nil {
			return nil, err
		}
		if err := consensusblocks.BeaconBlockIsNil(finalizedBlock); err != nil {
			return nil, err
		}
		switch finalizedBlock.Version() {
		case version.Phase0, version.Altair: // Blocks before Bellatrix don't have execution payloads. Use zeros as the hash.
		default:
			finalizedPayload, err := finalizedBlock.Block().Body().Execution()
			if err != nil {
				return nil, err
			}
			finalizedBlockHash = finalizedPayload.BlockHash()
		}
	}

	f := &enginev1.ForkchoiceState{
		HeadBlockHash:      parentHash,
		SafeBlockHash:      parentHash,
		FinalizedBlockHash: finalizedBlockHash,
	}

	feeRecipient := params.BeaconConfig().DefaultFeeRecipient
	recipient, err := vs.BeaconDB.FeeRecipientByValidatorID(ctx, vIdx)
	switch err == nil {
	case true:
		feeRecipient = recipient
	case errors.As(err, kv.ErrNotFoundFeeRecipient):
		// If fee recipient is not found in DB and not set from beacon node CLI,
		// use the burn address.
		if feeRecipient.String() == params.BeaconConfig().EthBurnAddressHex {
			logrus.WithFields(logrus.Fields{
				"validatorIndex": vIdx,
				"burnAddress":    params.BeaconConfig().EthBurnAddressHex,
			}).Warn("Fee recipient is currently using the burn address, " +
				"you will not be rewarded transaction fees on this setting. " +
				"Please set a different eth address as the fee recipient. " +
				"Please refer to our documentation for instructions")
		}
	default:
		return nil, errors.Wrap(err, "could not get fee recipient in db")
	}
	p := &enginev1.PayloadAttributes{
		Timestamp:             uint64(t.Unix()),
		PrevRandao:            random,
		SuggestedFeeRecipient: feeRecipient.Bytes(),
	}
	payloadID, _, err := vs.ExecutionEngineCaller.ForkchoiceUpdated(ctx, f, p)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare payload")
	}
	if payloadID == nil {
		return nil, fmt.Errorf("nil payload with block hash: %#x", parentHash)
	}
	payload, err := vs.ExecutionEngineCaller.GetPayload(ctx, *payloadID)
	if err != nil {
		return nil, err
	}
	// Warn if the fee recipient is not the value we expect.
	if payload != nil && !bytes.Equal(payload.FeeRecipient, feeRecipient[:]) {
		logrus.WithFields(logrus.Fields{
			"wantedFeeRecipient": fmt.Sprintf("%#x", feeRecipient),
			"received":           fmt.Sprintf("%#x", payload.FeeRecipient),
		}).Warn("Fee recipient address from execution client is not what was expected. " +
			"It is possible someone has compromised your client to try and take your transaction fees")
	}
	return payload, nil
}

// This returns the valid terminal block hash with an existence bool value.
//
// Spec code:
// def get_terminal_pow_block(pow_chain: Dict[Hash32, PowBlock]) -> Optional[PowBlock]:
//    if TERMINAL_BLOCK_HASH != Hash32():
//        # Terminal block hash override takes precedence over terminal total difficulty
//        if TERMINAL_BLOCK_HASH in pow_chain:
//            return pow_chain[TERMINAL_BLOCK_HASH]
//        else:
//            return None
//
//    return get_pow_block_at_terminal_total_difficulty(pow_chain)
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

// activationEpochNotReached returns true if activation epoch has not been reach.
// Which satisfy the following conditions in spec:
//        is_terminal_block_hash_set = TERMINAL_BLOCK_HASH != Hash32()
//        is_activation_epoch_reached = get_current_epoch(state) >= TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH
//        if is_terminal_block_hash_set and not is_activation_epoch_reached:
//      	return True
func activationEpochNotReached(slot types.Slot) bool {
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
		BaseFeePerGas: make([]byte, fieldparams.RootLength),
		BlockHash:     make([]byte, fieldparams.RootLength),
	}
}
