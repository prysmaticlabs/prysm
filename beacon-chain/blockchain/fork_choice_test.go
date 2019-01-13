package blockchain

import (
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/gogo/protobuf/proto"
	"testing"
	"time"
)

func TestLMDGhost_UpdatesHead(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	genesisValidatorRegistry := validators.InitialValidatorRegistry()
	deposits := make([]*pb.Deposit, len(genesisValidatorRegistry))
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey: genesisValidatorRegistry[i].Pubkey,
		}
		balance := genesisValidatorRegistry[i].Balance
		depositData, err := b.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
            t.Fatal(err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	genesisTime := uint64(params.BeaconConfig().GenesisTime.Unix())
	beaconState, err := state.InitialBeaconState(deposits, genesisTime, nil)
	if err != nil {
        t.Fatal(err)
	}

	// #nosec G104
	stateEnc, _ := proto.Marshal(beaconState)
	stateHash := hashutil.Hash(stateEnc)
	genesisBlock := b.NewGenesisBlock(stateHash[:])
	if err := db.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
}
