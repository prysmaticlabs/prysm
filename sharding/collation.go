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
	proposerBid          *big.Int
	periodStartPrevHash  *common.Hash
	parentCollationHash  *common.Hash
	txListRoot           *common.Hash
	coinbase             *common.Address
	postStateRoot        *common.Hash
	receiptsRoot         *common.Hash
	sig                  []byte
	collationNumber      uint64
}

type Collations []*Collation

func (c *Collation) Header() *CollationHeader           { return c.header }
func (c *Collation) Transactions() []*types.Transaction { return c.transactions }
func (c *Collation) BidPrice() *big.Int { return new(big.Int).Set(c.header.proposerBid) }
func (c *Collation) Number() uint64      { return c.header.collationNumber }


func (c *Collation) SetHeader(h *CollationHeader) { c.header = h }
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

// CollationByNumber implements the sort interface to allow sorting a list of collations
// by their collationNumber.
type CollationByNumber Collations

func (s CollationByNumber) Len() int           { return len(s) }
func (s CollationByNumber) Less(i, j int) bool { return s[i].header.collationNumber < s[j].header.collationNumber }
func (s CollationByNumber) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// CollationByPrice implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type CollationByBid Collations

func (s CollationByBid) Len() int           { return len(s) }
func (s CollationByBid) Less(i, j int) bool { return s[i].header.proposerBid.Cmp(s[j].header.proposerBid) > 0 }
func (s CollationByBid) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *CollationByBid) Push(x interface{}) {
	*s = append(*s, x.(*Collation))
}

func (s *CollationByBid) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}