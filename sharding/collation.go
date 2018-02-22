package sharding

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Collation struct {
	header       *CollationHeader
	transactions []*types.Transaction
}

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

func (c *Collation) Header() *CollationHeader           { return c.header }
func (c *Collation) Transactions() []*types.Transaction { return c.transactions }

func (c *Collation) SetHeader(h *collationHeader) { c.header = h }
func (c *Collation) AddTransaction(tx *types.Transaction) {
	// TODO: Check transaction does not exceed gas limit
	c.transactions = append(c.transactions, tx)
}

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
