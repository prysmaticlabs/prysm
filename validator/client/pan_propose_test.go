package client

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/prysmaticlabs/prysm/validator/pandora"
	"math/big"
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
	// Check with happy path
	header, extraData := testutil.NewPandoraBlock(98, 23)
	headerHash := sealHash(header)
	beaconBlkWithShard := testutil.NewBeaconBlockWithPandoraSharding(header, 98)
	beaconBlkWithShard.Block.Slot = 98
	beaconBlkWithShard.Block.ProposerIndex = 23
	epoch := types.Epoch(uint64(beaconBlkWithShard.Block.Slot) / 32)

	m.validatorClient.EXPECT().UpdateStateRoot(
		gomock.Any(), // ctx
		gomock.Any(), // beacon block
	).Return(beaconBlkWithShard.Block, nil)

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

	err = validator.processPandoraShardHeader(context.Background(), beaconBlkWithShard.Block, beaconBlkWithShard.Block.Slot, epoch, pubKey)
	require.NoError(t, err, "Should successfully process pandora sharding header")

	// Return rlp decoding error when calls `GetWork` api
	ErrRlpDecoding := errors.New("rlp: input contains more than one value")
	m.pandoraService.EXPECT().GetShardBlockHeader(
		gomock.Any(), // ctx
		gomock.Any(), // parentHash
		gomock.Any(), // next block number
	).Return(nil, common.Hash{}, nil, ErrRlpDecoding)
	err = validator.processPandoraShardHeader(context.Background(), beaconBlkWithShard.Block, beaconBlkWithShard.Block.Slot, epoch, pubKey)
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
	// Check with happy path
	header, extraData := testutil.NewPandoraBlock(98, 23)
	headerHash := sealHash(header)
	beaconBlkWithShard := testutil.NewBeaconBlockWithPandoraSharding(header, 98)
	beaconBlkWithShard.Block.Slot = 98
	beaconBlkWithShard.Block.ProposerIndex = 23
	epoch := types.Epoch(uint64(beaconBlkWithShard.Block.Slot) / 32)

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

	err = validator.processPandoraShardHeader(context.Background(), beaconBlkWithShard.Block, beaconBlkWithShard.Block.Slot, epoch, pubKey)
	require.ErrorContains(t, "Failed to process pandora chain shard header", err)
}

// TestValidator_ProposeBlock_Hash
/**
{
  difficulty: "0x1",
  extraData: "0xf866c30a800ab860a899054e1dd5ada5f5174edc532ffa39662cbfc90470233028096d7e41a3263114572cb7d0493ba213becec37f43145d041e0bfbaaf4bf8c2a7aeaebdd0d7fd6c326831b986a9802bf5e9ad1f180553ae0af77334cd4eb606ed71b0dc7db424e",
  gasLimit: "0x47ff2c",
  gasUsed: "0x0",
  hash: "0xaa1193c7d0d3cb6fbd33f5ddb748cd1e70e92b7bfc9667d4ecb4f61be63deb6c",
  logsBloom: "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  miner: "0xb46d14ef42ac9bb01303ba1842ea784e2460c7e7",
  mixHash: "0xa899054e1dd5ada5f5174edc532ffa39662cbfc90470233028096d7e41a32631",
  nonce: "0x0000000000000000",
  number: "0x4",
  parentHash: "0x3244474eb97faefc26df91a8c3d0f2a8f859855ba87b76b1cc6044cca29add40",
  receiptsRoot: "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
  sha3Uncles: "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
  size: "0x288",
  stateRoot: "0x03906b0760f3bec421d8a71c44273a5994c5f0e35b8b8d9e2112dc95a182aae6",
  timestamp: "0x60ed66d1",
  totalDifficulty: "0x80004",
  transactionsRoot: "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
}
*/
func TestValidator_ProposeBlock_Hash(t *testing.T) {
	var bloom gethTypes.Bloom
	bloom.SetBytes(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"))

	var header *gethTypes.Header
	header = &gethTypes.Header{
		ParentHash:  common.HexToHash("3244474eb97faefc26df91a8c3d0f2a8f859855ba87b76b1cc6044cca29add40"),
		UncleHash:   common.HexToHash("1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Coinbase:    common.HexToAddress("b46d14ef42ac9bb01303ba1842ea784e2460c7e7"),
		Root:        common.HexToHash("03906b0760f3bec421d8a71c44273a5994c5f0e35b8b8d9e2112dc95a182aae6"),
		TxHash:      common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
		ReceiptHash: common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
		Bloom:       bloom,
		Difficulty:  big.NewInt(1),
		Number:      big.NewInt(4),
		GasLimit:    4718380,
		GasUsed:     0,
		Time:        1626171089,
		Extra:       common.FromHex("0xf866c30a800ab860a899054e1dd5ada5f5174edc532ffa39662cbfc90470233028096d7e41a3263114572cb7d0493ba213becec37f43145d041e0bfbaaf4bf8c2a7aeaebdd0d7fd6c326831b986a9802bf5e9ad1f180553ae0af77334cd4eb606ed71b0dc7db424e"),
		MixDigest:   common.HexToHash("a899054e1dd5ada5f5174edc532ffa39662cbfc90470233028096d7e41a32631"),
		Nonce:       gethTypes.BlockNonce{0x0000000000000000},
	}

	//fmt.Printf("%+v", header)
	var extraDataWithSig pandora.PandoraExtraDataSig
	if err := rlp.DecodeBytes(header.Extra, &extraDataWithSig); err != nil {
		require.NoError(t, err)
	}

	expectedHeaderHash := "0xaa1193c7d0d3cb6fbd33f5ddb748cd1e70e92b7bfc9667d4ecb4f61be63deb6c"
	headerHash := header.Hash().Hex()
	assert.DeepEqual(t, expectedHeaderHash, headerHash)

	generatedHash, err := calculateHeaderHashWithSig(header, extraDataWithSig.ExtraData, *extraDataWithSig.BlsSignatureBytes)
	require.NoError(t, err)
	assert.DeepEqual(t, expectedHeaderHash, generatedHash.Hex())
}
