package blockchain

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockClient struct{}

func (f *mockClient) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (f *mockClient) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	return nil, nil
}

func (f *mockClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (f *mockClient) LatestBlockHash() common.Hash {
	return common.BytesToHash([]byte{'A'})
}

func TestDefaultConfig(t *testing.T) {
	if DefaultConfig().BeaconBlockBuf != 10 {
		t.Errorf("Default block buffer should be 10, got: %v", DefaultConfig().BeaconBlockBuf)
	}
}

func TestStartStop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	tmp := fmt.Sprintf("%s/beacontest", os.TempDir())
	defer os.RemoveAll(tmp)

	config := &database.DBConfig{DataDir: tmp, Name: "beacontestdata", InMemory: false}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	cfg := &Config{
		BeaconBlockBuf: 0,
	}

	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}

	chainService, err := NewChainService(ctx, cfg, beaconChain, db, web3Service)
	if err != nil {
		t.Fatalf("unable to setup chain service: %v", err)
	}

	chainService.Start()

	if len(chainService.ProcessedBlockHashes()) != 0 {
		t.Errorf("incorrect processedBlockHashes size")
	}
	if len(chainService.ProcessedCrystallizedStateHashes()) != 0 {
		t.Errorf("incorrect processedCrystallizedStateHashes size")
	}
	if len(chainService.ProcessedActiveStateHashes()) != 0 {
		t.Errorf("incorrect processedActiveStateHashes size")
	}
	if len(chainService.CurrentActiveState().RecentBlockHashes()) != 0 {
		t.Errorf("incorrect recent block hashes")
	}
	if len(chainService.CurrentCrystallizedState().Validators()) != 0 {
		t.Errorf("incorrect default validator size")
	}
	if chainService.ContainsBlock([32]byte{}) {
		t.Errorf("chain is not empty")
	}
	if chainService.ContainsCrystallizedState([32]byte{}) {
		t.Errorf("cyrstallized states is not empty")
	}
	if chainService.ContainsActiveState([32]byte{}) {
		t.Errorf("active states is not empty")
	}
	hasState, err := chainService.HasStoredState()
	if err != nil {
		t.Fatalf("calling HasStoredState failed")
	}
	if hasState {
		t.Errorf("has stored state should return false")
	}
	chainService, _ = NewChainService(ctx, cfg, beaconChain, db, web3Service)

	active := types.NewActiveState(&pb.ActiveState{RecentBlockHashes: [][]byte{{'A'}}})
	activeStateHash, err := active.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}
	chainService.chain.SetActiveState(active)

	crystallized := types.NewCrystallizedState(&pb.CrystallizedState{LastStateRecalc: 10000})
	crystallizedStateHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Cannot hash crystallized state: %v", err)
	}
	chainService.chain.SetCrystallizedState(crystallized)

	parentBlock := NewBlock(t, nil)
	parentHash, _ := parentBlock.Hash()

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            2,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		ParentHash:            parentHash[:],
		PowChainRef:           []byte("a"),
	})
	if err := chainService.SaveBlock(block); err != nil {
		t.Errorf("save block should have failed")
	}

	// Save states so HasStoredState state should return true.
	chainService.chain.SetActiveState(types.NewActiveState(&pb.ActiveState{}))
	chainService.chain.SetCrystallizedState(types.NewCrystallizedState(&pb.CrystallizedState{}))
	hasState, _ = chainService.HasStoredState()
	if !hasState {
		t.Errorf("has stored state should return false")
	}

	if err := chainService.Stop(); err != nil {
		t.Fatalf("unable to stop chain service: %v", err)
	}

	// The context should have been canceled.
	if chainService.ctx.Err() == nil {
		t.Error("context was not canceled")
	}
	hook.Reset()
}

func TestFaultyStop(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	tmp := fmt.Sprintf("%s/beacontest", os.TempDir())
	defer os.RemoveAll(tmp)

	config := &database.DBConfig{DataDir: tmp, Name: "beacontestdata", InMemory: false}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	cfg := &Config{
		BeaconBlockBuf: 0,
	}

	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}

	chainService, err := NewChainService(ctx, cfg, beaconChain, db, web3Service)
	if err != nil {
		t.Fatalf("unable to setup chain service: %v", err)
	}

	chainService.Start()

	chainService.chain.SetActiveState(types.NewActiveState(nil))
	err = chainService.Stop()
	if err == nil {
		t.Errorf("chain stop should have failed with persist active state")
	}

	chainService.chain.SetActiveState(types.NewActiveState(&pb.ActiveState{}))
	chainService.chain.SetCrystallizedState(types.NewCrystallizedState(nil))
	err = chainService.Stop()
	if err == nil {
		t.Errorf("chain stop should have failed with persist crystallized state")
	}
	hook.Reset()
}

func TestProcessingStates(t *testing.T) {
	ctx := context.Background()
	tmp := fmt.Sprintf("%s/beacontest", os.TempDir())
	defer os.RemoveAll(tmp)

	config := &database.DBConfig{DataDir: tmp, Name: "beacontestdata", InMemory: false}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	cfg := &Config{
		BeaconBlockBuf: 0,
	}
	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}
	chainService, _ := NewChainService(ctx, cfg, beaconChain, db, web3Service)

	if err := chainService.ProcessCrystallizedState(types.NewCrystallizedState(nil)); err == nil {
		t.Errorf("processing crystallized state should have failed")
	}

	if err := chainService.ProcessActiveState(types.NewActiveState(nil)); err == nil {
		t.Errorf("processing active state should have failed")
	}

	chainService.ProcessCrystallizedState(types.NewCrystallizedState(&pb.CrystallizedState{}))
	chainService.ProcessActiveState(types.NewActiveState(&pb.ActiveState{}))
}

func TestProcessingBadBlock(t *testing.T) {
	ctx := context.Background()
	tmp := fmt.Sprintf("%s/beacontest", os.TempDir())
	defer os.RemoveAll(tmp)

	config := &database.DBConfig{DataDir: tmp, Name: "beacontestdata", InMemory: false}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	cfg := &Config{
		BeaconBlockBuf: 0,
	}
	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}
	chainService, _ := NewChainService(ctx, cfg, beaconChain, db, web3Service)

	active := types.NewActiveState(&pb.ActiveState{RecentBlockHashes: [][]byte{{'A'}}})
	activeStateHash, err := active.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}
	chainService.chain.SetActiveState(active)

	crystallized := types.NewCrystallizedState(&pb.CrystallizedState{LastStateRecalc: 10000})
	crystallizedStateHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Cannot hash crystallized state: %v", err)
	}
	chainService.chain.SetCrystallizedState(crystallized)

	parentBlock := NewBlock(t, nil)
	parentHash, _ := parentBlock.Hash()

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            2,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		ParentHash:            parentHash[:],
		PowChainRef:           []byte("a"),
	})

	if err = chainService.ProcessBlock(block); err == nil {
		t.Error("process block should have failed with parent hash points to nil")
	}
}

func TestRunningChainService(t *testing.T) {
	ctx := context.Background()
	tmp := fmt.Sprintf("%s/beacontest", os.TempDir())
	defer os.RemoveAll(tmp)
	config := &database.DBConfig{DataDir: tmp, Name: "beacontestdata", InMemory: false}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	cfg := &Config{
		BeaconBlockBuf: 0,
	}
	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}
	testAttesterBitfield := []byte{200, 148, 146, 179, 49}
	state := types.NewActiveState(&pb.ActiveState{PendingAttestations: []*pb.AttestationRecord{{AttesterBitfield: testAttesterBitfield}}})
	if err := beaconChain.SetActiveState(state); err != nil {
		t.Fatalf("unable to Mutate Active state: %v", err)
	}

	chainService, _ := NewChainService(ctx, cfg, beaconChain, db, web3Service)

	exitRoutine := make(chan bool)
	go func() {
		chainService.run(chainService.ctx.Done())
		<-exitRoutine
	}()

	parentBlock := NewBlock(t, nil)
	parentHash, _ := parentBlock.Hash()

	activeStateHash, err := state.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}

	var validators []*pb.ValidatorRecord
	for i := 0; i < 40; i++ {
		validator := &pb.ValidatorRecord{Balance: 32, StartDynasty: 1, EndDynasty: 10}
		validators = append(validators, validator)
	}

	crystallized := types.NewCrystallizedState(&pb.CrystallizedState{Validators: validators, CurrentDynasty: 5})
	crystallizedStateHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Cannot hash crystallized state: %v", err)
	}
	chainService.chain.SetCrystallizedState(crystallized)

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            65,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		ParentHash:            parentHash[:],
		PowChainRef:           []byte("a"),
	})

	chainService.latestBeaconBlock <- block
	chainService.cancel()
	exitRoutine <- true
}
