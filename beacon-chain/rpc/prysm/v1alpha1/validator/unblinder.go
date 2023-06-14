package validator

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/builder"
	consensus_types "github.com/prysmaticlabs/prysm/v4/consensus-types"
	consensusblocks "github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type unblinder struct {
	b       interfaces.SignedBeaconBlock
	builder builder.BlockBuilder
}

func newUnblinder(b interfaces.SignedBeaconBlock, builder builder.BlockBuilder) (*unblinder, error) {
	if err := consensusblocks.BeaconBlockIsNil(b); err != nil {
		return nil, err
	}
	if builder == nil {
		return nil, errors.New("nil builder provided")
	}
	return &unblinder{
		b:       b,
		builder: builder,
	}, nil
}

func (u *unblinder) unblindBuilderBlock(ctx context.Context) (interfaces.SignedBeaconBlock, error) {
	if !u.b.IsBlinded() || u.b.Version() < version.Bellatrix {
		return u.b, nil
	}
	if u.b.IsBlinded() && !u.builder.Configured() {
		return nil, errors.New("builder not configured")
	}

	psb, err := u.blindedProtoBlock()
	if err != nil {
		return nil, errors.Wrap(err, "could not get blinded proto block")
	}
	sb, err := consensusblocks.NewSignedBeaconBlock(psb)
	if err != nil {
		return nil, errors.Wrap(err, "could not create signed block")
	}
	if err = copyBlockData(u.b, sb); err != nil {
		return nil, errors.Wrap(err, "could not copy block data")
	}
	h, err := u.b.Block().Body().Execution()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution")
	}
	if err = sb.SetExecution(h); err != nil {
		return nil, errors.Wrap(err, "could not set execution")
	}
	// TODO: replace nil with proper variable for signed blinded blob sidecars
	// replace _ with blob bundle and use it in the response.
	payload, _, err := u.builder.SubmitBlindedBlock(ctx, sb, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not submit blinded block")
	}
	headerRoot, err := h.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get header root")
	}
	payloadRoot, err := payload.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "could not get payload root")
	}
	if headerRoot != payloadRoot {
		return nil, fmt.Errorf("header and payload root do not match, consider disconnect from relay to avoid further issues, "+
			"%#x != %#x", headerRoot, payloadRoot)
	}

	bb, err := u.protoBlock()
	if err != nil {
		return nil, errors.Wrap(err, "could not get proto block")
	}
	wb, err := consensusblocks.NewSignedBeaconBlock(bb)
	if err != nil {
		return nil, errors.Wrap(err, "could not create signed block")
	}
	if err = copyBlockData(sb, wb); err != nil {
		return nil, errors.Wrap(err, "could not copy block data")
	}
	if err = wb.SetExecution(payload); err != nil {
		return nil, errors.Wrap(err, "could not set execution")
	}

	txs, err := payload.Transactions()
	if err != nil {
		return nil, errors.Wrap(err, "could not get transactions from payload")
	}
	log.WithFields(logrus.Fields{
		"blockHash":    fmt.Sprintf("%#x", h.BlockHash()),
		"feeRecipient": fmt.Sprintf("%#x", h.FeeRecipient()),
		"gasUsed":      h.GasUsed(),
		"slot":         u.b.Block().Slot(),
		"txs":          len(txs),
	}).Info("Retrieved full payload from builder")

	return wb, nil
}

func copyBlockData(src interfaces.SignedBeaconBlock, dst interfaces.SignedBeaconBlock) error {
	agg, err := src.Block().Body().SyncAggregate()
	if err != nil {
		return errors.Wrap(err, "could not get sync aggregate")
	}
	parentRoot := src.Block().ParentRoot()
	stateRoot := src.Block().StateRoot()
	randaoReveal := src.Block().Body().RandaoReveal()
	graffiti := src.Block().Body().Graffiti()
	sig := src.Signature()
	blsToExecChanges, err := src.Block().Body().BLSToExecutionChanges()
	if err != nil && !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return errors.Wrap(err, "could not get bls to execution changes")
	}

	dst.SetSlot(src.Block().Slot())
	dst.SetProposerIndex(src.Block().ProposerIndex())
	dst.SetParentRoot(parentRoot[:])
	dst.SetStateRoot(stateRoot[:])
	dst.SetRandaoReveal(randaoReveal[:])
	dst.SetEth1Data(src.Block().Body().Eth1Data())
	dst.SetGraffiti(graffiti[:])
	dst.SetProposerSlashings(src.Block().Body().ProposerSlashings())
	dst.SetAttesterSlashings(src.Block().Body().AttesterSlashings())
	dst.SetAttestations(src.Block().Body().Attestations())
	dst.SetDeposits(src.Block().Body().Deposits())
	dst.SetVoluntaryExits(src.Block().Body().VoluntaryExits())
	if err = dst.SetSyncAggregate(agg); err != nil {
		return errors.Wrap(err, "could not set sync aggregate")
	}
	dst.SetSignature(sig[:])
	if err = dst.SetBLSToExecutionChanges(blsToExecChanges); err != nil && !errors.Is(err, consensus_types.ErrUnsupportedField) {
		return errors.Wrap(err, "could not set bls to execution changes")
	}

	return nil
}

func (u *unblinder) blindedProtoBlock() (proto.Message, error) {
	switch u.b.Version() {
	case version.Bellatrix:
		return &ethpb.SignedBlindedBeaconBlockBellatrix{
			Block: &ethpb.BlindedBeaconBlockBellatrix{
				Body: &ethpb.BlindedBeaconBlockBodyBellatrix{},
			},
		}, nil
	case version.Capella:
		return &ethpb.SignedBlindedBeaconBlockCapella{
			Block: &ethpb.BlindedBeaconBlockCapella{
				Body: &ethpb.BlindedBeaconBlockBodyCapella{},
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid version %s", version.String(u.b.Version()))
	}
}

func (u *unblinder) protoBlock() (proto.Message, error) {
	switch u.b.Version() {
	case version.Bellatrix:
		return &ethpb.SignedBeaconBlockBellatrix{
			Block: &ethpb.BeaconBlockBellatrix{
				Body: &ethpb.BeaconBlockBodyBellatrix{},
			},
		}, nil
	case version.Capella:
		return &ethpb.SignedBeaconBlockCapella{
			Block: &ethpb.BeaconBlockCapella{
				Body: &ethpb.BeaconBlockBodyCapella{},
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid version %s", version.String(u.b.Version()))
	}
}
