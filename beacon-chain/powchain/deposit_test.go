package powchain

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

func TestProcessDeposit_OK(t *testing.T) {
	web3Service, err := NewWeb3Service(context.Background(), &Web3ServiceConfig{
		Endpoint:     endpoint,
		Reader:       &goodReader{},
		Logger:       &goodLogger{},
		HTTPLogger:   &goodLogger{},
		BeaconDB:     &db.BeaconDB{},
		BlockFetcher: &goodFetcher{},
	})
	if err != nil {
		t.Fatalf("unable to setup web3 ETH1.0 chain service: %v", err)
	}

	deposits, _ := testutil.SetupInitialDeposits(t, 1, true)

	leaf, err := hashutil.DepositHash(deposits[0].Data)
	if err != nil {
		t.Fatalf("Could not hash deposit %v", err)
	}

	trie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		log.Error(err)
	}

	root := trie.Root()

	eth1Data := &pb.Eth1Data{
		DepositCount: 1,
		DepositRoot:  root[:],
	}

	if err := web3Service.processDeposit(eth1Data, deposits[0]); err != nil {
		t.Fatalf("could not process deposit %v", err)
	}

	if web3Service.activeValidatorCount != 1 {
		t.Errorf("Did not get correct active validator count received %d, but wanted %d", web3Service.activeValidatorCount, 1)
	}
}
