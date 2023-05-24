package validator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	payloadattribute "github.com/prysmaticlabs/prysm/v4/consensus-types/payload-attribute"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v4/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
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
func (vs *Server) getExecutionPayload(ctx context.Context, slot primitives.Slot, vIdx primitives.ValidatorIndex, headRoot [32]byte, st state.BeaconState) (interfaces.ExecutionData, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.getExecutionPayload")
	defer span.End()

	proposerID, payloadId, ok := vs.ProposerSlotIndexCache.GetProposerPayloadIDs(slot, headRoot)
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

	if ok && proposerID == vIdx && payloadId != [8]byte{} { // Payload ID is cache hit. Return the cached payload ID.
		var pid [8]byte
		copy(pid[:], payloadId[:])
		payloadIDCacheHit.Inc()
		payload, err := vs.ExecutionEngineCaller.GetPayload(ctx, pid, slot)
		switch {
		case err == nil:
			warnIfFeeRecipientDiffers(payload, feeRecipient)
			return payload, nil
		case errors.Is(err, context.DeadlineExceeded):
		default:
			return nil, errors.Wrap(err, "could not get cached payload from execution client")
		}
	}
	payloadIDCacheMiss.Inc()

	finalizedBlockHash := [32]byte{}
	justifiedBlockHash := [32]byte{}
	// Blocks before Bellatrix don't have execution payloads. Use zeros as the hash.
	if st.Version() >= version.Altair {
		finalizedBlockHash = vs.FinalizationFetcher.FinalizedBlockHash()
		justifiedBlockHash = vs.FinalizationFetcher.UnrealizedJustifiedPayloadBlockHash()
	}
	header, err := st.LatestExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	f := &enginev1.ForkchoiceState{
		HeadBlockHash:      header.BlockHash(),
		SafeBlockHash:      justifiedBlockHash[:],
		FinalizedBlockHash: finalizedBlockHash[:],
	}
	t, err := slots.ToTime(st.GenesisTime(), slot)
	if err != nil {
		return nil, err
	}
	random, err := helpers.RandaoMix(st, time.CurrentEpoch(st))
	if err != nil {
		return nil, err
	}
	var attr payloadattribute.Attributer
	switch st.Version() {
	case version.Capella:
		withdrawals, err := st.ExpectedWithdrawals()
		if err != nil {
			return nil, err
		}
		attr, err = payloadattribute.New(&enginev1.PayloadAttributesV2{
			Timestamp:             uint64(t.Unix()),
			PrevRandao:            random,
			SuggestedFeeRecipient: feeRecipient.Bytes(),
			Withdrawals:           withdrawals,
		})
		if err != nil {
			return nil, err
		}
	case version.Bellatrix:
		attr, err = payloadattribute.New(&enginev1.PayloadAttributes{
			Timestamp:             uint64(t.Unix()),
			PrevRandao:            random,
			SuggestedFeeRecipient: feeRecipient.Bytes(),
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown beacon state version")
	}

	payloadID, _, err := vs.ExecutionEngineCaller.ForkchoiceUpdated(ctx, f, attr)
	if err != nil {
		return nil, errors.Wrap(err, "could not prepare payload")
	}
	if payloadID == nil {
		return nil, fmt.Errorf("nil payload with block hash: %#x", header.BlockHash())
	}
	payload, err := vs.ExecutionEngineCaller.GetPayload(ctx, *payloadID, slot)
	if err != nil {
		return nil, err
	}
	warnIfFeeRecipientDiffers(payload, feeRecipient)
	return payload, nil
}

// warnIfFeeRecipientDiffers logs a warning if the fee recipient in the included payload does not
// match the requested one.
func warnIfFeeRecipientDiffers(payload interfaces.ExecutionData, feeRecipient common.Address) {
	// Warn if the fee recipient is not the value we expect.
	if payload != nil && !bytes.Equal(payload.FeeRecipient(), feeRecipient[:]) {
		logrus.WithFields(logrus.Fields{
			"wantedFeeRecipient": fmt.Sprintf("%#x", feeRecipient),
			"received":           fmt.Sprintf("%#x", payload.FeeRecipient()),
		}).Warn("Fee recipient address from execution client is not what was expected. " +
			"It is possible someone has compromised your client to try and take your transaction fees")
	}
}
