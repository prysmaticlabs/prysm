// Package backend contains utilities for simulating an entire
// ETH 2.0 beacon chain for e2e tests and benchmarking
// purposes.
package backend

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// SimulatedBackend allowing for a programmatic advancement
// of an in-memory beacon chain for client test runs
// and other e2e use cases.
type SimulatedBackend struct {
	chainService *blockchain.ChainService
	db           *db.BeaconDB
}

// NewSimulatedBackend creates an instance by initializing a chain service
// utilizing a mockDB which will act according to test run parameters specified
// in the common ETH 2.0 client test YAML format.
func NewSimulatedBackend() (*SimulatedBackend, error) {
	db, err := setupDB()
	if err != nil {
		return nil, fmt.Errorf("could not setup simulated backend db: %v", err)
	}
	cs, err := blockchain.NewChainService(context.Background(), &blockchain.Config{
		BeaconDB:         db,
		IncomingBlockBuf: 0,
		EnablePOWChain:   false,
	})
	if err != nil {
		return nil, err
	}
	return &SimulatedBackend{
		chainService: cs,
		db:           db,
	}, nil
}

// RunChainTest uses a parsed set of chaintests from a YAML file
// according to the ETH 2.0 client chain test specification and runs them
// against the simulated backend.
func (sb *SimulatedBackend) RunChainTest(testCase *ChainTestCase) error {
	defer teardownDB(sb.db)
	// Utilize the config parameters in the test case to setup
	// the DB and set global config parameters accordingly.
	// Config parameters include: ValidatorCount, ShardCount,
	// CycleLength, MinCommitteeSize, and more based on the YAML
	// test language specification.
	c := params.BeaconConfig()
	c.ShardCount = testCase.Config.ShardCount
	c.CycleLength = testCase.Config.CycleLength
	c.MinCommitteeSize = testCase.Config.MinCommitteeSize
	params.OverrideBeaconConfig(c)

	// Then, we create the validators based on the custom test config.
	randaoPreCommit := [32]byte{}
	randaoReveal := hashutil.Hash(randaoPreCommit[:])
	validators := make([]*pb.ValidatorRecord, testCase.Config.ValidatorCount)
	for i := uint64(0); i < testCase.Config.ValidatorCount; i++ {
		validators[i] = &pb.ValidatorRecord{
			Status:            uint64(params.Active),
			Balance:           c.DepositSize * c.Gwei,
			WithdrawalAddress: []byte{},
			Pubkey:            []byte{},
			RandaoCommitment:  randaoReveal[:],
		}
	}
	// TODO(#718): Next step is to update and save the blocks specified
	// in the case case into the DB.
	//
	// Then, we call the updateHead routine and confirm the
	// chain's head is the expected result from the test case.
	return nil
}

// TODO(implement this for real)
// RunChainTest uses a parsed set of chaintests from a YAML file
// according to the ETH 2.0 client chain test specification and runs them
// against the simulated backend.
func (sb *SimulatedBackend) RunShuffleTest(testCase *ShuffleTestCase) error {
	defer teardownDB(sb.db)
	// Utilize the config parameters in the test case to setup
	// the DB and set global config parameters accordingly.
	// Config parameters include: ValidatorCount, ShardCount,
	// CycleLength, MinCommitteeSize, and more based on the YAML
	// test language specification.
	currentConfig := params.GetConfig()
	currentConfig.ShardCount = testCase.Config.ShardCount
	currentConfig.CycleLength = testCase.Config.CycleLength
	currentConfig.MinCommitteeSize = testCase.Config.MinCommitteeSize
	params.SetCustomConfig(currentConfig)

	// Then, we create the validators based on the custom test config.
	randaoPreCommit := [32]byte{}
	randaoReveal := hashutil.Hash(randaoPreCommit[:])
	validators := make([]*pb.ValidatorRecord, testCase.Config.ValidatorCount)
	for i := uint64(0); i < testCase.Config.ValidatorCount; i++ {
		validators[i] = &pb.ValidatorRecord{
			Status:            uint64(params.Active),
			Balance:           currentConfig.DepositSize * currentConfig.Gwei,
			WithdrawalAddress: []byte{},
			Pubkey:            []byte{},
			RandaoCommitment:  randaoReveal[:],
		}
	}
	// TODO(#718): Next step is to update and save the blocks specified
	// in the case case into the DB.
	//
	// Then, we call the updateHead routine and confirm the
	// chain's head is the expected result from the test case.
	return nil
}
