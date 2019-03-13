package forkutil

import (
	"math"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestForkVersion_OK(t *testing.T) {
	fork := &pb.Fork{
		Epoch:           10,
		PreviousVersion: 2,
		CurrentVersion:  3,
	}

	if ForkVersion(fork, 9) != 2 {
		t.Errorf("fork Version not equal to 2 %d", ForkVersion(fork, 9))
	}

	if ForkVersion(fork, 11) != 3 {
		t.Errorf("fork Version not equal to 3 %d", ForkVersion(fork, 11))
	}
}

func TestDomainVersion_OK(t *testing.T) {
	fork := &pb.Fork{
		Epoch:           10,
		PreviousVersion: 2,
		CurrentVersion:  3,
	}

	constant := uint64(math.Pow(2, 32))

	if DomainVersion(fork, 9, 2) != 2*constant+2 {
		t.Errorf("incorrect domain version %d", DomainVersion(fork, 9, 2))
	}

	if DomainVersion(fork, 11, 3) != 3*constant+3 {
		t.Errorf("incorrect domain version %d", DomainVersion(fork, 11, 3))
	}
}
