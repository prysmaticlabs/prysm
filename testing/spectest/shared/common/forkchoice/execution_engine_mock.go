package forkchoice

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

type ExecutionEngineMock struct {
	powBlocks map[[32]byte]*eth.PowBlock
}

func (m *ExecutionEngineMock) PreparePayload(context.Context, catalyst.ForkchoiceStateV1, catalyst.PayloadAttributesV1) (string, error) {
	return "", nil
}
func (m *ExecutionEngineMock) GetPayload(context.Context, string) (*catalyst.ExecutableDataV1, error) {
	return nil, nil
}
func (m *ExecutionEngineMock) NotifyForkChoiceValidated(context.Context, catalyst.ForkchoiceStateV1) error {
	return nil
}
func (m *ExecutionEngineMock) ExecutePayload(context.Context, *catalyst.ExecutableDataV1) ([]byte, error) {
	return nil, nil
}

func (m *ExecutionEngineMock) LatestExecutionBlock() (*powchain.ExecutionBlock, error) {
	return nil, nil
}

func (m *ExecutionEngineMock) ExecutionBlockByHash(blockHash common.Hash) (*powchain.ExecutionBlock, error) {
	b, ok := m.powBlocks[bytesutil.ToBytes32(blockHash.Bytes())]
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

	return &powchain.ExecutionBlock{
		ParentHash:      common.Bytes2Hex(b.ParentHash),
		TotalDifficulty: td256.String(),
		Hash:            common.Bytes2Hex(b.BlockHash),
	}, nil
}
