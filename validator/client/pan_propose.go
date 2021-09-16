package client

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	eth1Types "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"github.com/prysmaticlabs/prysm/validator/pandora"
	"golang.org/x/crypto/sha3"
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
	// errNilHeader
	errNilHeader = errors.New("pandora header is nil")
	// errPanShardingInfoNotFound
	errPanShardingInfoNotFound = errors.New("pandora sharding info not found in canonical head")
	// errSubmitShardingSignatureFailed
	errSubmitShardingSignatureFailed = errors.New("pandora sharding signature submission failed")

	errInvalidParentHash = errors.New("invalid parent hash")

	errInvalidBlockNumber = errors.New("invalid block number")
)

// processPandoraShardHeader method does the following tasks:
// - Get pandora block header, header hash, extraData from remote pandora node
// - Validate block header hash and extraData fields
// - Signs header hash using a validator key
// - Submit signature and header to pandora node
func (v *validator) processPandoraShardHeader(
	ctx context.Context,
	beaconBlk *ethpb.BeaconBlock,
	slot types.Slot,
	epoch types.Epoch,
	pubKey [48]byte,
) error {

	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	latestPandoraHash := eth1Types.EmptyRootHash
	latestPandoraBlkNum := uint64(0)

	// if pandoraShard is nil means there is no pandora blocks in pandora chain except block-0
	if beaconBlk.Body != nil && beaconBlk.Body.PandoraShard != nil {
		latestPandoraHash = common.BytesToHash(beaconBlk.Body.PandoraShard[0].Hash)
		latestPandoraBlkNum = beaconBlk.Body.PandoraShard[0].BlockNumber
	}

	// Request for pandora chain header
	header, headerHash, extraData, err := v.pandoraService.GetShardBlockHeader(ctx, latestPandoraHash, latestPandoraBlkNum+1, uint64(slot), uint64(epoch))
	if err != nil {
		log.WithField("blockSlot", slot).
			WithField("fmtKey", fmtKey).
			WithError(err).Error("Failed to request block header from pandora node")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return err
	}

	// Validate pandora chain header hash, extraData fields
	if err := v.verifyPandoraShardHeader(slot, epoch, header, headerHash, extraData, latestPandoraHash, latestPandoraBlkNum); err != nil {
		log.WithField("blockSlot", slot).
			WithField("fmtKey", fmtKey).
			WithError(err).Error("Failed to validate pandora block header")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return err
	}
	headerHashSig, err := v.keyManager.Sign(ctx, &validatorpb.SignRequest{
		PublicKey:       pubKey[:],
		SigningRoot:     headerHash[:],
		SignatureDomain: nil,
		Object:          nil,
	})

	if err != nil {
		log.WithField("blockSlot", slot).WithError(err).Error("Failed to sign pandora header hash")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return err
	}

	var headerHashSig96Bytes [96]byte
	copy(headerHashSig96Bytes[:], headerHashSig.Marshal())

	log.Debug("pandora sharding header info", "slot", slot, "sealHash",
		headerHash, "signature", common.Bytes2Hex(headerHashSig.Marshal()))

	// Submit bls signature to pandora
	if status, err := v.pandoraService.SubmitShardBlockHeader(
		ctx, header.Nonce.Uint64(), headerHash, headerHashSig96Bytes); !status || err != nil {
		// err nil means got success in api request but pandora does not write the header
		if err == nil {
			log.WithField("slot", slot).Debug("pandora refused to accept the sharding signature")
			err = errSubmitShardingSignatureFailed
		}
		log.WithError(err).
			WithField("pubKey", fmt.Sprintf("%#x", pubKey)).
			WithField("slot", slot).
			Error("Failed to process pandora chain shard header")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return err
	}

	var headerHashWithSig common.Hash
	if headerHashWithSig, err = calculateHeaderHashWithSig(header, *extraData, headerHashSig96Bytes); err != nil {
		log.WithError(err).
			WithField("pubKey", fmt.Sprintf("%#x", pubKey)).
			WithField("slot", slot).
			Error("Failed to process pandora chain shard header hash with signature")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return err
	}

	// fill pandora shard info with pandora header
	pandoraShard := v.preparePandoraShardingInfo(header, headerHashWithSig, headerHash, headerHashSig.Marshal())
	pandoraShards := make([]*ethpb.PandoraShard, 1)
	pandoraShards[0] = pandoraShard
	beaconBlk.Body.PandoraShard = pandoraShards
	log.WithField("slot", beaconBlk.Slot).Debug("successfully created pandora sharding block")

	// calling UpdateStateRoot api of beacon-chain so that state root will be updated after adding pandora shard
	updateBeaconBlk, err := v.updateStateRoot(ctx, beaconBlk)
	if err != nil {
		log.WithError(err).
			WithField("pubKey", fmt.Sprintf("%#x", pubKey)).
			WithField("slot", slot).
			Error("Failed to process pandora chain shard header")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return err
	}
	log.WithField("pandoraHeader", fmt.Sprintf("%+v", header)).Trace("Pandora sharding header")
	beaconBlk.StateRoot = updateBeaconBlk.StateRoot
	return nil
}

// verifyPandoraShardHeader verifies pandora sharding chain header hash and extraData field
func (v *validator) verifyPandoraShardHeader(
	slot types.Slot,
	epoch types.Epoch,
	header *eth1Types.Header,
	headerHash common.Hash,
	extraData *pandora.ExtraData,
	canonicalHash common.Hash,
	canonicalBlockNum uint64,
) error {

	// verify parent hash and block number
	if canonicalBlockNum > 0 {
		if canonicalHash != header.ParentHash {
			log.WithError(errInvalidParentHash).Error("invalid parent hash from pandora chain")
			return errInvalidParentHash
		}

		if canonicalBlockNum+1 != header.Number.Uint64() {
			log.WithError(errInvalidBlockNumber).Error("invalid block number from pandora chain")
			return errInvalidBlockNumber
		}
	}

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
	// verify slot number
	if extraData.Slot != uint64(slot) {
		log.WithError(errInvalidSlot).
			WithField("slot", slot).
			WithField("extraDataSlot", extraData.Slot).
			WithField("header", header.Extra).
			Error("invalid slot from pandora chain")
		return errInvalidSlot
	}
	// verify epoch number
	if extraData.Epoch != uint64(epoch) {
		log.WithError(errInvalidEpoch).Error("invalid epoch from pandora chain")
		return errInvalidEpoch
	}

	return nil
}

// panShardingCanonicalInfo method gets header hash and block number from sharding head
func (v *validator) updateStateRoot(
	ctx context.Context,
	beaconBlk *ethpb.BeaconBlock,
) (*ethpb.BeaconBlock, error) {

	log.WithField("slot", beaconBlk.Slot).WithField(
		"stateRoot", fmt.Sprintf("%X", beaconBlk.StateRoot)).Debug(
		"trying to update state root of the beacon block after adding pandora shard")
	// Request block from beacon node
	updatedBeaconBlk, err := v.validatorClient.UpdateStateRoot(ctx, beaconBlk)
	if err != nil {
		log.WithField("slot", updatedBeaconBlk.Slot).WithError(err).Error("Failed to update state root in beacon node")
		return beaconBlk, err
	}
	log.WithField("slot", updatedBeaconBlk.Slot).WithField(
		"stateRoot", fmt.Sprintf("%X", updatedBeaconBlk.StateRoot)).Debug(
		"successfully compute and update state root in beacon node")
	return updatedBeaconBlk, nil
}

// preparePandoraShardingInfo
func (v *validator) preparePandoraShardingInfo(
	header *eth1Types.Header,
	headerHashWithSig common.Hash,
	sealedHeaderHash common.Hash,
	sig []byte,
) *ethpb.PandoraShard {
	return &ethpb.PandoraShard{
		BlockNumber: header.Number.Uint64(),
		Hash:        headerHashWithSig.Bytes(),
		ParentHash:  header.ParentHash.Bytes(),
		StateRoot:   header.Root.Bytes(),
		TxHash:      header.TxHash.Bytes(),
		ReceiptHash: header.ReceiptHash.Bytes(),
		SealHash:    sealedHeaderHash.Bytes(),
		Signature:   sig,
	}
}

// SealHash returns the hash of a block prior to it being sealed.
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

func calculateHeaderHashWithSig(
	header *eth1Types.Header,
	pandoraExtraData pandora.ExtraData,
	signatureBytes [96]byte,
) (headerHash common.Hash, err error) {

	var blsSignatureBytes pandora.BlsSignatureBytes
	copy(blsSignatureBytes[:], signatureBytes[:])

	extraDataWithSig := new(pandora.PandoraExtraDataSig)
	extraDataWithSig.ExtraData = pandoraExtraData
	extraDataWithSig.BlsSignatureBytes = &blsSignatureBytes

	header.Extra, err = rlp.EncodeToBytes(extraDataWithSig)
	headerHash = header.Hash()
	log.WithField("headerHashWithSig", headerHash.Hex()).Debug("calculated header hash with signature")
	return
}
