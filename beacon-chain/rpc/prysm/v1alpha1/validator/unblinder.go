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

	agg, err := u.b.Block().Body().SyncAggregate()
	if err != nil {
		return nil, errors.Wrap(err, "could not get sync aggregate")
	}
	h, err := u.b.Block().Body().Execution()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution")
	}
	parentRoot := u.b.Block().ParentRoot()
	stateRoot := u.b.Block().StateRoot()
	randaoReveal := u.b.Block().Body().RandaoReveal()
	graffiti := u.b.Block().Body().Graffiti()
	sig := u.b.Signature()
	blsToExecChanges, err := u.b.Block().Body().BLSToExecutionChanges()
	if err != nil && !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, errors.Wrap(err, "could not get bls to execution changes")
	}

	psb, err := u.blindedProtoBlock()
	if err != nil {
		return nil, errors.Wrap(err, "could not get blinded proto block")
	}
	sb, err := consensusblocks.NewSignedBeaconBlock(psb)
	if err != nil {
		return nil, errors.Wrap(err, "could not create signed block")
	}
	sb.SetSlot(u.b.Block().Slot())
	sb.SetProposerIndex(u.b.Block().ProposerIndex())
	sb.SetParentRoot(parentRoot[:])
	sb.SetStateRoot(stateRoot[:])
	sb.SetRandaoReveal(randaoReveal[:])
	sb.SetEth1Data(u.b.Block().Body().Eth1Data())
	sb.SetGraffiti(graffiti[:])
	sb.SetProposerSlashings(u.b.Block().Body().ProposerSlashings())
	sb.SetAttesterSlashings(u.b.Block().Body().AttesterSlashings())
	sb.SetAttestations(u.b.Block().Body().Attestations())
	sb.SetDeposits(u.b.Block().Body().Deposits())
	sb.SetVoluntaryExits(u.b.Block().Body().VoluntaryExits())
	if err = sb.SetSyncAggregate(agg); err != nil {
		return nil, errors.Wrap(err, "could not set sync aggregate")
	}
	sb.SetSignature(sig[:])
	if err = sb.SetBLSToExecutionChanges(blsToExecChanges); err != nil && !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, errors.Wrap(err, "could not set bls to execution changes")
	}
	if err = sb.SetExecution(h); err != nil {
		return nil, errors.Wrap(err, "could not set execution")
	}

	payload, err := u.builder.SubmitBlindedBlock(ctx, sb)
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

	agg, err = sb.Block().Body().SyncAggregate()
	if err != nil {
		return nil, errors.Wrap(err, "could not get sync aggregate")
	}
	parentRoot = sb.Block().ParentRoot()
	stateRoot = sb.Block().StateRoot()
	randaoReveal = sb.Block().Body().RandaoReveal()
	graffiti = sb.Block().Body().Graffiti()
	sig = sb.Signature()
	blsToExecChanges, err = sb.Block().Body().BLSToExecutionChanges()
	if err != nil && !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, errors.Wrap(err, "could not get bls to execution changes")
	}

	bb, err := u.protoBlock()
	if err != nil {
		return nil, errors.Wrap(err, "could not get proto block")
	}
	wb, err := consensusblocks.NewSignedBeaconBlock(bb)
	if err != nil {
		return nil, errors.Wrap(err, "could not create signed block")
	}
	wb.SetSlot(sb.Block().Slot())
	wb.SetProposerIndex(sb.Block().ProposerIndex())
	wb.SetParentRoot(parentRoot[:])
	wb.SetStateRoot(stateRoot[:])
	wb.SetRandaoReveal(randaoReveal[:])
	wb.SetEth1Data(sb.Block().Body().Eth1Data())
	wb.SetGraffiti(graffiti[:])
	wb.SetProposerSlashings(sb.Block().Body().ProposerSlashings())
	wb.SetAttesterSlashings(sb.Block().Body().AttesterSlashings())
	wb.SetAttestations(sb.Block().Body().Attestations())
	wb.SetDeposits(sb.Block().Body().Deposits())
	wb.SetVoluntaryExits(sb.Block().Body().VoluntaryExits())
	if err = wb.SetSyncAggregate(agg); err != nil {
		return nil, errors.Wrap(err, "could not set sync aggregate")
	}
	if err = wb.SetBLSToExecutionChanges(blsToExecChanges); err != nil && !errors.Is(err, consensus_types.ErrUnsupportedGetter) {
		return nil, errors.Wrap(err, "could not set bls to execution changes")
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
	case version.Deneb:
		return &ethpb.SignedBlindedBeaconBlockDeneb{
			Block: &ethpb.BlindedBeaconBlockDeneb{
				Body: &ethpb.BlindedBeaconBlockBodyDeneb{},
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
	case version.Deneb:
		return &ethpb.SignedBeaconBlockDeneb{
			Block: &ethpb.BeaconBlockDeneb{
				Body: &ethpb.BeaconBlockBodyDeneb{},
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid version %s", version.String(u.b.Version()))
	}
}
