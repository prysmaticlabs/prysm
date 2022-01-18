package client

// Validator client proposer functions.
import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/async"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/crypto/rand"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/network/forks"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	validatorpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/runtime/version"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/prysmaticlabs/prysm/validator/client/iface"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/types/known/emptypb"
)

type signingFunc func(context.Context, *validatorpb.SignRequest) (bls.Signature, error)

const domainDataErr = "could not get domain data"
const signingRootErr = "could not get signing root"
const signExitErr = "could not sign voluntary exit proposal"

// ProposeBlock proposes a new beacon block for a given slot. This method collects the
// previous beacon block, any pending deposits, and ETH1 data from the beacon
// chain node to construct the new block. The new block is then processed with
// the state root computation, and finally signed by the validator before being
// sent back to the beacon node for broadcasting.
func (v *validator) ProposeBlock(ctx context.Context, slot types.Slot, pubKey [fieldparams.BLSPubkeyLength]byte) {
	currEpoch := slots.ToEpoch(slot)
	switch {
	case currEpoch >= params.BeaconConfig().BellatrixForkEpoch:
		v.proposeBlockBellatrix(ctx, slot, pubKey)
	case currEpoch >= params.BeaconConfig().AltairForkEpoch:
		v.proposeBlockAltair(ctx, slot, pubKey)
	default:
		v.proposeBlockPhase0(ctx, slot, pubKey)
	}
}

func (v *validator) proposeBlockPhase0(ctx context.Context, slot types.Slot, pubKey [fieldparams.BLSPubkeyLength]byte) {
	if slot == 0 {
		log.Debug("Assigned to genesis slot, skipping proposal")
		return
	}
	lock := async.NewMultilock(fmt.Sprint(iface.RoleProposer), string(pubKey[:]))
	lock.Lock()
	defer lock.Unlock()
	ctx, span := trace.StartSpan(ctx, "validator.proposeBlockPhase0")
	defer span.End()
	fmtKey := fmt.Sprintf("%#x", pubKey[:])

	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))
	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])))

	// Sign randao reveal, it's used to request block from beacon node
	epoch := types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch)
	randaoReveal, err := v.signRandaoReveal(ctx, pubKey, epoch, slot)
	if err != nil {
		log.WithError(err).Error("Failed to sign randao reveal")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	g, err := v.getGraffiti(ctx, pubKey)
	if err != nil {
		// Graffiti is not a critical enough to fail block production and cause
		// validator to miss block reward. When failed, validator should continue
		// to produce the block.
		log.WithError(err).Warn("Could not get graffiti")
	}

	// Request block from beacon node
	b, err := v.validatorClient.GetBlock(ctx, &ethpb.BlockRequest{
		Slot:         slot,
		RandaoReveal: randaoReveal,
		Graffiti:     g,
	})
	if err != nil {
		log.WithField("blockSlot", slot).WithError(err).Error("Failed to request block from beacon node")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	// Sign returned block from beacon node
	sig, domain, err := v.signBlock(ctx, pubKey, epoch, slot, wrapper.WrappedPhase0BeaconBlock(b))
	if err != nil {
		log.WithError(err).Error("Failed to sign block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	blk := &ethpb.SignedBeaconBlock{
		Block:     b,
		Signature: sig,
	}

	signingRoot, err := signing.ComputeSigningRoot(b, domain.SignatureDomain)
	if err != nil {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		log.WithError(err).Error("Failed to compute signing root for block")
		return
	}

	if err := v.slashableProposalCheck(ctx, pubKey, wrapper.WrappedPhase0SignedBeaconBlock(blk), signingRoot); err != nil {
		log.WithFields(
			blockLogFields(pubKey, wrapper.WrappedPhase0BeaconBlock(b), nil),
		).WithError(err).Error("Failed block slashing protection check")
		return
	}

	// Propose and broadcast block via beacon node
	blkResp, err := v.validatorClient.ProposeBlock(ctx, blk)
	if err != nil {
		log.WithError(err).Error("Failed to propose block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", blkResp.BlockRoot)),
		trace.Int64Attribute("numDeposits", int64(len(b.Body.Deposits))),
		trace.Int64Attribute("numAttestations", int64(len(b.Body.Attestations))),
	)

	blkRoot := fmt.Sprintf("%#x", bytesutil.Trunc(blkResp.BlockRoot))
	log.WithFields(logrus.Fields{
		"slot":            b.Slot,
		"blockRoot":       blkRoot,
		"numAttestations": len(b.Body.Attestations),
		"numDeposits":     len(b.Body.Deposits),
		"graffiti":        string(b.Body.Graffiti),
	}).Info("Submitted new block")

	if v.emitAccountMetrics {
		ValidatorProposeSuccessVec.WithLabelValues(fmtKey).Inc()
	}
}

// This is a routine to propose altair compatible beacon blocks.
func (v *validator) proposeBlockAltair(ctx context.Context, slot types.Slot, pubKey [fieldparams.BLSPubkeyLength]byte) {
	if slot == 0 {
		log.Debug("Assigned to genesis slot, skipping proposal")
		return
	}
	ctx, span := trace.StartSpan(ctx, "validator.proposeBlockAltair")
	defer span.End()

	lock := async.NewMultilock(fmt.Sprint(iface.RoleProposer), string(pubKey[:]))
	lock.Lock()
	defer lock.Unlock()

	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))
	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])))

	// Sign randao reveal, it's used to request block from beacon node
	epoch := types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch)
	randaoReveal, err := v.signRandaoReveal(ctx, pubKey, epoch, slot)
	if err != nil {
		log.WithError(err).Error("Failed to sign randao reveal")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	g, err := v.getGraffiti(ctx, pubKey)
	if err != nil {
		// Graffiti is not a critical enough to fail block production and cause
		// validator to miss block reward. When failed, validator should continue
		// to produce the block.
		log.WithError(err).Warn("Could not get graffiti")
	}

	// Request block from beacon node
	b, err := v.validatorClient.GetBeaconBlock(ctx, &ethpb.BlockRequest{
		Slot:         slot,
		RandaoReveal: randaoReveal,
		Graffiti:     g,
	})
	if err != nil {
		log.WithField("blockSlot", slot).WithError(err).Error("Failed to request block from beacon node")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	altairBlk, ok := b.Block.(*ethpb.GenericBeaconBlock_Altair)
	if !ok {
		log.Error("Not an Altair block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	// Sign returned block from beacon node
	wb, err := wrapper.WrappedAltairBeaconBlock(altairBlk.Altair)
	if err != nil {
		log.WithError(err).Error("Failed to wrap block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	sig, domain, err := v.signBlock(ctx, pubKey, epoch, slot, wb)
	if err != nil {
		log.WithError(err).Error("Failed to sign block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	blk := &ethpb.SignedBeaconBlockAltair{
		Block:     altairBlk.Altair,
		Signature: sig,
	}

	signingRoot, err := signing.ComputeSigningRoot(altairBlk.Altair, domain.SignatureDomain)
	if err != nil {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		log.WithError(err).Error("Failed to compute signing root for block")
		return
	}

	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(blk)
	if err != nil {
		log.WithError(err).Error("Failed to wrap signed block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	if err := v.slashableProposalCheck(ctx, pubKey, wsb, signingRoot); err != nil {
		log.WithFields(
			blockLogFields(pubKey, wb, nil),
		).WithError(err).Error("Failed block slashing protection check")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	// Propose and broadcast block via beacon node
	blkResp, err := v.validatorClient.ProposeBeaconBlock(ctx, &ethpb.GenericSignedBeaconBlock{
		Block: &ethpb.GenericSignedBeaconBlock_Altair{Altair: blk},
	})
	if err != nil {
		log.WithError(err).Error("Failed to propose block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", blkResp.BlockRoot)),
		trace.Int64Attribute("numDeposits", int64(len(altairBlk.Altair.Body.Deposits))),
		trace.Int64Attribute("numAttestations", int64(len(altairBlk.Altair.Body.Attestations))),
	)

	blkRoot := fmt.Sprintf("%#x", bytesutil.Trunc(blkResp.BlockRoot))
	log.WithFields(logrus.Fields{
		"slot":            altairBlk.Altair.Slot,
		"blockRoot":       blkRoot,
		"numAttestations": len(altairBlk.Altair.Body.Attestations),
		"numDeposits":     len(altairBlk.Altair.Body.Deposits),
		"graffiti":        string(altairBlk.Altair.Body.Graffiti),
		"fork":            "altair",
	}).Info("Submitted new block")

	if v.emitAccountMetrics {
		ValidatorProposeSuccessVec.WithLabelValues(fmtKey).Inc()
	}
}

// ProposeExit performs a voluntary exit on a validator.
// The exit is signed by the validator before being sent to the beacon node for broadcasting.
func ProposeExit(
	ctx context.Context,
	validatorClient ethpb.BeaconNodeValidatorClient,
	nodeClient ethpb.NodeClient,
	signer signingFunc,
	pubKey []byte,
) error {
	ctx, span := trace.StartSpan(ctx, "validator.ProposeExit")
	defer span.End()

	indexResponse, err := validatorClient.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: pubKey})
	if err != nil {
		return errors.Wrap(err, "gRPC call to get validator index failed")
	}
	genesisResponse, err := nodeClient.GetGenesis(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.Wrap(err, "gRPC call to get genesis time failed")
	}
	totalSecondsPassed := prysmTime.Now().Unix() - genesisResponse.GenesisTime.Seconds
	currentEpoch := types.Epoch(uint64(totalSecondsPassed) / uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot)))

	exit := &ethpb.VoluntaryExit{Epoch: currentEpoch, ValidatorIndex: indexResponse.Index}
	sig, err := signVoluntaryExit(ctx, validatorClient, signer, pubKey, exit)
	if err != nil {
		return errors.Wrap(err, "failed to sign voluntary exit")
	}

	signedExit := &ethpb.SignedVoluntaryExit{Exit: exit, Signature: sig}
	exitResp, err := validatorClient.ProposeExit(ctx, signedExit)
	if err != nil {
		return errors.Wrap(err, "failed to propose voluntary exit")
	}

	span.AddAttributes(
		trace.StringAttribute("exitRoot", fmt.Sprintf("%#x", exitResp.ExitRoot)),
	)

	return nil
}

// Sign randao reveal with randao domain and private key.
func (v *validator) signRandaoReveal(ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, epoch types.Epoch, slot types.Slot) ([]byte, error) {
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainRandao[:])
	if err != nil {
		return nil, errors.Wrap(err, domainDataErr)
	}
	if domain == nil {
		return nil, errors.New(domainDataErr)
	}

	var randaoReveal bls.Signature
	sszUint := types.SSZUint64(epoch)
	root, err := signing.ComputeSigningRoot(&sszUint, domain.SignatureDomain)
	if err != nil {
		return nil, err
	}
	fork, err := forks.Fork(epoch)
	if err != nil {
		return nil, fmt.Errorf("could not get fork on current slot: %d", epoch)
	}
	randaoReveal, err = v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     root[:],
		SignatureDomain: domain.SignatureDomain,
		Object:          &validatorpb.SignRequest_Epoch{Epoch: epoch},
<<<<<<< HEAD
		Fork:            fork,
=======
		SigningSlot:     slot,
>>>>>>> develop
	})
	if err != nil {
		return nil, err
	}
	return randaoReveal.Marshal(), nil
}

// Sign block with proposer domain and private key.
func (v *validator) signBlock(ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, epoch types.Epoch, slot types.Slot, b block.BeaconBlock) ([]byte, *ethpb.DomainResponse, error) {
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainBeaconProposer[:])
	if err != nil {
		return nil, nil, errors.Wrap(err, domainDataErr)
	}
	if domain == nil {
		return nil, nil, errors.New(domainDataErr)
	}

	// TODO: I'm not sure if this is the right way to do this.
	fork, err := forks.Fork(epoch)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get fork on current slot: %d", epoch)
	}
	var sig bls.Signature
	switch b.Version() {

	case version.Bellatrix:
		block, ok := b.Proto().(*ethpb.BeaconBlockMerge)
		if !ok {
			return nil, nil, errors.New("could not convert obj to beacon block merge")
		}
		blockRoot, err := signing.ComputeSigningRoot(block, domain.SignatureDomain)
		if err != nil {
			return nil, nil, errors.Wrap(err, signingRootErr)
		}
		sig, err = v.keyManager.Sign(ctx, &validatorpb.SignRequest{
			PublicKey:       pubKey[:],
			SigningRoot:     blockRoot[:],
			SignatureDomain: domain.SignatureDomain,
			Object:          &validatorpb.SignRequest_BlockV3{BlockV3: block},
		})
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not sign block proposal")
		}
		return sig.Marshal(), domain, nil
	case version.Altair:
		block, ok := b.Proto().(*ethpb.BeaconBlockAltair)
		if !ok {
			return nil, nil, errors.New("could not convert obj to beacon block altair")
		}
		blockRoot, err := signing.ComputeSigningRoot(block, domain.SignatureDomain)
		if err != nil {
			return nil, nil, errors.Wrap(err, signingRootErr)
		}
		sig, err = v.keyManager.Sign(ctx, &validatorpb.SignRequest{
			PublicKey:       pubKey[:],
			SigningRoot:     blockRoot[:],
			SignatureDomain: domain.SignatureDomain,
			Object:          &validatorpb.SignRequest_BlockV2{BlockV2: block},
<<<<<<< HEAD
			Fork:            fork,
=======
			SigningSlot:     slot,
>>>>>>> develop
		})
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not sign block proposal")
		}
		return sig.Marshal(), domain, nil
	case version.Phase0:
		block, ok := b.Proto().(*ethpb.BeaconBlock)
		if !ok {
			return nil, nil, errors.New("could not convert obj to beacon block phase 0")
		}
		blockRoot, err := signing.ComputeSigningRoot(block, domain.SignatureDomain)
		if err != nil {
			return nil, nil, errors.Wrap(err, signingRootErr)
		}
		sig, err = v.keyManager.Sign(ctx, &validatorpb.SignRequest{
			PublicKey:       pubKey[:],
			SigningRoot:     blockRoot[:],
			SignatureDomain: domain.SignatureDomain,
			Object:          &validatorpb.SignRequest_Block{Block: block},
<<<<<<< HEAD
			Fork:            fork,
=======
			SigningSlot:     slot,
>>>>>>> develop
		})
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not sign block proposal")
		}
		return sig.Marshal(), domain, nil
	default:
		return nil, nil, errors.New("unknown block type")
	}
}

// Sign voluntary exit with proposer domain and private key.
func signVoluntaryExit(
	ctx context.Context,
	validatorClient ethpb.BeaconNodeValidatorClient,
	signer signingFunc,
	pubKey []byte,
	exit *ethpb.VoluntaryExit,
) ([]byte, error) {
	req := &ethpb.DomainRequest{
		Epoch:  exit.Epoch,
		Domain: params.BeaconConfig().DomainVoluntaryExit[:],
	}

	domain, err := validatorClient.DomainData(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, domainDataErr)
	}
	if domain == nil {
		return nil, errors.New(domainDataErr)
	}

	exitRoot, err := signing.ComputeSigningRoot(exit, domain.SignatureDomain)
	if err != nil {
		return nil, errors.Wrap(err, signingRootErr)
	}

	sig, err := signer(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey,
		SigningRoot:     exitRoot[:],
		SignatureDomain: domain.SignatureDomain,
		Object:          &validatorpb.SignRequest_Exit{Exit: exit},
	})
	if err != nil {
		return nil, errors.Wrap(err, signExitErr)
	}
	return sig.Marshal(), nil
}

// Gets the graffiti from cli or file for the validator public key.
func (v *validator) getGraffiti(ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte) ([]byte, error) {
	// When specified, default graffiti from the command line takes the first priority.
	if len(v.graffiti) != 0 {
		return v.graffiti, nil
	}

	if v.graffitiStruct == nil {
		return nil, errors.New("graffitiStruct can't be nil")
	}

	// When specified, individual validator specified graffiti takes the second priority.
	idx, err := v.validatorClient.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: pubKey[:]})
	if err != nil {
		return []byte{}, err
	}
	g, ok := v.graffitiStruct.Specific[idx.Index]
	if ok {
		return []byte(g), nil
	}

	// When specified, a graffiti from the ordered list in the file take third priority.
	if v.graffitiOrderedIndex < uint64(len(v.graffitiStruct.Ordered)) {
		graffiti := v.graffitiStruct.Ordered[v.graffitiOrderedIndex]
		v.graffitiOrderedIndex = v.graffitiOrderedIndex + 1
		err := v.db.SaveGraffitiOrderedIndex(ctx, v.graffitiOrderedIndex)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update graffiti ordered index")
		}
		return []byte(graffiti), nil
	}

	// When specified, a graffiti from the random list in the file take fourth priority.
	if len(v.graffitiStruct.Random) != 0 {
		r := rand.NewGenerator()
		r.Seed(time.Now().Unix())
		i := r.Uint64() % uint64(len(v.graffitiStruct.Random))
		return []byte(v.graffitiStruct.Random[i]), nil
	}

	// Finally, default graffiti if specified in the file will be used.
	if v.graffitiStruct.Default != "" {
		return []byte(v.graffitiStruct.Default), nil
	}

	return []byte{}, nil
}

// This is a routine to propose bellatrix compatible beacon blocks.
func (v *validator) proposeBlockBellatrix(ctx context.Context, slot types.Slot, pubKey [48]byte) {
	if slot == 0 {
		log.Debug("Assigned to genesis slot, skipping proposal")
		return
	}
	ctx, span := trace.StartSpan(ctx, "validator.proposeBlockBellatrix")
	defer span.End()

	lock := async.NewMultilock(fmt.Sprint(iface.RoleProposer), string(pubKey[:]))
	lock.Lock()
	defer lock.Unlock()

	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))
	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])))

	// Sign randao reveal, it's used to request block from beacon node
	epoch := types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch)
	randaoReveal, err := v.signRandaoReveal(ctx, pubKey, epoch, slot)
	if err != nil {
		log.WithError(err).Error("Failed to sign randao reveal")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	g, err := v.getGraffiti(ctx, pubKey)
	if err != nil {
		log.WithError(err).Warn("Could not get graffiti")
	}

	// Request block from beacon node
	b, err := v.validatorClient.GetBeaconBlock(ctx, &ethpb.BlockRequest{
		Slot:         slot,
		RandaoReveal: randaoReveal,
		Graffiti:     g,
	})
	if err != nil {
		log.WithField("blockSlot", slot).WithError(err).Error("Failed to request block from beacon node")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	mergeBlk, ok := b.Block.(*ethpb.GenericBeaconBlock_Merge)
	if !ok {
		log.Error("Not an Merge block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	// Sign returned block from beacon node
	wb, err := wrapper.WrappedMergeBeaconBlock(mergeBlk.Merge)
	if err != nil {
		log.WithError(err).Error("Failed to wrap block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	sig, domain, err := v.signBlock(ctx, pubKey, epoch, slot, wb)
	if err != nil {
		log.WithError(err).Error("Failed to sign block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}
	blk := &ethpb.SignedBeaconBlockMerge{
		Block:     mergeBlk.Merge,
		Signature: sig,
	}

	signingRoot, err := signing.ComputeSigningRoot(mergeBlk.Merge, domain.SignatureDomain)
	if err != nil {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		log.WithError(err).Error("Failed to compute signing root for block")
		return
	}

	wsb, err := wrapper.WrappedMergeSignedBeaconBlock(blk)
	if err != nil {
		log.WithError(err).Error("Failed to wrap signed block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	if err := v.slashableProposalCheck(ctx, pubKey, wsb, signingRoot); err != nil {
		log.WithFields(
			blockLogFields(pubKey, wb, nil),
		).WithError(err).Error("Failed block slashing protection check")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	// Propose and broadcast block via beacon node
	blkResp, err := v.validatorClient.ProposeBeaconBlock(ctx, &ethpb.GenericSignedBeaconBlock{
		Block: &ethpb.GenericSignedBeaconBlock_Merge{Merge: blk},
	})
	if err != nil {
		log.WithError(err).Error("Failed to propose block")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	blkRoot := fmt.Sprintf("%#x", bytesutil.Trunc(blkResp.BlockRoot))
	log.WithFields(logrus.Fields{
		"slot":            mergeBlk.Merge.Slot,
		"blockRoot":       blkRoot,
		"numAttestations": len(mergeBlk.Merge.Body.Attestations),
		"numDeposits":     len(mergeBlk.Merge.Body.Deposits),
		"graffiti":        string(mergeBlk.Merge.Body.Graffiti),
		"fork":            "bellatrix",
	}).Info("Submitted new block")

	if v.emitAccountMetrics {
		ValidatorProposeSuccessVec.WithLabelValues(fmtKey).Inc()
	}
}
