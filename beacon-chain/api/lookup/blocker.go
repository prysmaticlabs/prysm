package lookup

import (
	"context"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
)

// BlockIdParseError represents an error scenario where a block ID could not be parsed.
type BlockIdParseError struct {
	message string
}

// NewBlockIdParseError creates a new error instance.
func NewBlockIdParseError(reason error) BlockIdParseError {
	return BlockIdParseError{
		message: errors.Wrapf(reason, "could not parse block ID").Error(),
	}
}

// Error returns the underlying error message.
func (e BlockIdParseError) Error() string {
	return e.message
}

// Blocker is responsible for retrieving blocks.
type Blocker interface {
	Block(ctx context.Context, id []byte) (interfaces.ReadOnlySignedBeaconBlock, error)
}

// BeaconDbBlocker is an implementation of Blocker. It retrieves blocks from the beacon chain database.
type BeaconDbBlocker struct {
	BeaconDB         db.ReadOnlyDatabase
	ChainInfoFetcher blockchain.ChainInfoFetcher
}

// Block returns the beacon block for a given identifier. The identifier can be one of:
//   - "head" (canonical head in node's view)
//   - "genesis"
//   - "finalized"
//   - "justified"
//   - <slot>
//   - <hex encoded block root with '0x' prefix>
//   - <block root>
func (p *BeaconDbBlocker) Block(ctx context.Context, id []byte) (interfaces.ReadOnlySignedBeaconBlock, error) {
	var err error
	var blk interfaces.ReadOnlySignedBeaconBlock
	switch string(id) {
	case "head":
		blk, err = p.ChainInfoFetcher.HeadBlock(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve head block")
		}
	case "finalized":
		finalized := p.ChainInfoFetcher.FinalizedCheckpt()
		finalizedRoot := bytesutil.ToBytes32(finalized.Root)
		blk, err = p.BeaconDB.Block(ctx, finalizedRoot)
		if err != nil {
			return nil, errors.New("could not get finalized block from db")
		}
	case "genesis":
		blk, err = p.BeaconDB.GenesisBlock(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve genesis block")
		}
	default:
		stringId := strings.ToLower(string(id))
		if len(stringId) >= 2 && stringId[:2] == "0x" {
			decoded, err := hexutil.Decode(string(id))
			if err != nil {
				e := NewBlockIdParseError(err)
				return nil, &e
			}
			blk, err = p.BeaconDB.Block(ctx, bytesutil.ToBytes32(decoded))
			if err != nil {
				return nil, errors.Wrap(err, "could not retrieve block")
			}
		} else if len(id) == 32 {
			blk, err = p.BeaconDB.Block(ctx, bytesutil.ToBytes32(id))
			if err != nil {
				return nil, errors.Wrap(err, "could not retrieve block")
			}
		} else {
			slot, err := strconv.ParseUint(string(id), 10, 64)
			if err != nil {
				e := NewBlockIdParseError(err)
				return nil, &e
			}
			blks, err := p.BeaconDB.BlocksBySlot(ctx, primitives.Slot(slot))
			if err != nil {
				return nil, errors.Wrapf(err, "could not retrieve blocks for slot %d", slot)
			}
			_, roots, err := p.BeaconDB.BlockRootsBySlot(ctx, primitives.Slot(slot))
			if err != nil {
				return nil, errors.Wrapf(err, "could not retrieve block roots for slot %d", slot)
			}
			numBlks := len(blks)
			if numBlks == 0 {
				return nil, nil
			}
			for i, b := range blks {
				canonical, err := p.ChainInfoFetcher.IsCanonical(ctx, roots[i])
				if err != nil {
					return nil, errors.Wrapf(err, "could not determine if block root is canonical")
				}
				if canonical {
					blk = b
					break
				}
			}
		}
	}
	return blk, nil
}
