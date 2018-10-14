package types

import (
	"fmt"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/shardutil"
	"github.com/prysmaticlabs/prysm/validator/params"
)

// Collation defines a base struct that serves as a primitive equivalent of a "block"
// in a sharded Ethereum blockchain.
type Collation struct {
	header *CollationHeader
	// body represents the serialized blob of a collation's transactions.
	// this is a read-only property.
	body []byte
	// transactions serves as a useful slice to store deserialized chunks from the
	// collation's body. Every time this transactions slice is updated, the serialized
	// body would need to be recalculated. This will be a useful property for proposers
	// in our system.
	transactions []*gethTypes.Transaction
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
	ProposerSignature [32]byte        // the proposer's signature for calculating collation hash.
}

// NewCollation initializes a collation and leaves it up to validators to serialize, deserialize
// and provide the body and transactions upon creation.
func NewCollation(header *CollationHeader, body []byte, transactions []*gethTypes.Transaction) *Collation {
	return &Collation{
		header:       header,
		body:         body,
		transactions: transactions,
	}
}

// NewCollationHeader initializes a collation header struct.
func NewCollationHeader(shardID *big.Int, chunkRoot *common.Hash, period *big.Int, proposerAddress *common.Address, proposerSignature [32]byte) *CollationHeader {
	data := collationHeaderData{
		ShardID:           shardID,
		ChunkRoot:         chunkRoot,
		Period:            period,
		ProposerAddress:   proposerAddress,
		ProposerSignature: proposerSignature,
	}
	return &CollationHeader{data: data}
}

// Hash takes the blake2b of the collation header's data contents.
func (h *CollationHeader) Hash() (hash common.Hash) {
	encoded, err := rlp.EncodeToBytes(h.data)
	if err != nil {
		log.Errorf("Failed to RLP encode data: %v", err)
	}
	return hashutil.Hash(encoded)
}

// AddSig adds the signature of proposer after collationHeader gets signed.
func (h *CollationHeader) AddSig(sig [32]byte) {
	h.data.ProposerSignature = sig
}

// Sig is the signature the collation corresponds to.
func (h *CollationHeader) Sig() [32]byte { return h.data.ProposerSignature }

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
func (c *Collation) Transactions() []*gethTypes.Transaction { return c.transactions }

// ProposerAddress is the coinbase addr of the creator for the collation.
func (c *Collation) ProposerAddress() *common.Address {
	return c.header.data.ProposerAddress
}

// CalculateChunkRoot updates the collation header's chunk root based on the body.
func (c *Collation) CalculateChunkRoot() {
	chunks := BytesToChunks(c.body)          // wrapper allowing us to merklizing the chunks.
	chunkRoot := gethTypes.DeriveSha(chunks) // merklize the serialized blobs.
	c.header.data.ChunkRoot = &chunkRoot
}

// CalculatePOC calculates the Proof of Custody given the collation body and
// some salt, which is appended to each chunk in the collation body before it
// is hashed.
func (c *Collation) CalculatePOC(salt []byte) common.Hash {
	body := make([]byte, 0, len(c.body)*(1+len(salt)))

	for _, chunk := range c.body {
		body = append(append(body, salt...), chunk)
	}
	if len(c.body) == 0 {
		body = salt
	}
	chunks := BytesToChunks(body)      // wrapper allowing us to merklizing the chunks.
	return gethTypes.DeriveSha(chunks) // merklize the serialized blobs.
}

// BytesToChunks takes the collation body bytes and wraps it into type Chunks,
// which can be merklized.
func BytesToChunks(body []byte) Chunks {
	return Chunks(body)
}

// convertTxToRawBlob transactions into RawBlobs. This step encodes transactions uses RLP encoding
func convertTxToRawBlob(txs []*gethTypes.Transaction) ([]*shardutil.RawBlob, error) {
	blobs := make([]*shardutil.RawBlob, len(txs))
	for i := 0; i < len(txs); i++ {
		err := error(nil)
		blobs[i], err = shardutil.NewRawBlob(txs[i], false)
		if err != nil {
			return nil, err
		}
	}
	return blobs, nil
}

// SerializeTxToBlob converts transactions using two steps. First performs RLP encoding, and then blob encoding.
func SerializeTxToBlob(txs []*gethTypes.Transaction) ([]byte, error) {
	blobs, err := convertTxToRawBlob(txs)
	if err != nil {
		return nil, err
	}

	serializedTx, err := shardutil.Serialize(blobs)
	if err != nil {
		return nil, err
	}

	csl := params.DefaultCollationSizeLimit()
	if int64(len(serializedTx)) > csl {
		return nil, fmt.Errorf("the serialized body size %d exceeded the collation size limit %d", len(serializedTx), csl)
	}

	return serializedTx, nil
}

// convertRawBlobToTx converts raw blobs back to their original transactions.
func convertRawBlobToTx(rawBlobs []shardutil.RawBlob) ([]*gethTypes.Transaction, error) {
	blobs := make([]*gethTypes.Transaction, len(rawBlobs))

	for i := 0; i < len(rawBlobs); i++ {
		blobs[i] = gethTypes.NewTransaction(0, common.HexToAddress("0x"), nil, 0, nil, nil)

		err := shardutil.ConvertFromRawBlob(&rawBlobs[i], blobs[i])
		if err != nil {
			return nil, fmt.Errorf("creation of transactions from raw blobs failed: %v", err)
		}
	}
	return blobs, nil
}

// DeserializeBlobToTx takes byte array blob and converts it back
// to original txs and returns the txs in tx array.
func DeserializeBlobToTx(serialisedBlob []byte) (*[]*gethTypes.Transaction, error) {
	deserializedBlobs, err := shardutil.Deserialize(serialisedBlob)
	if err != nil {
		return nil, err
	}

	txs, err := convertRawBlobToTx(deserializedBlobs)

	if err != nil {
		return nil, err
	}

	return &txs, nil
}

// Chunks is a wrapper around a chunk array to implement DerivableList,
// which allows us to Merklize the chunks into the chunkRoot.
type Chunks []byte

// Len returns the number of chunks in this list.
func (ch Chunks) Len() int { return len(ch) }

// GetRlp returns the RLP encoding of one chunk from the list.
func (ch Chunks) GetRlp(i int) []byte {
	bytes, err := rlp.EncodeToBytes(ch[i])
	if err != nil {
		log.Errorf("Unable to RLP encode to bytes: %v", err)
	}
	return bytes
}
