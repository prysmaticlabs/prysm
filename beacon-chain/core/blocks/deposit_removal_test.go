package blocks

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestAddDeposit(t *testing.T) {
	dep := &pb.Deposit{
		DepositData: []byte{'a', 'b', 'c'},
	}
	defer func() {
		depositsToRemove = make([]*pb.Deposit, 0)
	}()

	addToDepositRemovalList(false, dep)

	if len(depositsToRemove) == 1 {
		t.Fatal("Deposit was added to removal list despite not having removal enabled")
	}

	addToDepositRemovalList(true, dep)

	if len(depositsToRemove) != 1 {
		t.Fatal("Deposit was not added to removal list")
	}
}

func TestRemoveDeposits(t *testing.T) {
	defer func() {
		depositsToRemove = make([]*pb.Deposit, 0)
	}()
	dep1 := &pb.Deposit{
		DepositData: []byte{'a', 'b', 'c'},
	}

	dep2 := &pb.Deposit{
		DepositData: []byte{'d', 'e', 'f'},
	}
	dep3 := &pb.Deposit{
		DepositData: []byte{'g', 'h', 'i'},
	}

	addToDepositRemovalList(true, dep1)
	addToDepositRemovalList(true, dep2)
	addToDepositRemovalList(true, dep3)

	ClearFromDepositRemovalList(dep1)
	ClearFromDepositRemovalList(dep3)

	deposits := DepositsToRemove()

	if len(deposits) != 1 {
		t.Fatalf("length of deposits is not equal to 1 :%d", len(deposits))
	}

	if !proto.Equal(deposits[0], dep2) {
		t.Fatalf("The wrong deposit was removed: %s", proto.MarshalTextString(deposits[0]))
	}
}
