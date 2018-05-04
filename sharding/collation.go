package sharding

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

// Collation base struct.
type Collation struct {
	header       *CollationHeader
	body         []byte
	transactions []*types.Transaction
}

// CollationHeader base struct.
type CollationHeader struct {
	shardID           *big.Int        //the shard ID of the shard.
	chunkRoot         *common.Hash    //the root of the chunk tree which identifies collation body.
	period            *big.Int        //the period number in which collation to be included.
	proposerAddress   *common.Address //address of the collation proposer.
	proposerSignature []byte          //the proposer's signature for calculating collation hash.
}

// Hash takes the keccak256 of the collation header's contents.
func (h *CollationHeader) Hash() (hash common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, h)
	hw.Sum(hash[:0])
	return hash
}

// Header returns the collation's header.
func (c *Collation) Header() *CollationHeader { return c.header }

// Transactions returns an array of tx's in the collation.
func (c *Collation) Transactions() []*types.Transaction { return c.transactions }

// ShardID is the identifier for a shard.
func (c *Collation) ShardID() *big.Int { return c.header.shardID }

// Period the collation corresponds to.
func (c *Collation) Period() *big.Int { return c.header.period }

// ProposerAddress is the coinbase addr of the creator for the collation.
func (c *Collation) ProposerAddress() *common.Address { return c.header.proposerAddress }

// SetHeader updates the collation's header.
func (c *Collation) SetHeader(h *CollationHeader) { c.header = h }

// AddTransaction adds to the collation's body of tx blobs.
func (c *Collation) AddTransaction(tx *types.Transaction) {
	// TODO: Include blob serialization instead.
	c.transactions = append(c.transactions, tx)
}
