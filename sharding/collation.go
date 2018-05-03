package sharding

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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

func (c *Collation) Header() *CollationHeader         { return c.header }
func (c *Collation) ShardID() *big.Int                { return c.header.shardID }
func (c *Collation) Period() *big.Int                 { return c.header.period }
func (c *Collation) ProposerAddress() *common.Address { return c.header.proposerAddress }

func (c *Collation) SetHeader(h *CollationHeader) { c.header = h }
