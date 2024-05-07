package validator

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v5/api/client/builder"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	"github.com/prysmaticlabs/prysm/v5/math"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/v5/network/forks"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	builderValueGweiGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "builder_value_gwei",
		Help: "Builder payload value in gwei",
	})
	localValueGweiGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "local_value_gwei",
		Help: "Local payload value in gwei",
	})
	builderGetPayloadMissCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "builder_get_payload_miss_count",
		Help: "The number of get payload misses for validator requests to builder",
	})
)

// emptyTransactionsRoot represents the returned value of ssz.TransactionsRoot([][]byte{}) and
// can be used as a constant to avoid recomputing this value in every call.
var emptyTransactionsRoot = [32]byte{127, 254, 36, 30, 166, 1, 135, 253, 176, 24, 123, 250, 34, 222, 53, 209, 249, 190, 215, 171, 6, 29, 148, 1, 253, 71, 227, 74, 84, 251, 237, 225}

// blockBuilderTimeout is the maximum amount of time allowed for a block builder to respond to a
// block request. This value is known as `BUILDER_PROPOSAL_DELAY_TOLERANCE` in builder spec.
const blockBuilderTimeout = 1 * time.Second

// Sets the execution data for the block. Execution data can come from local EL client or remote builder depends on validator registration and circuit breaker conditions.
func choosePayload(ctx context.Context, resp *proposalResponseConstructor, builderBoostFactor uint64) (*PayloadOption, error) {
	ctx, span := trace.StartSpan(ctx, "validator.choosePayload")
	defer span.End()
	blk := resp.block
	slot := blk.Block().Slot()
	if slots.ToEpoch(slot) < params.BeaconConfig().BellatrixForkEpoch {
		return nil, nil
	}

	if resp.local.IsNil() {
		return nil, errors.New("local payload is nil")
	}

	// Use local payload if builder payload is nil.
	if resp.builder.IsNil() {
		return resp.local, nil
	}

	switch {
	case blk.Version() >= version.Capella:
		// Compare payload values between local and builder. Default to the local value if it is higher.
		localValueGwei := uint64(resp.local.ValueInGwei())
		builderValueGwei := uint64(resp.builder.ValueInGwei())
		if builderValueGwei == 0 {
			log.WithField("builderGwei", 0).Warn("Proposer: failed to get builder payload value") // Default to local if can't get builder value.
			return resp.local, nil
		}

		withdrawalsMatched, err := matchingWithdrawalsRoot(resp.local.ExecutionData, resp.local.ExecutionData)
		if err != nil {
			tracing.AnnotateError(span, err)
			log.WithError(err).Warn("Proposer: failed to match withdrawals root")
			return resp.local, nil
		}

		// Use builder payload if the following in true:
		// builder_bid_value * builderBoostFactor(default 100) > local_block_value * (local-block-value-boost + 100)
		boost := params.BeaconConfig().LocalBlockValueBoost
		higherValueBuilder := builderValueGwei*builderBoostFactor > localValueGwei*(100+boost)
		if boost > 0 && builderBoostFactor != defaultBuilderBoostFactor {
			log.WithFields(logrus.Fields{
				"localGweiValue":       localValueGwei,
				"localBoostPercentage": boost,
				"builderGweiValue":     builderValueGwei,
				"builderBoostFactor":   builderBoostFactor,
			}).Warn("Proposer: both local boost and builder boost are using non default values")
		}
		builderValueGweiGauge.Set(float64(builderValueGwei))
		localValueGweiGauge.Set(float64(localValueGwei))

		// If we can't get the builder value, just use local block.
		if higherValueBuilder && withdrawalsMatched { // Builder value is higher and withdrawals match.
			return resp.builder, nil
		}
		if !higherValueBuilder {
			log.WithFields(logrus.Fields{
				"localGweiValue":       localValueGwei,
				"localBoostPercentage": boost,
				"builderGweiValue":     builderValueGwei,
				"builderBoostFactor":   builderBoostFactor,
			}).Warn("Proposer: using local execution payload because higher value")
		}
		span.AddAttributes(
			trace.BoolAttribute("higherValueBuilder", higherValueBuilder),
			trace.Int64Attribute("localGweiValue", int64(localValueGwei)),         // lint:ignore uintcast -- This is OK for tracing.
			trace.Int64Attribute("localBoostPercentage", int64(boost)),            // lint:ignore uintcast -- This is OK for tracing.
			trace.Int64Attribute("builderGweiValue", int64(builderValueGwei)),     // lint:ignore uintcast -- This is OK for tracing.
			trace.Int64Attribute("builderBoostFactor", int64(builderBoostFactor)), // lint:ignore uintcast -- This is OK for tracing.
		)
		return resp.local, nil
	default: // Bellatrix case.
		return resp.builder, nil
	}
}

// This function retrieves the payload header and kzg commitments given the slot number and the validator index.
// It's a no-op if the latest head block is not versioned bellatrix.
func (vs *Server) setBuilderResponseHeader(ctx context.Context, resp *proposalResponseConstructor, slot primitives.Slot, idx primitives.ValidatorIndex) error {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.setBuilderResponseHeader")
	defer span.End()

	if slots.ToEpoch(slot) < params.BeaconConfig().BellatrixForkEpoch {
		return errors.New("can't get payload header from builder before bellatrix epoch")
	}

	b, err := vs.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return err
	}

	h, err := b.Block().Body().Execution()
	if err != nil {
		return errors.Wrap(err, "failed to get execution header")
	}
	pk, err := vs.HeadFetcher.HeadValidatorIndexToPublicKey(ctx, idx)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, blockBuilderTimeout)
	defer cancel()

	signedBid, err := vs.BlockBuilder.GetHeader(ctx, slot, bytesutil.ToBytes32(h.BlockHash()), pk)
	if err != nil {
		return err
	}
	if signedBid.IsNil() {
		return errors.New("builder returned nil bid")
	}
	fork, err := forks.Fork(slots.ToEpoch(slot))
	if err != nil {
		return errors.Wrap(err, "unable to get fork information")
	}
	forkName, ok := params.BeaconConfig().ForkVersionNames[bytesutil.ToBytes4(fork.CurrentVersion)]
	if !ok {
		return errors.New("unable to find current fork in schedule")
	}
	if !strings.EqualFold(version.String(signedBid.Version()), forkName) {
		return fmt.Errorf("builder bid response version: %d is different from head block version: %d for epoch %d", signedBid.Version(), b.Version(), slots.ToEpoch(slot))
	}

	bid, err := signedBid.Message()
	if err != nil {
		return errors.Wrap(err, "could not get bid")
	}
	if bid.IsNil() {
		return errors.New("builder returned nil bid")
	}

	v := bytesutil.LittleEndianBytesToBigInt(bid.Value())
	if v.String() == "0" {
		return errors.New("builder returned header with 0 bid amount")
	}

	header, err := bid.Header()
	if err != nil {
		return errors.Wrap(err, "could not get bid header")
	}
	bidWei := math.BigEndianBytesToWei(bid.Value())
	txRoot, err := header.TransactionsRoot()
	if err != nil {
		return errors.Wrap(err, "could not get transaction root")
	}
	if bytesutil.ToBytes32(txRoot) == emptyTransactionsRoot {
		return errors.New("builder returned header with an empty tx root")
	}

	if !bytes.Equal(header.ParentHash(), h.BlockHash()) {
		return fmt.Errorf("incorrect parent hash %#x != %#x", header.ParentHash(), h.BlockHash())
	}

	t, err := slots.ToTime(uint64(vs.TimeFetcher.GenesisTime().Unix()), slot)
	if err != nil {
		return err
	}
	if header.Timestamp() != uint64(t.Unix()) {
		return fmt.Errorf("incorrect timestamp %d != %d", header.Timestamp(), uint64(t.Unix()))
	}

	if err := validateBuilderSignature(signedBid); err != nil {
		return errors.Wrap(err, "could not validate builder signature")
	}

	var kzgCommitments [][]byte
	if bid.Version() >= version.Deneb {
		kzgCommitments, err = bid.BlobKzgCommitments()
		if err != nil {
			return errors.Wrap(err, "could not get blob kzg commitments")
		}
		if len(kzgCommitments) > fieldparams.MaxBlobsPerBlock {
			return fmt.Errorf("builder returned too many kzg commitments: %d", len(kzgCommitments))
		}
		for _, c := range kzgCommitments {
			if len(c) != fieldparams.BLSPubkeyLength {
				return fmt.Errorf("builder returned invalid kzg commitment length: %d", len(c))
			}
		}
	}

	l := log.WithFields(logrus.Fields{
		"gweiValue":          math.WeiToGwei(bidWei),
		"builderPubKey":      fmt.Sprintf("%#x", bid.Pubkey()),
		"blockHash":          fmt.Sprintf("%#x", header.BlockHash()),
		"slot":               slot,
		"validator":          idx,
		"sinceSlotStartTime": time.Since(t),
	})
	if len(kzgCommitments) > 0 {
		l = l.WithField("kzgCommitmentCount", len(kzgCommitments))
	}
	l.Info("Received header with bid")

	span.AddAttributes(
		trace.StringAttribute("value", math.WeiToBigInt(bidWei).String()),
		trace.StringAttribute("builderPubKey", fmt.Sprintf("%#x", bid.Pubkey())),
		trace.StringAttribute("blockHash", fmt.Sprintf("%#x", header.BlockHash())),
	)

	pwb, err := NewPayloadOption(header, bidWei, nil, kzgCommitments)
	resp.builder = pwb
	return err
}

// Validates builder signature and returns an error if the signature is invalid.
func validateBuilderSignature(signedBid builder.SignedBid) error {
	d, err := signing.ComputeDomain(params.BeaconConfig().DomainApplicationBuilder,
		nil, /* fork version */
		nil /* genesis val root */)
	if err != nil {
		return err
	}
	if signedBid.IsNil() {
		return errors.New("nil builder bid")
	}
	bid, err := signedBid.Message()
	if err != nil {
		return errors.Wrap(err, "could not get bid")
	}
	if bid.IsNil() {
		return errors.New("builder returned nil bid")
	}
	return signing.VerifySigningRoot(bid, bid.Pubkey(), signedBid.Signature(), d)
}

func matchingWithdrawalsRoot(local, builder interfaces.ExecutionData) (bool, error) {
	wds, err := local.Withdrawals()
	if err != nil {
		return false, errors.Wrap(err, "could not get local withdrawals")
	}
	br, err := builder.WithdrawalsRoot()
	if err != nil {
		return false, errors.Wrap(err, "could not get builder withdrawals root")
	}
	wr, err := ssz.WithdrawalSliceRoot(wds, fieldparams.MaxWithdrawalsPerPayload)
	if err != nil {
		return false, errors.Wrap(err, "could not compute local withdrawals root")
	}

	if !bytes.Equal(br, wr[:]) {
		log.WithFields(logrus.Fields{
			"local":   fmt.Sprintf("%#x", wr),
			"builder": fmt.Sprintf("%#x", br),
		}).Warn("Proposer: withdrawal roots don't match, using local block")
		return false, nil
	}
	return true, nil
}
