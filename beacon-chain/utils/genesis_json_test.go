package utils

import (
	"os"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/prysmaticlabs/prysm/shared/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestInitGenesisJsonFailure(t *testing.T) {
	fname := "/genesis.json"
	pwd, _ := os.Getwd()
	fnamePath := pwd + fname

	_, err := InitialValidatorsFromJSON(fnamePath)
	if err == nil {
		t.Fatalf("genesis.json should have failed %v", err)
	}
}

func TestInitGenesisJson(t *testing.T) {
	fname := "/genesis.json"
	pwd, _ := os.Getwd()
	fnamePath := pwd + fname
	os.Remove(fnamePath)

	oc := params.SetDemoBeaconConfig()
	
	cStateJSON := &pb.CrystallizedState{
		LastStateRecalculationSlot: 0,
		JustifiedStreak:            1,
		LastFinalizedSlot:          99,
		Validators: []*pb.ValidatorRecord{
			{Pubkey: []byte{}, Balance: 32, Status: uint64(params.Active)},
		},
	}
	os.Create(fnamePath)
	f, err := os.OpenFile(fnamePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		t.Fatalf("can't open file %v", err)
	}

	ma := jsonpb.Marshaler{}
	err = ma.Marshal(f, cStateJSON)
	if err != nil {
		t.Fatalf("can't marshal file %v", err)
	}

	validators, err := InitialValidatorsFromJSON(fnamePath)
	if err != nil {
		t.Fatalf("genesis.json failed %v", err)
	}

	if validators[0].Status != 1 {
		t.Errorf("Failed to load of genesis json")
	}
	os.Remove(fnamePath)
}
