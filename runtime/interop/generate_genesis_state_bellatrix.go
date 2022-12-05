// Package interop contains deterministic utilities for generating
// genesis states and keys.
package interop

import (
	"context"
	"encoding/hex"

	"github.com/pkg/errors"
	coreState "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	statenative "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time"
)

// GenerateGenesisStateBellatrix deterministically given a genesis time and number of validators.
// If a genesis time of 0 is supplied it is set to the current time.
func GenerateGenesisStateBellatrix(ctx context.Context, genesisTime, numValidators uint64) (*ethpb.BeaconStateBellatrix, []*ethpb.Deposit, error) {
	privKeys, pubKeys, err := DeterministicallyGenerateKeys(0 /*startIndex*/, numValidators)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not deterministically generate keys for %d validators", numValidators)
	}
	depositDataItems, depositDataRoots, err := DepositDataFromKeys(privKeys, pubKeys)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not generate deposit data from keys")
	}
	return GenerateGenesisStateBellatrixFromDepositData(ctx, genesisTime, depositDataItems, depositDataRoots)
}

// GenerateGenesisStateBellatrixFromDepositData creates a genesis state given a list of
// deposit data items and their corresponding roots.
func GenerateGenesisStateBellatrixFromDepositData(
	ctx context.Context, genesisTime uint64, depositData []*ethpb.Deposit_Data, depositDataRoots [][]byte,
) (*ethpb.BeaconStateBellatrix, []*ethpb.Deposit, error) {
	t, err := trie.GenerateTrieFromItems(depositDataRoots, params.BeaconConfig().DepositContractTreeDepth)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not generate Merkle trie for deposit proofs")
	}
	deposits, err := GenerateDepositsFromData(depositData, t)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not generate deposits from the deposit data provided")
	}
	root, err := t.HashTreeRoot()
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not hash tree root of deposit trie")
	}
	if genesisTime == 0 {
		genesisTime = uint64(time.Now().Unix())
	}
	beaconState, err := coreState.GenesisBeaconStateBellatrix(ctx, deposits, genesisTime, &ethpb.Eth1Data{
		DepositRoot:  root[:],
		DepositCount: uint64(len(deposits)),
		BlockHash:    mockEth1BlockHash,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not generate genesis state")
	}
	rawSt, ok := beaconState.ToProtoUnsafe().(*ethpb.BeaconStateBellatrix)
	if !ok {
		return nil, nil, errors.New("state is of invalid type")
	}
	stateRoot, err := hex.DecodeString("5c9ba34f1167c7f2ccb56623667e04f2d3e769148d181f4a51484e14b3ced910")
	if err != nil {
		return nil, nil, err
	}
	receiptsRoot, err := hex.DecodeString("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	if err != nil {
		return nil, nil, err
	}
	blkHash, err := hex.DecodeString("837b03b21e5c2bc5484342b5082848748046071c5882624022b46f20ff0b46db")
	if err != nil {
		return nil, nil, err
	}
	txRoot, err := hex.DecodeString("7ffe241ea60187fdb0187bfa22de35d1f9bed7ab061d9401fd47e34a54fbede1")
	if err != nil {
		return nil, nil, err
	}
	baseFee := make([]byte, 32)
	baseFee[1] = 202
	baseFee[2] = 154
	baseFee[3] = 59
	rawSt.LatestExecutionPayloadHeader = &v1.ExecutionPayloadHeader{
		ParentHash:       make([]byte, 32),
		FeeRecipient:     make([]byte, 20),
		StateRoot:        stateRoot,
		ReceiptsRoot:     receiptsRoot,
		LogsBloom:        make([]byte, 256),
		PrevRandao:       make([]byte, 32),
		BlockNumber:      0,
		GasLimit:         4000000,
		GasUsed:          0,
		Timestamp:        rawSt.GenesisTime,
		BaseFeePerGas:    baseFee,
		BlockHash:        blkHash,
		TransactionsRoot: txRoot,
	}

	pbState, err := statenative.ProtobufBeaconStateBellatrix(rawSt)
	if err != nil {
		return nil, nil, err
	}
	return pbState, deposits, nil
}
