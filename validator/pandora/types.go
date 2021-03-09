package pandora

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	eth1Types "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	eth2Types "github.com/prysmaticlabs/eth2-types"

	"golang.org/x/crypto/sha3"
)

type PrepareBlockRequest struct {
	Slot 					uint64				`json:"slot"`
	Epoch   				uint64				`json:"epoch"`
	ProposerIndex 			uint64     			`json:"proposer_index"`
	CoinbaseAddress         common.Address  	`json:"coinbase_address"`
	BlsSignature			[]byte				`json:"bls_signature"`
}

type InsertBlockRequest struct {
	ExecutableBlock        *eth1Types.Block		`json:"block"`
	Sig 				   []byte 				`json:"signature"`
}

type PrepareBlockResponse struct {
	ExecutableBlock   	 	*eth1Types.Block		`json:"block"`
}

type InsertBlockResponse struct {
	Success 				bool 				`json:"success"`
}

type ExtraData struct {
	Slot 					uint64
	Epoch   				uint64
	ProposerIndex 			uint64
	CoinbaseAddress         common.Address
}

func NewExtraData(slot eth2Types.Slot, epoch eth2Types.Epoch,
	proposerIndex eth2Types.ValidatorIndex,coinbaseAddress common.Address) *ExtraData {

	var slotUint, epochUnit, proposerIndexUint uint64
	slotUint = uint64(slot)
	epochUnit = uint64(epoch)
	proposerIndexUint = uint64(proposerIndex)

	return &ExtraData {
		Slot: 				slotUint,
		Epoch: 				epochUnit,
		ProposerIndex: 		proposerIndexUint,
		CoinbaseAddress: 	coinbaseAddress,
	}
}

func NewPrepareBlockRequest(extraData *ExtraData, extraDataSignature []byte) *PrepareBlockRequest {
	return &PrepareBlockRequest{
		Slot: 				extraData.Slot,
		Epoch: 				extraData.Epoch,
		ProposerIndex: 		extraData.ProposerIndex,
		CoinbaseAddress: 	extraData.CoinbaseAddress,
		BlsSignature: 		extraDataSignature,
	}
}

func NewInsertBlockRequest(executableBlock *eth1Types.Block, blockSignature []byte) *InsertBlockRequest {
	return &InsertBlockRequest{
		ExecutableBlock: executableBlock,
		Sig: blockSignature,
	}
}

// hasherPool holds LegacyKeccak hashers.
var hasherPool = sync.Pool{
	New: func() interface{} {
		return sha3.NewLegacyKeccak256()
	},
}

func rlpHash(x interface{}) (h common.Hash) {
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	rlp.Encode(sha, x)
	sha.Read(h[:])
	return h
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (ed *ExtraData) Hash() common.Hash {
	return rlpHash(ed)
}