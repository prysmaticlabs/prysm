package components

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	state_native "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/interop"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	e2e "github.com/prysmaticlabs/prysm/v3/testing/endtoend/params"
	e2etypes "github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

func generateGenesis(ctx context.Context) (state.BeaconState, error) {
	if e2e.TestParams.Eth1GenesisBlock == nil {
		return nil, errors.New("Cannot construct bellatrix block, e2e.TestParams.Eth1GenesisBlock == nil")
	}
	gb := e2e.TestParams.Eth1GenesisBlock
	// so the DepositRoot in the BeaconState should be set to the HTR of an empty deposit trie.
	t, err := trie.NewTrie(params.BeaconConfig().DepositContractTreeDepth)
	if err != nil {
		return nil, err
	}
	dr, err := t.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	e1d := &ethpb.Eth1Data{
		DepositRoot:  dr[:],
		DepositCount: 0,
		BlockHash:    gb.Hash().Bytes(),
	}
	v := e2etypes.GenesisFork()
	switch v {
	case version.Bellatrix:
		return generateGenesisBellatrix(ctx, gb, e1d)
	case version.Phase0:
		return generateGenesisPhase0(ctx, e1d)
	default:
		return nil, fmt.Errorf("unsupported genesis fork version %s", version.String(v))
	}
}

func generateGenesisPhase0(ctx context.Context, e1d *ethpb.Eth1Data) (state.BeaconState, error) {
	g, _, err := interop.GeneratePreminedGenesisState(ctx, e2e.TestParams.CLGenesisTime, params.BeaconConfig().MinGenesisActiveValidatorCount, e1d)
	if err != nil {
		return nil, err
	}
	return state_native.InitializeFromProtoUnsafePhase0(g)
}

func generateGenesisBellatrix(ctx context.Context, gb *types.Block, e1d *ethpb.Eth1Data) (state.BeaconState, error) {
	payload := &enginev1.ExecutionPayload{
		ParentHash:    gb.ParentHash().Bytes(),
		FeeRecipient:  gb.Coinbase().Bytes(),
		StateRoot:     gb.Root().Bytes(),
		ReceiptsRoot:  gb.ReceiptHash().Bytes(),
		LogsBloom:     gb.Bloom().Bytes(),
		PrevRandao:    params.BeaconConfig().ZeroHash[:],
		BlockNumber:   gb.NumberU64(),
		GasLimit:      gb.GasLimit(),
		GasUsed:       gb.GasUsed(),
		Timestamp:     gb.Time(),
		ExtraData:     gb.Extra()[:32],
		BaseFeePerGas: bytesutil.PadTo(bytesutil.ReverseByteOrder(gb.BaseFee().Bytes()), fieldparams.RootLength),
		BlockHash:     gb.Hash().Bytes(),
		Transactions:  make([][]byte, 0),
	}
	g, _, err := interop.GenerateGenesisStateBellatrix(ctx, e2e.TestParams.CLGenesisTime, params.BeaconConfig().MinGenesisActiveValidatorCount, payload, e1d)
	if err != nil {
		return nil, err
	}
	return state_native.InitializeFromProtoUnsafeBellatrix(g)
}
