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

	log.Debug("")
	fmtKey := fmt.Sprintf("%#x", pubKey[:])

	// Request for canonical sharding info from beacon node
	err, parentHash, blockNumber := v.panShardingCanonicalInfo(ctx, slot, pubKey)
	if err != nil {
		return err
	}
	// Request for pandora chain header
	header, headerHash, extraData, err := v.pandoraService.GetShardBlockHeader(ctx, parentHash, blockNumber)
	if err != nil {
		log.WithField("blockSlot", slot).
			WithField("fmtKey", fmtKey).
			WithError(err).Error("Failed to request block from pandora node")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return err
	}
	// Validate pandora chain header hash, extraData fields
	if err := v.verifyPandoraShardHeader(slot, epoch, header, headerHash, extraData); err != nil {
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

	header.MixDigest = common.BytesToHash(headerHashSig.Marshal())
	var headerHashSig96Bytes [96]byte
	copy(headerHashSig96Bytes[:], headerHashSig.Marshal())

	// Submit bls signature to pandora
	if status, err := v.pandoraService.SubmitShardBlockHeader(
		ctx, header.Nonce.Uint64(), headerHash, headerHashSig96Bytes); !status || err != nil {

		log.WithError(err).
			WithField("pubKey", fmt.Sprintf("%#x", pubKey)).
			WithField("slot", slot).
			Error("Failed to process pandora chain shard header")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return err
	}

	// fill pandora shard info with pandora header
	pandoraShard := v.preparePandoraShardingInfo(header, headerHash, headerHashSig.Marshal())
	pandoraShards := make([]*ethpb.PandoraShard, 2)
	pandoraShards[0] = pandoraShard
	beaconBlk.Body.PandoraShard = pandoraShards
	return nil
}

// verifyPandoraShardHeader verifies pandora sharding chain header hash and extraData field
func (v *validator) verifyPandoraShardHeader(
	slot types.Slot,
	epoch types.Epoch,
	header *eth1Types.Header,
	headerHash common.Hash,
	extraData *pandora.ExtraData,
) error {

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
func (v *validator) panShardingCanonicalInfo(
	ctx context.Context,
	slot types.Slot,
	pubKey [48]byte,
) (error, common.Hash, uint64) {

	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	// Request block from beacon node
	headBlock, err := v.beaconClient.GetCanonicalBlock(ctx, nil)
	if err != nil {
		log.WithField("blockSlot", slot).WithError(err).Error("Failed to get canonical block from beacon node")
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return err, eth1Types.EmptyRootHash, 0
	}

	// Pandora shard info nil means there is no pandora sharding data into beacon state
	// if pandora sharding info is nil then canonical pandora block number should zero and hash will be unknown for first time
	if headBlock.Block.Body.PandoraShard == nil || len(headBlock.Block.Body.PandoraShard) == 0 {
		return nil, eth1Types.EmptyRootHash, 0
	}

	headerHash := common.BytesToHash(headBlock.Block.Body.PandoraShard[0].Hash)
	blkNum := headBlock.Block.Body.PandoraShard[0].BlockNumber
	log.WithField("slot", headBlock.Block.Slot).WithField("panHeaderHash", headerHash).WithField(
		"panBlockNum", blkNum).Debug("canonical pandora sharding info from beacon head block")
	return nil, headerHash, blkNum
}

// preparePandoraShardingInfo
func (v *validator) preparePandoraShardingInfo(
	header *eth1Types.Header,
	headerHash common.Hash,
	sig []byte,
) *ethpb.PandoraShard {

	pandoraShard := new(ethpb.PandoraShard)
	pandoraShard.BlockNumber = header.Number.Uint64()
	pandoraShard.Hash = headerHash.Bytes()
	pandoraShard.ParentHash = header.ParentHash.Bytes()
	pandoraShard.StateRoot = header.Root.Bytes()
	pandoraShard.TxHash = header.TxHash.Bytes()
	pandoraShard.ReceiptHash = header.ReceiptHash.Bytes()
	pandoraShard.Signature = sig

	return pandoraShard
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
