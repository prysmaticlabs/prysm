package enginev1_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestJsonMarshalUnmarshal(t *testing.T) {
	t.Run("payload attributes", func(t *testing.T) {
		random := bytesutil.PadTo([]byte("random"), fieldparams.RootLength)
		feeRecipient := bytesutil.PadTo([]byte("feeRecipient"), fieldparams.FeeRecipientLength)
		jsonPayload := &enginev1.PayloadAttributes{
			Timestamp:             1,
			PrevRandao:            random,
			SuggestedFeeRecipient: feeRecipient,
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.PayloadAttributes{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, uint64(1), payloadPb.Timestamp)
		require.DeepEqual(t, random, payloadPb.PrevRandao)
		require.DeepEqual(t, feeRecipient, payloadPb.SuggestedFeeRecipient)
	})
	t.Run("payload status", func(t *testing.T) {
		hash := bytesutil.PadTo([]byte("hash"), fieldparams.RootLength)
		jsonPayload := &enginev1.PayloadStatus{
			Status:          enginev1.PayloadStatus_INVALID,
			LatestValidHash: hash,
			ValidationError: "failed validation",
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.PayloadStatus{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, "INVALID", payloadPb.Status.String())
		require.DeepEqual(t, hash, payloadPb.LatestValidHash)
		require.DeepEqual(t, "failed validation", payloadPb.ValidationError)
	})
	t.Run("forkchoice state", func(t *testing.T) {
		head := bytesutil.PadTo([]byte("head"), fieldparams.RootLength)
		safe := bytesutil.PadTo([]byte("safe"), fieldparams.RootLength)
		finalized := bytesutil.PadTo([]byte("finalized"), fieldparams.RootLength)
		jsonPayload := &enginev1.ForkchoiceState{
			HeadBlockHash:      head,
			SafeBlockHash:      safe,
			FinalizedBlockHash: finalized,
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.ForkchoiceState{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, head, payloadPb.HeadBlockHash)
		require.DeepEqual(t, safe, payloadPb.SafeBlockHash)
		require.DeepEqual(t, finalized, payloadPb.FinalizedBlockHash)
	})
	t.Run("transition configuration", func(t *testing.T) {
		blockHash := []byte("head")
		jsonPayload := &enginev1.TransitionConfiguration{
			TerminalBlockHash:       blockHash,
			TerminalTotalDifficulty: params.BeaconConfig().TerminalTotalDifficulty,
			TerminalBlockNumber:     big.NewInt(0).Bytes(),
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.TransitionConfiguration{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, blockHash, payloadPb.TerminalBlockHash)

		require.DeepEqual(t, params.BeaconConfig().TerminalTotalDifficulty, payloadPb.TerminalTotalDifficulty)
		require.DeepEqual(t, big.NewInt(0).Bytes(), payloadPb.TerminalBlockNumber)
	})
	t.Run("execution payload", func(t *testing.T) {
		baseFeePerGas := big.NewInt(1770307273)
		parentHash := bytesutil.PadTo([]byte("parent"), fieldparams.RootLength)
		feeRecipient := bytesutil.PadTo([]byte("feeRecipient"), fieldparams.FeeRecipientLength)
		stateRoot := bytesutil.PadTo([]byte("stateRoot"), fieldparams.RootLength)
		receiptsRoot := bytesutil.PadTo([]byte("receiptsRoot"), fieldparams.RootLength)
		logsBloom := bytesutil.PadTo([]byte("logs"), fieldparams.LogsBloomLength)
		random := bytesutil.PadTo([]byte("random"), fieldparams.RootLength)
		extra := bytesutil.PadTo([]byte("extraData"), fieldparams.RootLength)
		hash := bytesutil.PadTo([]byte("hash"), fieldparams.RootLength)
		jsonPayload := &enginev1.ExecutionPayload{
			ParentHash:    parentHash,
			FeeRecipient:  feeRecipient,
			StateRoot:     stateRoot,
			ReceiptsRoot:  receiptsRoot,
			LogsBloom:     logsBloom,
			PrevRandao:    random,
			BlockNumber:   1,
			GasLimit:      2,
			GasUsed:       3,
			Timestamp:     4,
			ExtraData:     extra,
			BaseFeePerGas: baseFeePerGas.Bytes(),
			BlockHash:     hash,
			Transactions:  [][]byte{[]byte("hi")},
		}
		enc, err := json.Marshal(jsonPayload)
		require.NoError(t, err)
		payloadPb := &enginev1.ExecutionPayload{}
		require.NoError(t, json.Unmarshal(enc, payloadPb))
		require.DeepEqual(t, parentHash, payloadPb.ParentHash)
		require.DeepEqual(t, feeRecipient, payloadPb.FeeRecipient)
		require.DeepEqual(t, stateRoot, payloadPb.StateRoot)
		require.DeepEqual(t, receiptsRoot, payloadPb.ReceiptsRoot)
		require.DeepEqual(t, logsBloom, payloadPb.LogsBloom)
		require.DeepEqual(t, random, payloadPb.PrevRandao)
		require.DeepEqual(t, uint64(1), payloadPb.BlockNumber)
		require.DeepEqual(t, uint64(2), payloadPb.GasLimit)
		require.DeepEqual(t, uint64(3), payloadPb.GasUsed)
		require.DeepEqual(t, uint64(4), payloadPb.Timestamp)
		require.DeepEqual(t, extra, payloadPb.ExtraData)
		require.DeepEqual(t, bytesutil.PadTo(baseFeePerGas.Bytes(), fieldparams.RootLength), payloadPb.BaseFeePerGas)
		require.DeepEqual(t, hash, payloadPb.BlockHash)
		require.DeepEqual(t, [][]byte{[]byte("hi")}, payloadPb.Transactions)
	})
	t.Run("execution block", func(t *testing.T) {
		baseFeePerGas := big.NewInt(1770307273)
		want := &gethtypes.Header{
			Number:      big.NewInt(1),
			ParentHash:  common.BytesToHash([]byte("parent")),
			UncleHash:   common.BytesToHash([]byte("uncle")),
			Coinbase:    common.BytesToAddress([]byte("coinbase")),
			Root:        common.BytesToHash([]byte("uncle")),
			TxHash:      common.BytesToHash([]byte("txHash")),
			ReceiptHash: common.BytesToHash([]byte("receiptHash")),
			Bloom:       gethtypes.BytesToBloom([]byte("bloom")),
			Difficulty:  big.NewInt(2),
			GasLimit:    3,
			GasUsed:     4,
			Time:        5,
			BaseFee:     baseFeePerGas,
			Extra:       []byte("extraData"),
			MixDigest:   common.BytesToHash([]byte("mix")),
			Nonce:       gethtypes.EncodeNonce(6),
		}
		enc, err := json.Marshal(want)
		require.NoError(t, err)

		payloadItems := make(map[string]interface{})
		require.NoError(t, json.Unmarshal(enc, &payloadItems))

		blockHash := want.Hash()
		payloadItems["hash"] = blockHash.String()
		payloadItems["totalDifficulty"] = "0x393a2e53de197c"

		encodedPayloadItems, err := json.Marshal(payloadItems)
		require.NoError(t, err)

		payloadPb := &enginev1.ExecutionBlock{}
		require.NoError(t, json.Unmarshal(encodedPayloadItems, payloadPb))

		require.DeepEqual(t, blockHash, payloadPb.Hash)
		require.DeepEqual(t, want.Number, payloadPb.Number)
		require.DeepEqual(t, want.ParentHash, payloadPb.ParentHash)
		require.DeepEqual(t, want.UncleHash, payloadPb.UncleHash)
		require.DeepEqual(t, want.Coinbase, payloadPb.Coinbase)
		require.DeepEqual(t, want.Root, payloadPb.Root)
		require.DeepEqual(t, want.TxHash, payloadPb.TxHash)
		require.DeepEqual(t, want.ReceiptHash, payloadPb.ReceiptHash)
		require.DeepEqual(t, want.Bloom, payloadPb.Bloom)
		require.DeepEqual(t, want.Difficulty, payloadPb.Difficulty)
		require.DeepEqual(t, payloadItems["totalDifficulty"], payloadPb.TotalDifficulty)
		require.DeepEqual(t, want.GasUsed, payloadPb.GasUsed)
		require.DeepEqual(t, want.GasLimit, payloadPb.GasLimit)
		require.DeepEqual(t, want.Time, payloadPb.Time)
		require.DeepEqual(t, want.BaseFee, payloadPb.BaseFee)
		require.DeepEqual(t, want.Extra, payloadPb.Extra)
		require.DeepEqual(t, want.MixDigest, payloadPb.MixDigest)
		require.DeepEqual(t, want.Nonce, payloadPb.Nonce)
	})
	t.Run("execution block with txs as hashes", func(t *testing.T) {
		baseFeePerGas := big.NewInt(1770307273)
		want := &gethtypes.Header{
			Number:      big.NewInt(1),
			ParentHash:  common.BytesToHash([]byte("parent")),
			UncleHash:   common.BytesToHash([]byte("uncle")),
			Coinbase:    common.BytesToAddress([]byte("coinbase")),
			Root:        common.BytesToHash([]byte("uncle")),
			TxHash:      common.BytesToHash([]byte("txHash")),
			ReceiptHash: common.BytesToHash([]byte("receiptHash")),
			Bloom:       gethtypes.BytesToBloom([]byte("bloom")),
			Difficulty:  big.NewInt(2),
			GasLimit:    3,
			GasUsed:     4,
			Time:        5,
			BaseFee:     baseFeePerGas,
			Extra:       []byte("extraData"),
			MixDigest:   common.BytesToHash([]byte("mix")),
			Nonce:       gethtypes.EncodeNonce(6),
		}
		enc, err := json.Marshal(want)
		require.NoError(t, err)

		payloadItems := make(map[string]interface{})
		require.NoError(t, json.Unmarshal(enc, &payloadItems))

		blockHash := want.Hash()
		payloadItems["hash"] = blockHash.String()
		payloadItems["totalDifficulty"] = "0x393a2e53de197c"
		payloadItems["transactions"] = []string{"0xd57870623ea84ac3e2ffafbee9417fd1263b825b1107b8d606c25460dabeb693"}

		encodedPayloadItems, err := json.Marshal(payloadItems)
		require.NoError(t, err)

		payloadPb := &enginev1.ExecutionBlock{}
		require.NoError(t, json.Unmarshal(encodedPayloadItems, payloadPb))

		require.DeepEqual(t, blockHash, payloadPb.Hash)
		require.DeepEqual(t, want.Number, payloadPb.Number)
		require.DeepEqual(t, want.ParentHash, payloadPb.ParentHash)
		require.DeepEqual(t, want.UncleHash, payloadPb.UncleHash)
		require.DeepEqual(t, want.Coinbase, payloadPb.Coinbase)
		require.DeepEqual(t, want.Root, payloadPb.Root)
		require.DeepEqual(t, want.TxHash, payloadPb.TxHash)
		require.DeepEqual(t, want.ReceiptHash, payloadPb.ReceiptHash)
		require.DeepEqual(t, want.Bloom, payloadPb.Bloom)
		require.DeepEqual(t, want.Difficulty, payloadPb.Difficulty)
		require.DeepEqual(t, payloadItems["totalDifficulty"], payloadPb.TotalDifficulty)
		require.DeepEqual(t, want.GasUsed, payloadPb.GasUsed)
		require.DeepEqual(t, want.GasLimit, payloadPb.GasLimit)
		require.DeepEqual(t, want.Time, payloadPb.Time)
		require.DeepEqual(t, want.BaseFee, payloadPb.BaseFee)
		require.DeepEqual(t, want.Extra, payloadPb.Extra)
		require.DeepEqual(t, want.MixDigest, payloadPb.MixDigest)
		require.DeepEqual(t, want.Nonce, payloadPb.Nonce)

		// Expect no transaction objects in the unmarshaled data.
		require.Equal(t, 0, len(payloadPb.Transactions))
	})
	t.Run("execution block with full transaction data", func(t *testing.T) {
		baseFeePerGas := big.NewInt(1770307273)
		want := &gethtypes.Header{
			Number:      big.NewInt(1),
			ParentHash:  common.BytesToHash([]byte("parent")),
			UncleHash:   common.BytesToHash([]byte("uncle")),
			Coinbase:    common.BytesToAddress([]byte("coinbase")),
			Root:        common.BytesToHash([]byte("uncle")),
			TxHash:      common.BytesToHash([]byte("txHash")),
			ReceiptHash: common.BytesToHash([]byte("receiptHash")),
			Bloom:       gethtypes.BytesToBloom([]byte("bloom")),
			Difficulty:  big.NewInt(2),
			GasLimit:    3,
			GasUsed:     4,
			Time:        5,
			BaseFee:     baseFeePerGas,
			Extra:       []byte("extraData"),
			MixDigest:   common.BytesToHash([]byte("mix")),
			Nonce:       gethtypes.EncodeNonce(6),
		}
		enc, err := json.Marshal(want)
		require.NoError(t, err)

		payloadItems := make(map[string]interface{})
		require.NoError(t, json.Unmarshal(enc, &payloadItems))

		tx := gethtypes.NewTransaction(
			1,
			common.BytesToAddress([]byte("hi")),
			big.NewInt(0),
			21000,
			big.NewInt(1e6),
			[]byte{},
		)
		txs := []*gethtypes.Transaction{tx}

		blockHash := want.Hash()
		payloadItems["hash"] = blockHash.String()
		payloadItems["totalDifficulty"] = "0x393a2e53de197c"
		payloadItems["transactions"] = txs

		encodedPayloadItems, err := json.Marshal(payloadItems)
		require.NoError(t, err)

		payloadPb := &enginev1.ExecutionBlock{}
		require.NoError(t, json.Unmarshal(encodedPayloadItems, payloadPb))

		require.DeepEqual(t, blockHash, payloadPb.Hash)
		require.DeepEqual(t, want.Number, payloadPb.Number)
		require.DeepEqual(t, want.ParentHash, payloadPb.ParentHash)
		require.DeepEqual(t, want.UncleHash, payloadPb.UncleHash)
		require.DeepEqual(t, want.Coinbase, payloadPb.Coinbase)
		require.DeepEqual(t, want.Root, payloadPb.Root)
		require.DeepEqual(t, want.TxHash, payloadPb.TxHash)
		require.DeepEqual(t, want.ReceiptHash, payloadPb.ReceiptHash)
		require.DeepEqual(t, want.Bloom, payloadPb.Bloom)
		require.DeepEqual(t, want.Difficulty, payloadPb.Difficulty)
		require.DeepEqual(t, payloadItems["totalDifficulty"], payloadPb.TotalDifficulty)
		require.DeepEqual(t, want.GasUsed, payloadPb.GasUsed)
		require.DeepEqual(t, want.GasLimit, payloadPb.GasLimit)
		require.DeepEqual(t, want.Time, payloadPb.Time)
		require.DeepEqual(t, want.BaseFee, payloadPb.BaseFee)
		require.DeepEqual(t, want.Extra, payloadPb.Extra)
		require.DeepEqual(t, want.MixDigest, payloadPb.MixDigest)
		require.DeepEqual(t, want.Nonce, payloadPb.Nonce)
		require.Equal(t, 1, len(payloadPb.Transactions))
		require.DeepEqual(t, txs[0].Hash(), payloadPb.Transactions[0].Hash())
	})
}

func TestPayloadIDBytes_MarshalUnmarshalJSON(t *testing.T) {
	item := [8]byte{1, 0, 0, 0, 0, 0, 0, 0}
	enc, err := json.Marshal(enginev1.PayloadIDBytes(item))
	require.NoError(t, err)
	require.DeepEqual(t, "\"0x0100000000000000\"", string(enc))
	res := &enginev1.PayloadIDBytes{}
	err = res.UnmarshalJSON(enc)
	require.NoError(t, err)
	require.Equal(t, true, item == *res)
}
