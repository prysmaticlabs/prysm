package blocks

import (
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var depositsToRemove = make([]*pb.Deposit, 0)

// DepositsToRemove returns the deposits that we have to remove.
func DepositsToRemove() []*pb.Deposit {
	deps := make([]*pb.Deposit, len(depositsToRemove))
	copy(deps, depositsToRemove)
	return deps
}

// ClearFromDepositRemovalList removes the deposit from the list.
func ClearFromDepositRemovalList(invalidDeposit *pb.Deposit) {
	for i, deposit := range depositsToRemove {
		if proto.Equal(deposit, invalidDeposit) {
			depositsToRemove = append(depositsToRemove[:i], depositsToRemove[i+1:]...)
		}
	}
}

func addToDepositRemovalList(removalEnabled bool, deposit *pb.Deposit) {
	if removalEnabled {
		depositsToRemove = append(depositsToRemove, deposit)
	}
}
