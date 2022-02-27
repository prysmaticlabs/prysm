package forkchoice

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	pb "github.com/prysmaticlabs/prysm/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type ExecutionEngineMock struct {
	powBlocks map[[32]byte]*eth.PowBlock
}

func (m *ExecutionEngineMock) GetPayload(context.Context, [8]byte) (*pb.ExecutionPayload, error) {
	return nil, nil
}
func (m *ExecutionEngineMock) ForkchoiceUpdated(context.Context, *pb.ForkchoiceState, *pb.PayloadAttributes) (*pb.PayloadIDBytes, []byte, error) {
	return nil, nil, nil
}
func (m *ExecutionEngineMock) NewPayload(context.Context, *pb.ExecutionPayload) ([]byte, error) {
	return nil, nil
}

func (m *ExecutionEngineMock) LatestExecutionBlock(context.Context) (*pb.ExecutionBlock, error) {
	return nil, nil
}

func (m *ExecutionEngineMock) ExchangeTransitionConfiguration(context.Context, *pb.TransitionConfiguration) (*pb.TransitionConfiguration, error) {
	return nil, nil
}

func (m *ExecutionEngineMock) ExecutionBlockByHash(ctx context.Context, hash common.Hash) (*pb.ExecutionBlock, error) {
	b, ok := m.powBlocks[bytesutil.ToBytes32(hash.Bytes())]
	if !ok {
		return nil, nil
	}
	tdInBigEndian := bytesutil.ReverseByteOrder(b.TotalDifficulty)
	tdBigint := new(big.Int)
	tdBigint.SetBytes(tdInBigEndian)
	td256, of := uint256.FromBig(tdBigint)
	if of {
		return nil, errors.New("could not convert big.Int to uint256")
	}

	return &pb.ExecutionBlock{
		ParentHash:      b.ParentHash,
		TotalDifficulty: td256.String(),
		Hash:            b.BlockHash,
	}, nil
}
