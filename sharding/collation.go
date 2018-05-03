package sharding

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

type Collation struct {
	header *CollationHeader
	body   []byte
}

type CollationHeader struct {
	shardID           *big.Int        //the shard ID of the shard
	chunkRoot         *common.Hash    //the root of the chunk tree which identifies collation body
	period            *big.Int        //the period number in which collation to be included
	proposerAddress   *common.Address //address of the collation proposer
	proposerSignature []byte          //the proposer's signature for calculating collation hash
}

// Hash takes the keccak256 of the collation header's contents.
func (h *CollationHeader) Hash() (hash common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, h)
	hw.Sum(hash[:0])
	return hash
}

func (c *Collation) Header() *CollationHeader         { return c.header }
func (c *Collation) ShardID() *big.Int                { return c.header.shardID }
func (c *Collation) Period() *big.Int                 { return c.header.period }
func (c *Collation) ProposerAddress() *common.Address { return c.header.proposerAddress }

func (c *Collation) SetHeader(h *CollationHeader) { c.header = h }
