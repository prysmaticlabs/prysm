package helpers

import (
	"math/big"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestCountVote_OK(t *testing.T) {
	vm := EmptyVoteHierarchyMap()
	ed := &pb.Eth1Data{DepositRoot: []byte{1}, DepositCount: 100, BlockHash: []byte{10}}
	vm, err := CountVote(vm, ed, big.NewInt(0))
	if err != nil {
		t.Fatal("fail to add deposit to map")
	}
	if !reflect.DeepEqual(ed, vm.BestVote) {
		t.Errorf(
			"Expected best vote to be %v, got %v",
			ed,
			vm.BestVote,
		)
	}

}

func TestCountVote_ByVoteCount(t *testing.T) {
	vm := EmptyVoteHierarchyMap()
	ed1 := &pb.Eth1Data{DepositRoot: []byte{1}, DepositCount: 100, BlockHash: []byte{10}}
	ed2 := &pb.Eth1Data{DepositRoot: []byte{1}, DepositCount: 101, BlockHash: []byte{10}}

	vm, err := CountVote(vm, ed1, big.NewInt(0))
	if err != nil {
		t.Fatal("fail to add deposit to map")
	}
	vm, err = CountVote(vm, ed2, big.NewInt(0))
	if err != nil {
		t.Fatal("fail to add deposit to map")
	}
	vm, err = CountVote(vm, ed2, big.NewInt(0))
	if err != nil {
		t.Fatal("fail to add deposit to map")
	}
	if !reflect.DeepEqual(ed2, vm.BestVote) {
		t.Errorf(
			"Expected best vote to be %v, got %v",
			ed2,
			vm.BestVote,
		)
	}
}

func TestCountVote_PreferVoteCountToHeight(t *testing.T) {
	vm := EmptyVoteHierarchyMap()
	ed1 := &pb.Eth1Data{DepositRoot: []byte{1}, DepositCount: 100, BlockHash: []byte{10}}
	ed2 := &pb.Eth1Data{DepositRoot: []byte{1}, DepositCount: 101, BlockHash: []byte{10}}

	vm, err := CountVote(vm, ed1, big.NewInt(10))
	if err != nil {
		t.Fatal("fail to add deposit to map")
	}
	vm, err = CountVote(vm, ed2, big.NewInt(0))
	if err != nil {
		t.Fatal("fail to add deposit to map")
	}
	vm, err = CountVote(vm, ed2, big.NewInt(0))
	if err != nil {
		t.Fatal("fail to add deposit to map")
	}
	if !reflect.DeepEqual(ed2, vm.BestVote) {
		t.Errorf(
			"Expected best vote to be %v, got %v",
			ed2,
			vm.BestVote,
		)
	}
}

func TestCountVote_BreakTiesByHeight(t *testing.T) {
	vm := EmptyVoteHierarchyMap()
	ed1 := &pb.Eth1Data{DepositRoot: []byte{1}, DepositCount: 100, BlockHash: []byte{10}}
	ed2 := &pb.Eth1Data{DepositRoot: []byte{1}, DepositCount: 101, BlockHash: []byte{10}}
	vm, err := CountVote(vm, ed1, big.NewInt(10))
	if err != nil {
		t.Fatal("fail to add deposit to map")
	}
	vm, err = CountVote(vm, ed2, big.NewInt(0))
	if err != nil {
		t.Fatal("fail to add deposit to map")
	}

	if !reflect.DeepEqual(ed1, vm.BestVote) {
		t.Errorf(
			"Expected best vote to be %v, got %v",
			ed1,
			vm.BestVote,
		)
	}
}
