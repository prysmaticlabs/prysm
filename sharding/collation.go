package sharding

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/sharding/utils"
)

// Collation base struct.
type Collation struct {
	header *CollationHeader
	// body represents the serialized blob of a collation's transactions.
	// this is a read-only property.
	body []byte
	// transactions serves as a useful slice to store deserialized chunks from the
	// collation's body. Every time this transactions slice is updated, the serialized
	// body would need to be recalculated. This will be a useful property for proposers
	// in our system.
	transactions []*types.Transaction
}

// CollationHeader base struct.
type CollationHeader struct {
	// RLP decoding only works on exported properties of structs. In this case, we want
	// to keep collation properties as read-only and only accessible through getters.
	// We can accomplish this through this nested data property.
	data collationHeaderData
}

type collationHeaderData struct {
	ShardID           *big.Int        // the shard ID of the shard.
	ChunkRoot         *common.Hash    // the root of the chunk tree which identifies collation body.
	Period            *big.Int        // the period number in which collation to be included.
	ProposerAddress   *common.Address // address of the collation proposer.
	ProposerSignature []byte          // the proposer's signature for calculating collation hash.
}

var collationSizelimit = int64(math.Pow(float64(2), float64(20)))

// NewCollation initializes a collation and leaves it up to clients to serialize, deserialize
// and provide the body and transactions upon creation.
func NewCollation(header *CollationHeader, body []byte, transactions []*types.Transaction) *Collation {
	return &Collation{header, body, transactions}
}

// NewCollationHeader initializes a collation header struct.
func NewCollationHeader(shardID *big.Int, chunkRoot *common.Hash, period *big.Int, proposerAddress *common.Address, proposerSignature []byte) *CollationHeader {
	data := collationHeaderData{
		ShardID:           shardID,
		ChunkRoot:         chunkRoot,
		Period:            period,
		ProposerAddress:   proposerAddress,
		ProposerSignature: proposerSignature,
	}
	return &CollationHeader{data}
}

// Hash takes the keccak256 of the collation header's data contents.
func (h *CollationHeader) Hash() (hash common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, h.data)
	hw.Sum(hash[:0])
	return hash
}

// AddSig adds the signature of proposer after collationHeader gets signed
func (h *CollationHeader) AddSig(sig []byte) {
	h.data.ProposerSignature = sig
}

// ShardID the collation corresponds to.
func (h *CollationHeader) ShardID() *big.Int { return h.data.ShardID }

// Period the collation corresponds to.
func (h *CollationHeader) Period() *big.Int { return h.data.Period }

// ChunkRoot of the serialized collation body.
func (h *CollationHeader) ChunkRoot() *common.Hash { return h.data.ChunkRoot }

// EncodeRLP gives an encoded representation of the collation header.
func (h *CollationHeader) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(&h.data)
}

// DecodeRLP uses an RLP Stream to populate the data field of a collation header.
func (h *CollationHeader) DecodeRLP(s *rlp.Stream) error {
	return s.Decode(&h.data)
}

// Header returns the collation's header.
func (c *Collation) Header() *CollationHeader { return c.header }

// Body returns the collation's byte body.
func (c *Collation) Body() []byte { return c.body }

// Transactions returns an array of tx's in the collation.
func (c *Collation) Transactions() []*types.Transaction { return c.transactions }

// ProposerAddress is the coinbase addr of the creator for the collation.
func (c *Collation) ProposerAddress() *common.Address {
	return c.header.data.ProposerAddress
}

// CalculateChunkRoot updates the collation header's chunk root based on the body.
func (c *Collation) CalculateChunkRoot() {
	// TODO: this needs to be based on blob serialization.
	// For proof of custody we need to split chunks (body) into chunk + salt and
	// take the merkle root of that.
	chunkRoot := common.BytesToHash(c.body)
	c.header.data.ChunkRoot = &chunkRoot
}

// ConvertBackToTx converts raw blobs back to their original transactions.
func ConvertBackToTx(rawBlobs []utils.RawBlob) ([]*types.Transaction, error) {

	blobs := make([]*types.Transaction, len(rawBlobs))

	for i := 0; i < len(rawBlobs); i++ {

		blobs[i] = types.NewTransaction(0, common.HexToAddress("0x"), nil, 0, nil, nil)

		err := utils.ConvertFromRawBlob(&rawBlobs[i], blobs[i])
		if err != nil {
			return nil, fmt.Errorf("Creation of transactions from raw blobs failed: %v", err)
		}
	}
	return blobs, nil

}

// SerializeTxToBlob method serializes the input tx
// and returns the blobs in byte array.
func SerializeTxToBlob(txs []*types.Transaction) ([]byte, error) {

	blobs := make([]*utils.RawBlob, len(txs))
	for i := 0; i < len(txs); i++ {
		err := error(nil)
		blobs[i], err = utils.NewRawBlob(txs[i], false)
		if err != nil {
			return nil, fmt.Errorf("%v", err)
		}
	}
	serializedTx, err := utils.Serialize(blobs)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	if int64(len(serializedTx)) > collationSizelimit {
		return nil, fmt.Errorf("The serialized body exceeded the collation size limit: %v", serializedTx)
	}

	return serializedTx, nil
}

// DeserializeBlobToTx takes byte array blob and converts it back
// to original txs and returns the txs in tx array.
func DeserializeBlobToTx(serialisedBlob []byte) (*[]*types.Transaction, error) {

	deserializedBlobs, err := utils.Deserialize(serialisedBlob)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	txs, err := ConvertBackToTx(deserializedBlobs)

	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	return &txs, nil
}
