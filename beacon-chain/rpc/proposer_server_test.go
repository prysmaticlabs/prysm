package rpc

import (
	"testing"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/shared/params"
	"context"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	ptypes"github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"time"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestProposeBlock(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	mockChain := &mockChainService{}

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	deposits := make([]*pbp2p.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		depositData, err := b.EncodeDepositData(
			&pbp2p.DepositInput{
				Pubkey: []byte(strconv.Itoa(i)),
				RandaoCommitmentHash32: []byte{41, 13, 236, 217, 84, 139, 98, 168, 214, 3, 69,
					169, 136, 56, 111, 200, 75, 166, 188, 149, 72, 64, 8, 246, 54, 47, 147, 22, 14, 243, 229, 99},
			},
			params.BeaconConfig().MaxDepositInGwei,
			time.Now().Unix(),
		)
		if err != nil {
			t.Fatalf("Could not encode deposit input: %v", err)
		}
		deposits[i] = &pbp2p.Deposit{
			DepositData: depositData,
		}
	}

	beaconState, err := state.InitialBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Could not instantiate initial state: %v", err)
	}

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	rpcService := NewRPCService(context.Background(), &Config{
		Port:            "6372",
		ChainService:    mockChain,
		BeaconDB:        db,
		POWChainService: &mockPOWChainService{},
	})
	req := &pbp2p.BeaconBlock{
		Slot:             5,
		ParentRootHash32: []byte("parent-hash"),
	}
	if _, err := rpcService.ProposeBlock(context.Background(), req); err != nil {
		t.Errorf("Could not propose block correctly: %v", err)
	}
}

func TestLatestPOWChainBlockHash(t *testing.T) {
	mockChain := &mockChainService{}
	mockPOWChain := &mockPOWChainService{}
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	rpcService := NewRPCService(context.Background(), &Config{
		Port:            "6372",
		ChainService:    mockChain,
		BeaconDB:        db,
		POWChainService: mockPOWChain,
	})

	rpcService.enablePOWChain = false

	res, err := rpcService.LatestPOWChainBlockHash(context.Background(), nil)
	if err != nil {
		t.Fatalf("Could not get latest POW Chain block hash %v", err)
	}

	expectedHash := common.BytesToHash([]byte{'p', 'o', 'w', 'c', 'h', 'a', 'i', 'n'})
	if !bytes.Equal(res.BlockHash, expectedHash[:]) {
		t.Errorf("pow chain hash received is not the expected hash")
	}

}

func TestComputeStateRoot(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	mockChain := &mockChainService{}

	genesis := b.NewGenesisBlock([]byte{})
	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	deposits := make([]*pbp2p.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		depositData, err := b.EncodeDepositData(
			&pbp2p.DepositInput{
				Pubkey: []byte(strconv.Itoa(i)),
				RandaoCommitmentHash32: []byte{41, 13, 236, 217, 84, 139, 98, 168, 214, 3, 69,
					169, 136, 56, 111, 200, 75, 166, 188, 149, 72, 64, 8, 246, 54, 47, 147, 22, 14, 243, 229, 99},
			},
			params.BeaconConfig().MaxDepositInGwei,
			time.Now().Unix(),
		)
		if err != nil {
			t.Fatalf("Could not encode deposit input: %v", err)
		}
		deposits[i] = &pbp2p.Deposit{
			DepositData: depositData,
		}
	}

	beaconState, err := state.InitialBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Could not instantiate initial state: %v", err)
	}

	beaconState.Slot = 10

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	rpcService := NewRPCService(context.Background(), &Config{
		Port:            "6372",
		ChainService:    mockChain,
		BeaconDB:        db,
		POWChainService: &mockPOWChainService{},
	})

	req := &pbp2p.BeaconBlock{
		ParentRootHash32:   nil,
		Slot:               11,
		RandaoRevealHash32: nil,
		Body: &pbp2p.BeaconBlockBody{
			ProposerSlashings: nil,
			CasperSlashings:   nil,
		},
	}

	_, _ = rpcService.ComputeStateRoot(context.Background(), req)
}

func TestProposerIndex(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	mockChain := &mockChainService{}
	mockPOWChain := &mockPOWChainService{}
	genesis := b.NewGenesisBlock([]byte{})

	if err := db.SaveBlock(genesis); err != nil {
		t.Fatalf("Could not save genesis block: %v", err)
	}

	deposits := make([]*pbp2p.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		depositData, err := b.EncodeDepositData(
			&pbp2p.DepositInput{
				Pubkey: []byte(strconv.Itoa(i)),
				RandaoCommitmentHash32: []byte{41, 13, 236, 217, 84, 139, 98, 168, 214, 3, 69,
					169, 136, 56, 111, 200, 75, 166, 188, 149, 72, 64, 8, 246, 54, 47, 147, 22, 14, 243, 229, 99},
			},
			params.BeaconConfig().MaxDepositInGwei,
			time.Now().Unix(),
		)
		if err != nil {
			t.Fatalf("Could not encode deposit input: %v", err)
		}
		deposits[i] = &pbp2p.Deposit{
			DepositData: depositData,
		}
	}

	beaconState, err := state.InitialBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Could not instantiate initial state: %v", err)
	}

	beaconState.Slot = 10

	if err := db.UpdateChainHead(genesis, beaconState); err != nil {
		t.Fatalf("Could not save genesis state: %v", err)
	}

	rpcService := NewRPCService(context.Background(), &Config{
		Port:            "6372",
		ChainService:    mockChain,
		BeaconDB:        db,
		POWChainService: mockPOWChain,
	})

	expectedIndex := 1

	req := &pb.ProposerIndexRequest{
		SlotNumber: 1,
	}

	res, err := rpcService.ProposerIndex(context.Background(), req)
	if err != nil {
		t.Fatalf("Could not get proposer index %v", err)
	}

	if res.Index != uint32(expectedIndex) {
		t.Errorf("Expected index of %d but got %d", expectedIndex, res.Index)
	}
}
