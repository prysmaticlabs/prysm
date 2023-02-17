package testing

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	payloadattribute "github.com/prysmaticlabs/prysm/v3/consensus-types/payload-attribute"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// EngineClient --
type EngineClient struct {
	NewPayloadResp              []byte
	PayloadIDBytes              *pb.PayloadIDBytes
	ForkChoiceUpdatedResp       []byte
	ExecutionPayload            *pb.ExecutionPayload
	ExecutionPayloadCapella     *pb.ExecutionPayloadCapella
	ExecutionBlock              *pb.ExecutionBlock
	Err                         error
	ErrLatestExecBlock          error
	ErrExecBlockByHash          error
	ErrForkchoiceUpdated        error
	ErrNewPayload               error
	ErrGetPayload               error
	ExecutionPayloadByBlockHash map[[32]byte]*pb.ExecutionPayload
	BlockByHashMap              map[[32]byte]*pb.ExecutionBlock
	NumReconstructedPayloads    uint64
	TerminalBlockHash           []byte
	TerminalBlockHashExists     bool
	OverrideValidHash           [32]byte
	BlockValue                  *big.Int
}

// NewPayload --
func (e *EngineClient) NewPayload(_ context.Context, _ interfaces.ExecutionData) ([]byte, error) {
	return e.NewPayloadResp, e.ErrNewPayload
}

// ForkchoiceUpdated --
func (e *EngineClient) ForkchoiceUpdated(
	_ context.Context, fcs *pb.ForkchoiceState, _ payloadattribute.Attributer,
) (*pb.PayloadIDBytes, []byte, error) {
	if e.OverrideValidHash != [32]byte{} && bytesutil.ToBytes32(fcs.HeadBlockHash) == e.OverrideValidHash {
		return e.PayloadIDBytes, e.ForkChoiceUpdatedResp, nil
	}
	return e.PayloadIDBytes, e.ForkChoiceUpdatedResp, e.ErrForkchoiceUpdated
}

// GetPayload --
func (e *EngineClient) GetPayload(_ context.Context, _ [8]byte, s primitives.Slot) (interfaces.ExecutionData, error) {
	if slots.ToEpoch(s) >= params.BeaconConfig().CapellaForkEpoch {
		return blocks.WrappedExecutionPayloadCapella(e.ExecutionPayloadCapella, e.BlockValue)
	}
	p, err := blocks.WrappedExecutionPayload(e.ExecutionPayload)
	if err != nil {
		return nil, err
	}
	return p, e.ErrGetPayload
}

// ExchangeTransitionConfiguration --
func (e *EngineClient) ExchangeTransitionConfiguration(_ context.Context, _ *pb.TransitionConfiguration) error {
	return e.Err
}

// LatestExecutionBlock --
func (e *EngineClient) LatestExecutionBlock(_ context.Context) (*pb.ExecutionBlock, error) {
	return e.ExecutionBlock, e.ErrLatestExecBlock
}

// ExecutionBlockByHash --
func (e *EngineClient) ExecutionBlockByHash(_ context.Context, h common.Hash, _ bool) (*pb.ExecutionBlock, error) {
	b, ok := e.BlockByHashMap[h]
	if !ok {
		return nil, errors.New("block not found")
	}
	return b, e.ErrExecBlockByHash
}

// ReconstructFullBlock --
func (e *EngineClient) ReconstructFullBlock(
	_ context.Context, blindedBlock interfaces.ReadOnlySignedBeaconBlock,
) (interfaces.SignedBeaconBlock, error) {
	if !blindedBlock.Block().IsBlinded() {
		return nil, errors.New("block must be blinded")
	}
	header, err := blindedBlock.Block().Body().Execution()
	if err != nil {
		return nil, err
	}
	payload, ok := e.ExecutionPayloadByBlockHash[bytesutil.ToBytes32(header.BlockHash())]
	if !ok {
		return nil, errors.New("block not found")
	}
	e.NumReconstructedPayloads++
	return blocks.BuildSignedBeaconBlockFromExecutionPayload(blindedBlock, payload)
}

// ReconstructFullBellatrixBlockBatch --
func (e *EngineClient) ReconstructFullBellatrixBlockBatch(
	ctx context.Context, blindedBlocks []interfaces.ReadOnlySignedBeaconBlock,
) ([]interfaces.SignedBeaconBlock, error) {
	fullBlocks := make([]interfaces.SignedBeaconBlock, 0, len(blindedBlocks))
	for _, b := range blindedBlocks {
		newBlock, err := e.ReconstructFullBlock(ctx, b)
		if err != nil {
			return nil, err
		}
		fullBlocks = append(fullBlocks, newBlock)
	}
	return fullBlocks, nil
}

// GetTerminalBlockHash --
func (e *EngineClient) GetTerminalBlockHash(ctx context.Context, transitionTime uint64) ([]byte, bool, error) {
	ttd := new(big.Int)
	ttd.SetString(params.BeaconConfig().TerminalTotalDifficulty, 10)
	terminalTotalDifficulty, overflows := uint256.FromBig(ttd)
	if overflows {
		return nil, false, errors.New("could not convert terminal total difficulty to uint256")
	}
	blk, err := e.LatestExecutionBlock(ctx)
	if err != nil {
		return nil, false, errors.Wrap(err, "could not get latest execution block")
	}
	if blk == nil {
		return nil, false, errors.New("latest execution block is nil")
	}

	for {
		b, err := hexutil.DecodeBig(blk.TotalDifficulty)
		if err != nil {
			return nil, false, errors.Wrap(err, "could not convert total difficulty to uint256")
		}
		currentTotalDifficulty, _ := uint256.FromBig(b)
		blockReachedTTD := currentTotalDifficulty.Cmp(terminalTotalDifficulty) >= 0

		parentHash := blk.ParentHash
		if parentHash == params.BeaconConfig().ZeroHash {
			return nil, false, nil
		}
		parentBlk, err := e.ExecutionBlockByHash(ctx, parentHash, false /* with txs */)
		if err != nil {
			return nil, false, errors.Wrap(err, "could not get parent execution block")
		}
		if blockReachedTTD {
			b, err := hexutil.DecodeBig(parentBlk.TotalDifficulty)
			if err != nil {
				return nil, false, errors.Wrap(err, "could not convert total difficulty to uint256")
			}
			parentTotalDifficulty, _ := uint256.FromBig(b)
			parentReachedTTD := parentTotalDifficulty.Cmp(terminalTotalDifficulty) >= 0
			if blk.Time >= transitionTime {
				return nil, false, nil
			}
			if !parentReachedTTD {
				return blk.Hash[:], true, nil
			}
		} else {
			return nil, false, nil
		}
		blk = parentBlk
	}
}
