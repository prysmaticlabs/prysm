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
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/math"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/v5/network/forks"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// emptyTransactionsRoot represents the returned value of ssz.TransactionsRoot([][]byte{}) and
// can be used as a constant to avoid recomputing this value in every call.
var emptyTransactionsRoot = [32]byte{127, 254, 36, 30, 166, 1, 135, 253, 176, 24, 123, 250, 34, 222, 53, 209, 249, 190, 215, 171, 6, 29, 148, 1, 253, 71, 227, 74, 84, 251, 237, 225}

// blockBuilderTimeout is the maximum amount of time allowed for a block builder to respond to a
// block request. This value is known as `BUILDER_PROPOSAL_DELAY_TOLERANCE` in builder spec.
const blockBuilderTimeout = 1 * time.Second

// proposalBlock accumulates data needed to respond to the proposer GetBeaconBlock request.
type proposalResponseConstructor struct {
	head            state.BeaconState
	block           interfaces.SignedBeaconBlock
	overrideBuilder bool
	FeeRecipient    primitives.ExecutionAddress
	local           *PayloadPossibility
	builder         *PayloadPossibility
	winner          *PayloadPossibility
}

func newProposalResponseConstructor(blk interfaces.SignedBeaconBlock, st state.BeaconState, feeRecipient primitives.ExecutionAddress) *proposalResponseConstructor {
	return &proposalResponseConstructor{block: blk, head: st, FeeRecipient: feeRecipient}
}

var errNoProposalSource = errors.New("proposal process did not pick between builder and local block")

// construct picks the best proposal and sets the execution header/payload attributes for it on the block.
// It also takes the parent state as an argument so that it can compute and set the state root using the
// completely updated block.
func (pc *proposalResponseConstructor) construct(ctx context.Context, builderBoostFactor uint64) (*ethpb.GenericBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "proposalResponseConstructor.construct")
	defer span.End()
	if err := blocks.HasNilErr(pc.block); err != nil {
		return nil, err
	}
	best, err := pc.choosePayload(ctx, builderBoostFactor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error constructing execution payload for block: %v", err)
	}
	best, err = pc.complete(ctx, best)
	if err != nil {
		return nil, err
	}
	return constructGenericBeaconBlock(pc.block, best.bid, best.bundle)
}

func (pc *proposalResponseConstructor) complete(ctx context.Context, best *PayloadPossibility) (*PayloadPossibility, error) {
	err := pc.completeWithBest(ctx, best)
	if err == nil {
		return best, nil
	}
	// We can fall back from the builder to local, but not the other way. If local fails, we're done.
	if best == pc.local {
		return nil, err
	}

	// Try again with the local payload. If this fails then we're truly done.
	return pc.local, pc.completeWithBest(ctx, pc.local)
}

func (pc *proposalResponseConstructor) completeWithBest(ctx context.Context, best *PayloadPossibility) error {
	ctx, span := trace.StartSpan(ctx, "proposalResponseConstructor.completeWithBest")
	defer span.End()
	if best.IsNil() {
		return errNoProposalSource
	}

	if err := pc.block.SetExecution(best.ExecutionData); err != nil {
		return err
	}
	if pc.block.Version() >= version.Deneb {
		kzgc := best.kzgCommitments
		if best.bundle != nil {
			kzgc = best.bundle.KzgCommitments
		}
		if err := pc.block.SetBlobKzgCommitments(kzgc); err != nil {
			return err
		}
	}

	root, err := transition.CalculateStateRoot(ctx, pc.head, pc.block)
	if err != nil {
		return errors.Wrapf(err, "could not calculate state root for proposal with parent root=%#x at slot %d", pc.block.Block().ParentRoot(), pc.head.Slot())
	}
	log.WithField("beaconStateRoot", fmt.Sprintf("%#x", root)).Debugf("Computed state root")
	pc.block.SetStateRoot(root[:])

	return nil
}

// constructGenericBeaconBlock constructs a `GenericBeaconBlock` based on the block version and other parameters.
func constructGenericBeaconBlock(blk interfaces.SignedBeaconBlock, bid math.Wei, bundle *enginev1.BlobsBundle) (*ethpb.GenericBeaconBlock, error) {
	if err := blocks.HasNilErr(blk); err != nil {
		return nil, err
	}
	blockProto, err := blk.Block().Proto()
	if err != nil {
		return nil, err
	}
	payloadValue := math.WeiToBigInt(bid).String()

	switch pb := blockProto.(type) {
	case *ethpb.BeaconBlockDeneb:
		denebContents := &ethpb.BeaconBlockContentsDeneb{Block: pb}
		if bundle != nil {
			denebContents.KzgProofs = bundle.Proofs
			denebContents.Blobs = bundle.Blobs
		}
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Deneb{Deneb: denebContents}, IsBlinded: false, PayloadValue: payloadValue}, nil
	case *ethpb.BlindedBeaconBlockDeneb:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedDeneb{BlindedDeneb: pb}, IsBlinded: true, PayloadValue: payloadValue}, nil
	case *ethpb.BeaconBlockCapella:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Capella{Capella: pb}, IsBlinded: false, PayloadValue: payloadValue}, nil
	case *ethpb.BlindedBeaconBlockCapella:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedCapella{BlindedCapella: pb}, IsBlinded: true, PayloadValue: payloadValue}, nil
	case *ethpb.BeaconBlockBellatrix:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Bellatrix{Bellatrix: pb}, IsBlinded: false, PayloadValue: payloadValue}, nil
	case *ethpb.BlindedBeaconBlockBellatrix:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: pb}, IsBlinded: true, PayloadValue: payloadValue}, nil
	case *ethpb.BeaconBlockAltair:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Altair{Altair: pb}}, nil
	case *ethpb.BeaconBlock:
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Phase0{Phase0: pb}}, nil
	}
	return nil, fmt.Errorf("unknown .block version: %d", blk.Version())
}

// PayloadPossibility represents one of the payload possibilities that the proposer may select between (local, builder).
type PayloadPossibility struct {
	interfaces.ExecutionData
	bid            math.Wei
	bundle         *enginev1.BlobsBundle
	kzgCommitments [][]byte
}

// NewPayloadPossibility initializes a PayloadPossibility. This should only be used to represent payloads that have a bid,
// otherwise directly use an ExecutionData type.
func NewPayloadPossibility(p interfaces.ExecutionData, bid math.Wei, bundle *enginev1.BlobsBundle, kzgc [][]byte) (*PayloadPossibility, error) {
	if err := blocks.HasNilErr(p); err != nil {
		return nil, err
	}
	if bid == nil {
		bid = math.ZeroWei
	}
	return &PayloadPossibility{ExecutionData: p, bid: bid, bundle: bundle, kzgCommitments: kzgc}, nil
}

func (p *PayloadPossibility) IsNil() bool {
	return p == nil || p.ExecutionData.IsNil()
}

// ValueInGwei is a helper to converts the bid value to its gwei representation.
func (p *PayloadPossibility) ValueInGwei() math.Gwei {
	return math.WeiToGwei(p.bid)
}
func (resp *proposalResponseConstructor) buildBlockParallel(ctx context.Context, vs *Server, skipMevBoost bool, builderBoostFactor uint64) (*ethpb.GenericBeaconBlock, error) {
	sBlk := resp.block
	// Build consensus fields in background
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		// Set eth1 data.
		eth1Data, err := vs.eth1DataMajorityVote(ctx, resp.head)
		if err != nil {
			eth1Data = &ethpb.Eth1Data{DepositRoot: params.BeaconConfig().ZeroHash[:], BlockHash: params.BeaconConfig().ZeroHash[:]}
			log.WithError(err).Error("Could not get eth1data")
		}
		sBlk.SetEth1Data(eth1Data)

		// Set deposit and attestation.
		deposits, atts, err := vs.packDepositsAndAttestations(ctx, resp.head, eth1Data) // TODO: split attestations and deposits
		if err != nil {
			sBlk.SetDeposits([]*ethpb.Deposit{})
			if err := sBlk.SetAttestations([]interfaces.Attestation{}); err != nil {
				log.WithError(err).Error("Could not set attestations on block")
			}
			log.WithError(err).Error("Could not pack deposits and attestations")
		} else {
			sBlk.SetDeposits(deposits)
			if err := sBlk.SetAttestations(atts); err != nil {
				log.WithError(err).Error("Could not set attestations on block")
			}
		}

		// Set slashings.
		validProposerSlashings, validAttSlashings := vs.getSlashings(ctx, resp.head)
		sBlk.SetProposerSlashings(validProposerSlashings)
		if err := sBlk.SetAttesterSlashings(validAttSlashings); err != nil {
			log.WithError(err).Error("Could not set attester slashings on block")
		}

		// Set exits.
		sBlk.SetVoluntaryExits(vs.getExits(resp.head, sBlk.Block().Slot()))

		// Set sync aggregate. New in Altair.
		vs.setSyncAggregate(ctx, sBlk)

		// Set bls to execution change. New in Capella.
		vs.setBlsToExecData(sBlk, resp.head)
		return nil
	})

	builderCtx, cancelBuilder := context.WithCancel(ctx)
	eg.Go(func() error {
		if err := resp.populateLocalPossibility(ctx, vs, cancelBuilder); err != nil {
			if !errors.Is(err, errActivationNotReached) && !errors.Is(err, errNoTerminalBlockHash) {
				return status.Errorf(codes.Internal, "Could not get local payload: %v", err)
			}
		}
		return nil
	})

	if !skipMevBoost {
		eg.Go(func() error {
			// builderCtx will be canceled by populateLocalPossibility if the engine decides to override the builder.
			if err := resp.populateBuilderPossibility(builderCtx, vs); err != nil {
				builderGetPayloadMissCount.Inc()
				log.WithError(err).Error("Could not get builder payload")
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return resp.construct(ctx, builderBoostFactor)
}

// populateLocalPossibility queries the execution layer engine API to retrieve the local payload for the proposal.
func (resp *proposalResponseConstructor) populateLocalPossibility(ctx context.Context, vs *Server, cancelBuilder context.CancelFunc) error {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.setLocalPayloadResp")
	defer span.End()

	blk := resp.block.Block()
	if blk.Version() < version.Bellatrix {
		return nil
	}

	pid, err := vs.getPayloadID(ctx, resp.block, resp.head, resp.FeeRecipient)
	if err != nil {
		return errors.Wrap(err, "unable to determine payload id for proposal request")
	}
	payload, bid, bundle, overrideBuilder, err := vs.ExecutionEngineCaller.GetPayload(ctx, pid, blk.Slot())
	if err != nil {
		return err
	}
	resp.overrideBuilder = overrideBuilder
	if resp.overrideBuilder {
		cancelBuilder()
	}
	warnIfFeeRecipientDiffers(payload, resp.FeeRecipient)
	pwb, err := NewPayloadPossibility(payload, bid, bundle, nil)
	if err != nil {
		return err
	}
	resp.local = pwb

	log.WithField("value", math.WeiToGwei(bid)).Debug("received execution payload from local engine")
	return nil
}

func (resp *proposalResponseConstructor) populateBuilderPossibility(ctx context.Context, vs *Server) error {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.populateBuilderPossibility")
	defer span.End()

	blk := resp.block.Block()
	if blk.Version() < version.Bellatrix {
		return nil
	}
	slot := blk.Slot()
	vIdx := blk.ProposerIndex()
	canUseBuilder, err := vs.canUseBuilder(ctx, slot, vIdx)
	if err != nil {
		return errors.Wrap(err, "failed to check if we can use the builder")
	}
	span.AddAttributes(trace.BoolAttribute("canUseBuilder", canUseBuilder))
	if !canUseBuilder {
		return nil
	}

	b, err := vs.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return err
	}

	h, err := b.Block().Body().Execution()
	if err != nil {
		return errors.Wrap(err, "failed to get execution header")
	}
	pk, err := vs.HeadFetcher.HeadValidatorIndexToPublicKey(ctx, vIdx)
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
		"validator":          vIdx,
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

	pwb, err := NewPayloadPossibility(header, bidWei, nil, kzgCommitments)
	resp.builder = pwb
	return err
}

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

// Sets the execution data for the block. Execution data can come from local EL client or remote builder depends on validator registration and circuit breaker conditions.
func (resp *proposalResponseConstructor) choosePayload(ctx context.Context, builderBoostFactor uint64) (*PayloadPossibility, error) {
	_, span := trace.StartSpan(ctx, "validator.choosePayload")
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
