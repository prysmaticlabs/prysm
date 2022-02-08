package forkchoice

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
)

type ExecutionEngineMock struct {
}

func (m *ExecutionEngineMock) PreparePayload(ctx context.Context, forkchoiceState catalyst.ForkchoiceStateV1, payloadAttributes catalyst.PayloadAttributesV1) (string, error) {
	return "", nil
}
func (m *ExecutionEngineMock) GetPayload(ctx context.Context, payloadID string) (*catalyst.ExecutableDataV1, error) {
	return nil, nil
}
func (m *ExecutionEngineMock) NotifyForkChoiceValidated(ctx context.Context, forkchoiceState catalyst.ForkchoiceStateV1) error {
	return nil
}
func (m *ExecutionEngineMock) ExecutePayload(ctx context.Context, data *catalyst.ExecutableDataV1) ([]byte, error) {
	return nil, nil
}

func (m *ExecutionEngineMock) LatestExecutionBlock() (*powchain.ExecutionBlock, error) {
	return nil, nil
}
func (m *ExecutionEngineMock) ExecutionBlockByHash(blockHash common.Hash) (*powchain.ExecutionBlock, error) {
	return nil, nil
}
