package lookup

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/peerdas"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db/filesystem"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/core"
	"github.com/prysmaticlabs/prysm/v5/cmd/beacon-chain/flags"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	log "github.com/sirupsen/logrus"
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
	Blobs(ctx context.Context, id string, indices map[uint64]bool) ([]*blocks.VerifiedROBlob, *core.RpcError)
}

// BeaconDbBlocker is an implementation of Blocker. It retrieves blocks from the beacon chain database.
type BeaconDbBlocker struct {
	BeaconDB           db.ReadOnlyDatabase
	ChainInfoFetcher   blockchain.ChainInfoFetcher
	GenesisTimeFetcher blockchain.TimeFetcher
	BlobStorage        *filesystem.BlobStorage
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

// blobsFromStoredBlobs retrieves blobs corresponding to `indices` and `root` from the store, expecting blobs to be
// stored directly (aka. no data columns).
func (p *BeaconDbBlocker) blobsFromStoredBlobs(indices map[uint64]bool, root []byte) ([]*blocks.VerifiedROBlob, *core.RpcError) {
	// If no indices are provided in the request, we fetch all available blobs for the block.
	if len(indices) == 0 {
		// Get all blob indices for the block.
		indicesMap, err := p.BlobStorage.Indices(bytesutil.ToBytes32(root))
		if err != nil {
			log.WithField("blockRoot", hexutil.Encode(root)).Error(errors.Wrapf(err, "could not retrieve blob indices for root %#x", root))
			return nil, &core.RpcError{Err: fmt.Errorf("could not retrieve blob indices for root %#x", root), Reason: core.Internal}
		}

		for indice, exists := range indicesMap {
			if exists {
				indices[uint64(indice)] = true
			}
		}
	}

	// Retrieve from the store the blobs corresponding to the indices for this block root.
	blobs := make([]*blocks.VerifiedROBlob, 0, len(indices))
	for index := range indices {
		vblob, err := p.BlobStorage.Get(bytesutil.ToBytes32(root), index)
		if err != nil {
			log.WithFields(log.Fields{
				"blockRoot": hexutil.Encode(root),
				"blobIndex": index,
			}).Error(errors.Wrapf(err, "could not retrieve blob for block root %#x at index %d", root, index))
			return nil, &core.RpcError{Err: fmt.Errorf("could not retrieve blob for block root %#x at index %d", root, index), Reason: core.Internal}
		}
		blobs = append(blobs, &vblob)
	}

	return blobs, nil
}

// blobsFromStoredDataColumns retrieves data columns from the store, convert them to blobs, and return blobs corresponding to `indices` and `root` from the store,
// expecting data columns to be stored (aka. no blobs).
// If not all data columns are available, the function returns a "not found" error.
// This function expects the block associated with root has blobs.
func (p *BeaconDbBlocker) blobsFromStoredDataColumns(indices map[uint64]bool, rootBytes []byte) ([]*blocks.VerifiedROBlob, *core.RpcError) {
	// Multiple implementations are possible here, depending on the data storage strategy.
	// 1. If the `--subscribe-all-subnets` flag is not set, the respond with a "not found" error.
	// 2. If all columns are available (either because the `--subscribe-all-subnets` flag is set or because we have all columns thanks to vaidator custody),
	//    then respond with the blobs. In the contrary, we Haharespond with a "not found" error.
	//    However, this strategy may confuse the requester, since the ability of the node to respond to the request will depend on the number of validators attached to the node.
	//    So, sometimes the node will be able to respond, and sometimes not, which is not ideal.
	// 3. If at least half of the columns are available, then we can reconstruct and respond with the blobs. As with `2.`, this strategy may confuse the requester because
	//    the ability of the node to respond to the request will depend on the number of validators attached to the node.
	//    So, sometimes the node will be able to respond, and sometimes not, which is not ideal.
	// 4. If some columns are missing, we requests the missing columns to peers until we are able to reconstruct, then we reconstruct and respond with the blobs.
	//    Unlike `2.` and `3.`, the node will always be able to respond to the request, but it will provoke possibly a lot of network traffic.
	// ==> In this implementation, we choose the strategy `1.`.
	if !flags.Get().SubscribeToAllSubnets {
		return nil, &core.RpcError{
			Err:    errors.Errorf("please start the beacon node with the `--%s` flag before querying this endpoint", flags.SubscribeToAllSubnets.Name),
			Reason: core.NotFound,
		}
	}

	root := bytesutil.ToBytes32(rootBytes)

	// Check we effectively custody all data columns for this block.
	storedDataColumns, err := p.BlobStorage.ColumnIndices(root)
	if err != nil {
		log.WithField("blockRoot", hexutil.Encode(rootBytes)).Error(errors.Wrap(err, "Could not retrieve columns indices stored for block root"))
		return nil, &core.RpcError{Err: errors.Wrap(err, "could not retrieve columns indices stored for block root"), Reason: core.Internal}
	}

	storedDataColumnsCount := len(storedDataColumns)
	if storedDataColumnsCount != fieldparams.NumberOfColumns {
		// The main reason for this error is that the node did not (yet) completed the backfill.

		log.WithFields(log.Fields{
			"blockRoot":           hexutil.Encode(rootBytes),
			"custodyColumnsCount": storedDataColumnsCount,
			"columnsCount":        fieldparams.NumberOfColumns,
		}).Warning("not all data columns are available for this block")

		return nil, &core.RpcError{
			Err:    errors.Errorf("not all data columns are available for this blob. Wanted: %d, got: %d. Please retry later.", fieldparams.NumberOfColumns, storedDataColumnsCount),
			Reason: core.NotFound,
		}
	}

	columnsCount := fieldparams.NumberOfColumns

	if columnsCount%2 != 0 {
		log.WithField("columnsCount", columnsCount).Error("The number of columns must be even")
		return nil, &core.RpcError{Err: errors.New("the number of columns must be even"), Reason: core.Internal}
	}

	// Retrieve columns corresponding to the (non-extended) blobs.
	blobColumnsCount := fieldparams.NumberOfColumns / 2

	dataColumnsSidecar := make([]*ethpb.DataColumnSidecar, 0, blobColumnsCount)
	for i := range blobColumnsCount {
		dataColumnSidecar, err := p.BlobStorage.GetColumn(root, uint64(i))
		if err != nil {
			log.WithFields(log.Fields{
				"blockRoot": hexutil.Encode(rootBytes),
				"column":    i,
			}).Error(errors.Wrapf(err, "could not retrieve column %d for block root %#x", i, root))

			return nil, &core.RpcError{Err: fmt.Errorf("could not retrieve column %d for block root %#x", i, root), Reason: core.Internal}
		}

		dataColumnsSidecar = append(dataColumnsSidecar, dataColumnSidecar)
	}

	// Compute verified RO blobs from the data columns.
	verifiedROBlobs, err := peerdas.Blobs(dataColumnsSidecar)
	if err != nil {
		log.WithField("blockRoot", hexutil.Encode(rootBytes)).Error(errors.Wrap(err, "could not compute blobs from data columns"))
		return nil, &core.RpcError{Err: errors.Wrap(err, "could not compute blobs from data columns"), Reason: core.Internal}
	}

	return verifiedROBlobs, nil
}

// Blobs returns the blobs for a given block id identifier and blob indices. The identifier can be one of:
//   - "head" (canonical head in node's view)
//   - "genesis"
//   - "finalized"
//   - "justified"
//   - <slot>
//   - <hex encoded block root with '0x' prefix>
//   - <block root>
//
// cases:
//   - no block, 404
//   - block exists, no commitment, 200 w/ empty list
//   - block exists, has commitments, inside retention period (greater of protocol- or user-specified) serve then w/ 200 unless we hit an error reading them.
//     we are technically not supposed to import a block to forkchoice unless we have the blobs, so the nuance here is if we can't find the file and we are inside the protocol-defined retention period, then it's actually a 500.
//   - block exists, has commitments, outside retention period (greater of protocol- or user-specified) - ie just like block exists, no commitment
func (p *BeaconDbBlocker) Blobs(ctx context.Context, id string, indices map[uint64]bool) ([]*blocks.VerifiedROBlob, *core.RpcError) {
	var root []byte
	switch id {
	case "genesis":
		return nil, &core.RpcError{Err: errors.New("blobs are not supported for Phase 0 fork"), Reason: core.BadRequest}
	case "head":
		var err error
		root, err = p.ChainInfoFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, &core.RpcError{Err: errors.Wrapf(err, "could not retrieve head root"), Reason: core.Internal}
		}
	case "finalized":
		fcp := p.ChainInfoFetcher.FinalizedCheckpt()
		if fcp == nil {
			return nil, &core.RpcError{Err: errors.New("received nil finalized checkpoint"), Reason: core.Internal}
		}
		root = fcp.Root
	case "justified":
		jcp := p.ChainInfoFetcher.CurrentJustifiedCheckpt()
		if jcp == nil {
			return nil, &core.RpcError{Err: errors.New("received nil justified checkpoint"), Reason: core.Internal}
		}
		root = jcp.Root
	default:
		if bytesutil.IsHex([]byte(id)) {
			var err error
			root, err = hexutil.Decode(id)
			if len(root) != fieldparams.RootLength {
				return nil, &core.RpcError{Err: fmt.Errorf("invalid block root of length %d", len(root)), Reason: core.BadRequest}
			}
			if err != nil {
				return nil, &core.RpcError{Err: NewBlockIdParseError(err), Reason: core.BadRequest}
			}
		} else {
			slot, err := strconv.ParseUint(id, 10, 64)
			if err != nil {
				return nil, &core.RpcError{Err: NewBlockIdParseError(err), Reason: core.BadRequest}
			}
			denebStart, err := slots.EpochStart(params.BeaconConfig().DenebForkEpoch)
			if err != nil {
				return nil, &core.RpcError{Err: errors.Wrap(err, "could not calculate Deneb start slot"), Reason: core.Internal}
			}
			if primitives.Slot(slot) < denebStart {
				return nil, &core.RpcError{Err: errors.New("blobs are not supported before Deneb fork"), Reason: core.BadRequest}
			}
			ok, roots, err := p.BeaconDB.BlockRootsBySlot(ctx, primitives.Slot(slot))
			if !ok {
				return nil, &core.RpcError{Err: fmt.Errorf("block not found: no block roots at slot %d", slot), Reason: core.NotFound}
			}
			if err != nil {
				return nil, &core.RpcError{Err: errors.Wrap(err, "failed to get block roots by slot"), Reason: core.Internal}
			}
			root = roots[0][:]
			if len(roots) == 1 {
				break
			}
			for _, blockRoot := range roots {
				canonical, err := p.ChainInfoFetcher.IsCanonical(ctx, blockRoot)
				if err != nil {
					return nil, &core.RpcError{Err: errors.Wrap(err, "could not determine if block root is canonical"), Reason: core.Internal}
				}
				if canonical {
					root = blockRoot[:]
					break
				}
			}
		}
	}
	if !p.BeaconDB.HasBlock(ctx, bytesutil.ToBytes32(root)) {
		return nil, &core.RpcError{Err: errors.New("block not found"), Reason: core.NotFound}
	}
	b, err := p.BeaconDB.Block(ctx, bytesutil.ToBytes32(root))
	if err != nil {
		return nil, &core.RpcError{Err: errors.Wrap(err, "failed to retrieve block from db"), Reason: core.Internal}
	}
	// if block is not in the retention window  return 200 w/ empty list
	if !p.BlobStorage.WithinRetentionPeriod(slots.ToEpoch(b.Block().Slot()), slots.ToEpoch(p.GenesisTimeFetcher.CurrentSlot())) {
		return make([]*blocks.VerifiedROBlob, 0), nil
	}
	commitments, err := b.Block().Body().BlobKzgCommitments()
	if err != nil {
		return nil, &core.RpcError{Err: errors.Wrap(err, "failed to retrieve kzg commitments from block"), Reason: core.Internal}
	}
	// if there are no commitments return 200 w/ empty list
	if len(commitments) == 0 {
		return make([]*blocks.VerifiedROBlob, 0), nil
	}

	// Get the slot of the block.
	blockSlot := b.Block().Slot()

	// Get the first peerDAS epoch.
	eip7594ForkEpoch := params.BeaconConfig().Eip7594ForkEpoch

	// Compute the first peerDAS slot.
	peerDASStartSlot := primitives.Slot(math.MaxUint64)
	if eip7594ForkEpoch != primitives.Epoch(math.MaxUint64) {
		peerDASStartSlot, err = slots.EpochStart(eip7594ForkEpoch)
		if err != nil {
			return nil, &core.RpcError{Err: errors.Wrap(err, "could not calculate peerDAS start slot"), Reason: core.Internal}
		}
	}

	// Is peerDAS enabled for this block?
	isPeerDASEnabledForBlock := blockSlot >= peerDASStartSlot

	if indices == nil {
		indices = make(map[uint64]bool)
	}

	if !isPeerDASEnabledForBlock {
		return p.blobsFromStoredBlobs(indices, root)
	}

	return p.blobsFromStoredDataColumns(indices, root)
}
