package sharding

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Collation defines the base struct across the sharding clients
type Collation struct {
	header       *CollationHeader
	transactions []*types.Transaction
}

// CollationHeader defines the struct that will be serialized and broadcast over the wire protocol to collators
type CollationHeader struct {
	shardID              *big.Int
	expectedPeriodNumber *big.Int
	periodStartPrevHash  *common.Hash
	parentCollationHash  *common.Hash
	txListRoot           *common.Hash
	coinbase             *common.Address
	postStateRoot        *common.Hash
	receiptsRoot         *common.Hash
	sig                  []byte
}

// Header fetches a collation header object
func (c *Collation) Header() *CollationHeader { return c.header }

// Transactions fetches an array of transaction types stored in a collation
func (c *Collation) Transactions() []*types.Transaction { return c.transactions }

// SetHeader updates a collation header object
func (c *Collation) SetHeader(h *CollationHeader) { c.header = h }

// AddTransaction adds a tx to the collation body
func (c *Collation) AddTransaction(tx *types.Transaction) {
	// TODO: Check transaction does not exceed gas limit
	c.transactions = append(c.transactions, tx)
}

// GasUsed gives us a count of gas used in the entire tx body of the collation
func (c *Collation) GasUsed() *big.Int {
	g := uint64(0)
	for _, tx := range c.transactions {
		if g > math.MaxUint64-(g+tx.Gas()) {
			g = math.MaxUint64
			break
		}
		g += tx.Gas()
	}
	return big.NewInt(0).SetUint64(g)
}
