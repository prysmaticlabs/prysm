package client

// Validator client proposer functions.
import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/prysm/validator/pandora"
	"golang.org/x/crypto/sha3"
	"time"

	eth1Types "github.com/ethereum/go-ethereum/core/types"
	pbtypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/mputil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	// errInvalidHeaderHash is returned if the header hash does not match with incoming header hash
	errInvalidHeaderHash = errors.New("invalid header hash")
	// errInvalidSlot is returned if the current slot does not match with incoming slot
	errInvalidSlot = errors.New("invalid slot")
	// errInvalidEpoch is returned if the epoch does not match with incoming epoch
	errInvalidEpoch = errors.New("invalid epoch")
	// errInvalidProposerIndex is returned if the proposer index does not match with incoming proposer index
	errInvalidProposerIndex = errors.New("invalid proposer index")
	// errInvalidTimestamp is returned if the timestamp of a block is higher than the current time
	errInvalidTimestamp = errors.New("invalid timestamp")
)

type signingFunc func(context.Context, *validatorpb.SignRequest) (bls.Signature, error)

const domainDataErr = "could not getverify domain data"
const signingRootErr = "could not get signing root"
const signExitErr = "could not sign voluntary exit proposal"

// ProposeBlock proposes a new beacon block for a given slot. This method collects the
// previous beacon block, any pending deposits, and ETH1 data from the beacon
// chain node to construct the new block. The new block is then processed with
// the state root computation, and finally signed by the validator before being
// sent back to the beacon node for broadcasting.
func (v *validator) ProposeBlock(ctx context.Context, slot types.Slot, pubKey [48]byte) {
	if slot == 0 {
		log.Debug("Assigned to genesis slot, skipping proposal")
		return
	}
	lock := mputil.NewMultilock(string(rune(roleProposer)), string(pubKey[:]))
	lock.Lock()
	defer lock.Unlock()
	ctx, span := trace.StartSpan(ctx, "validator.ProposeBlock")
	defer span.End()
	fmtKey := fmt.Sprintf("%#x", pubKey[:])

	span.AddAttributes(trace.StringAttribute("validator", fmt.Sprintf("%#x", pubKey)))
	log := log.WithField("pubKey", fmt.Sprintf("%#x", bytesutil.Trunc(pubKey[:])))

	// Sign randao reveal, it's used to request block from beacon node
	epoch := types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch)
	randaoReveal, err := v.signRandaoReveal(ctx, pubKey, epoch)
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

	// processPandoraShardHeader method process the block header from pandora chain
	if status, err := v.processPandoraShardHeader(ctx, b, slot, epoch, pubKey); !status || err != nil {
		log.WithError(err).
			WithField("pubKey", fmt.Sprintf("%#x", pubKey)).
			WithField("slot", slot).
			Error("Failed to process pandora chain shard header")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return
	}

	// Sign returned block from beacon node
	sig, domain, err := v.signBlock(ctx, pubKey, epoch, b)
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

	signingRoot, err := helpers.ComputeSigningRoot(b, domain.SignatureDomain)
	if err != nil {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		log.WithError(err).Error("Failed to compute signing root for block")
		return
	}

	if err := v.preBlockSignValidations(ctx, pubKey, b, signingRoot); err != nil {
		log.WithFields(
			blockLogFields(pubKey, b, nil),
		).WithError(err).Error("Failed block slashing protection check")
		return
	}

	if err := v.postBlockSignUpdate(ctx, pubKey, blk, signingRoot); err != nil {
		log.WithFields(
			blockLogFields(pubKey, b, sig),
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
	genesisResponse, err := nodeClient.GetGenesis(ctx, &pbtypes.Empty{})
	if err != nil {
		return errors.Wrap(err, "gRPC call to get genesis time failed")
	}
	totalSecondsPassed := timeutils.Now().Unix() - genesisResponse.GenesisTime.Seconds
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
func (v *validator) signRandaoReveal(ctx context.Context, pubKey [48]byte, epoch types.Epoch) ([]byte, error) {
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainRandao[:])
	if err != nil {
		return nil, errors.Wrap(err, domainDataErr)
	}
	if domain == nil {
		return nil, errors.New(domainDataErr)
	}

	var randaoReveal bls.Signature
	sszUint := types.SSZUint64(epoch)
	root, err := helpers.ComputeSigningRoot(&sszUint, domain.SignatureDomain)
	if err != nil {
		return nil, err
	}
	randaoReveal, err = v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     root[:],
		SignatureDomain: domain.SignatureDomain,
		Object:          &validatorpb.SignRequest_Epoch{Epoch: epoch},
	})
	if err != nil {
		return nil, err
	}
	return randaoReveal.Marshal(), nil
}

// Sign block with proposer domain and private key.
func (v *validator) signBlock(ctx context.Context, pubKey [48]byte, epoch types.Epoch, b *ethpb.BeaconBlock) ([]byte, *ethpb.DomainResponse, error) {
	domain, err := v.domainData(ctx, epoch, params.BeaconConfig().DomainBeaconProposer[:])
	if err != nil {
		return nil, nil, errors.Wrap(err, domainDataErr)
	}
	if domain == nil {
		return nil, nil, errors.New(domainDataErr)
	}

	var sig bls.Signature
	blockRoot, err := helpers.ComputeSigningRoot(b, domain.SignatureDomain)
	if err != nil {
		return nil, nil, errors.Wrap(err, signingRootErr)
	}
	sig, err = v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     blockRoot[:],
		SignatureDomain: domain.SignatureDomain,
		Object:          &validatorpb.SignRequest_Block{Block: b},
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not sign block proposal")
	}
	return sig.Marshal(), domain, nil
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

	exitRoot, err := helpers.ComputeSigningRoot(exit, domain.SignatureDomain)
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
func (v *validator) getGraffiti(ctx context.Context, pubKey [48]byte) ([]byte, error) {
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

// processPandoraShardHeader method does the following tasks:
// - Get pandora block header, header hash, extraData from remote pandora node
// - Validate block header hash and extraData fields
// - Signs header hash using a validator key
// - Submit signature and header to pandora node
func (v *validator) processPandoraShardHeader(ctx context.Context, beaconBlk *ethpb.BeaconBlock,
	slot types.Slot, epoch types.Epoch, pubKey [48]byte) (bool, error) {

	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	// Request for pandora chain header
	header, headerHash, extraData, err := v.pandoraService.GetShardBlockHeader(ctx)
	if err != nil {
		log.WithField("blockSlot", slot).
			WithField("fmtKey", fmtKey).
			WithError(err).Error("Failed to request block from pandora node")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return false, err
	}
	// Validate pandora chain header hash, extraData fields
	if err := v.verifyPandoraShardHeader(beaconBlk, slot, epoch, header, headerHash, extraData); err != nil {
		log.WithField("blockSlot", slot).
			WithField("fmtKey", fmtKey).
			WithError(err).Error("Failed to validate pandora block header")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return false, err
	}
	headerHashSig, err := v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     headerHash[:],
		SignatureDomain: nil,
		Object:          nil,
	})
	//compressedSig := headerHashSig.Marshal()
	if err != nil {
		log.WithField("blockSlot", slot).WithError(err).Error("Failed to sign pandora header hash")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return false, err
	}
	header.MixDigest = common.BytesToHash(headerHashSig.Marshal())
	var headerHashSig96Bytes [96]byte
	copy(headerHashSig96Bytes[:], headerHashSig.Marshal())
	return v.pandoraService.SubmitShardBlockHeader(ctx, header.Nonce.Uint64(), headerHash, headerHashSig96Bytes)
}

// verifyPandoraShardHeader verifies pandora sharding chain header hash and extraData field
func (v *validator) verifyPandoraShardHeader(beaconBlk *ethpb.BeaconBlock, slot types.Slot, epoch types.Epoch,
	header *eth1Types.Header, headerHash common.Hash, extraData *pandora.ExtraData) error {

	// verify header hash
	if sealHash(header) != headerHash {
		log.WithError(errInvalidHeaderHash).Error("invalid header hash from pandora chain")
		return errInvalidHeaderHash
	}
	// verify timestamp. Timestamp should not be future time
	if header.Time > uint64(timeutils.Now().Unix()) {
		log.WithError(errInvalidTimestamp).Error("invalid timestamp from pandora chain")
		return errInvalidTimestamp
	}

	// verify epoch number
	if extraData.Epoch != uint64(epoch) {
		log.WithError(errInvalidEpoch).Error("invalid epoch from pandora chain")
		return errInvalidEpoch
	}

	expectedTimeStart, err := helpers.SlotToTime(v.genesisTime, slot)

	if nil != err {
		return err
	}

	// verify slot number
	if extraData.Slot != uint64(slot) {
		log.WithError(errInvalidSlot).
			WithField("slot", slot).
			WithField("extraDataSlot", extraData.Slot).
			WithField("header", header.Extra).
			WithField("headerTime", header.Time).
			WithField("expectedTimeStart", expectedTimeStart.Unix()).
			WithField("currentSlot", helpers.CurrentSlot(v.genesisTime)).
			Error("invalid slot from pandora chain")
		return errInvalidSlot
	}

	err = helpers.VerifySlotTime(
		v.genesisTime,
		types.Slot(extraData.Slot),
		params.BeaconNetworkConfig().MaximumGossipClockDisparity,
	)

	if nil != err {
		log.WithError(errInvalidSlot).
			WithField("slot", slot).
			WithField("extraDataSlot", extraData.Slot).
			WithField("header", header.Extra).
			WithField("headerTime", header.Time).
			WithField("expectedTimeStart", expectedTimeStart.Unix()).
			WithField("currentSlot", helpers.CurrentSlot(v.genesisTime)).
			WithField("unixTimeNow", time.Now().Unix()).
			Error(err)

		return err
	}

	return nil
}

// sealHash returns the hash of a block prior to it being sealed.
func sealHash(header *eth1Types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	if err := rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra,
	}); err != nil {
		return eth1Types.EmptyRootHash
	}
	hasher.Sum(hash[:0])
	return hash
}
