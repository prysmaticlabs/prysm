package client

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"testing"
)

// TestVerifyPandoraHeader_Ok method checks pandora header validation method
func TestVerifyPandoraShardHeader(t *testing.T) {
	validator, _, _, finish := setup(t)
	validator.enableVanguardNode = true
	defer finish()

	blk := testutil.NewBeaconBlock()
	blk.Block.Slot = 98
	blk.Block.ProposerIndex = 23
	epoch := types.Epoch(uint64(blk.Block.Slot) / 32)

	header, extraData := testutil.NewPandoraBlock(blk.Block.Slot, uint64(blk.Block.ProposerIndex))
	headerHash := sealHash(header)

	// Checks all the validations
	err := validator.verifyPandoraShardHeader(blk.Block.Slot, epoch, header, headerHash, extraData)
	require.NoError(t, err, "Should pass without any error")

	// Should get an `errInvalidHeaderHash` error
	header.Time = uint64(14265167)
	want := "invalid header hash"
	err = validator.verifyPandoraShardHeader(blk.Block.Slot, epoch, header, headerHash, extraData)
	require.ErrorContains(t, want, err, "Should get an errInvalidHeaderHash error")

	// Should get an `errInvalidSlot` error
	header.Time = uint64(1426516743)
	blk.Block.Slot = 90
	want = "invalid slot"
	err = validator.verifyPandoraShardHeader(blk.Block.Slot, epoch, header, headerHash, extraData)
	require.ErrorContains(t, want, err, "Should get an errInvalidSlot error")

	// Should get an `errInvalidEpoch` error
	blk.Block.Slot = 98
	epoch = 2
	want = "invalid epoch"
	err = validator.verifyPandoraShardHeader(blk.Block.Slot, epoch, header, headerHash, extraData)
	require.ErrorContains(t, want, err, "Should get an errInvalidEpoch error")
}

// TestProcessPandoraShardHeader method checks the `processPandoraShardHeader`
func TestProcessPandoraShardHeader(t *testing.T) {
	validator, m, _, finish := setup(t)
	validator.enableVanguardNode = true
	defer finish()

	secretKey, err := bls.SecretKeyFromBytes(bytesutil.PadTo([]byte{1}, 32))
	require.NoError(t, err, "Failed to generate key from bytes")
	publicKey := secretKey.PublicKey()
	var pubKey [48]byte
	copy(pubKey[:], publicKey.Marshal())
	km := &mockKeymanager{
		keysMap: map[[48]byte]bls.SecretKey{
			pubKey: secretKey,
		},
	}
	validator.keyManager = km
	blk := testutil.NewBeaconBlock()
	blk.Block.Slot = 98
	blk.Block.ProposerIndex = 23
	epoch := types.Epoch(uint64(blk.Block.Slot) / 32)

	// Check with happy path
	header, extraData := testutil.NewPandoraBlock(blk.Block.Slot, uint64(blk.Block.ProposerIndex))
	headerHash := sealHash(header)

	beaconBlkWithShard := testutil.NewBeaconBlockWithPandoraSharding(header, blk.Block.Slot)

	m.beaconChainClient.EXPECT().GetCanonicalBlock(
		gomock.Any(), // ctx
		gomock.Any(), // nil
	).Times(2).Return(beaconBlkWithShard, nil) // *SignedBeaconBlock, error

	m.pandoraService.EXPECT().GetShardBlockHeader(
		gomock.Any(), // ctx
		gomock.Any(), // parentHash
		gomock.Any(), // next block number
	).Return(header, headerHash, extraData, nil) // nil - error

	m.pandoraService.EXPECT().SubmitShardBlockHeader(
		gomock.Any(), // ctx
		gomock.Any(), // blockNonce
		gomock.Any(), // headerHash
		gomock.Any(), // sig
	).Return(true, nil)

	err = validator.processPandoraShardHeader(context.Background(), blk.Block, blk.Block.Slot, epoch, pubKey)
	require.NoError(t, err, "Should successfully process pandora sharding header")

	// Return rlp decoding error when calls `GetWork` api
	ErrRlpDecoding := errors.New("rlp: input contains more than one value")
	m.pandoraService.EXPECT().GetShardBlockHeader(
		gomock.Any(), // ctx
		gomock.Any(), // parentHash
		gomock.Any(), // next block number
	).Return(nil, common.Hash{}, nil, ErrRlpDecoding)
	err = validator.processPandoraShardHeader(context.Background(), blk.Block, blk.Block.Slot, epoch, pubKey)
	require.ErrorContains(t, "rlp: input contains more than one value", err)
}

// TestValidator_ProposeBlock_Failed_WhenSubmitShardInfoFails methods checks when `SubmitShardInfo` fails
func TestValidator_ProposeBlock_Failed_WhenSubmitShardInfoFails(t *testing.T) {
	validator, m, _, finish := setup(t)
	validator.enableVanguardNode = true
	defer finish()

	secretKey, err := bls.SecretKeyFromBytes(bytesutil.PadTo([]byte{1}, 32))
	require.NoError(t, err, "Failed to generate key from bytes")
	publicKey := secretKey.PublicKey()
	var pubKey [48]byte
	copy(pubKey[:], publicKey.Marshal())
	km := &mockKeymanager{
		keysMap: map[[48]byte]bls.SecretKey{
			pubKey: secretKey,
		},
	}
	validator.keyManager = km
	blk := testutil.NewBeaconBlock()
	blk.Block.Slot = 98
	blk.Block.ProposerIndex = 23
	epoch := types.Epoch(uint64(blk.Block.Slot) / 32)

	// Check with happy path
	header, extraData := testutil.NewPandoraBlock(blk.Block.Slot, uint64(blk.Block.ProposerIndex))
	headerHash := sealHash(header)

	beaconBlkWithShard := testutil.NewBeaconBlockWithPandoraSharding(header, blk.Block.Slot)

	m.beaconChainClient.EXPECT().GetCanonicalBlock(
		gomock.Any(), // ctx
		gomock.Any(), // nil
	).Return(beaconBlkWithShard, nil) // *SignedBeaconBlock, error

	m.pandoraService.EXPECT().GetShardBlockHeader(
		gomock.Any(), // ctx
		gomock.Any(), // parentHash
		gomock.Any(), // next block number
	).Return(header, headerHash, extraData, nil) // nil - error

	m.pandoraService.EXPECT().SubmitShardBlockHeader(
		gomock.Any(), // ctx
		gomock.Any(), // blockNonce
		gomock.Any(), // headerHash
		gomock.Any(), // sig
	).Return(false, errors.New("Failed to process pandora chain shard header"))

	err = validator.processPandoraShardHeader(context.Background(), blk.Block, blk.Block.Slot, epoch, pubKey)
	require.ErrorContains(t, "Failed to process pandora chain shard header", err)
}
