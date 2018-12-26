package misc

import (
	"math"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestForkVersion(t *testing.T) {
	forkData := &pb.ForkData{
		ForkSlot:        10,
		PreForkVersion:  2,
		PostForkVersion: 3,
	}

	if ForkVersion(forkData, 9) != 2 {
		t.Errorf("Fork Version not equal to 2 %d", ForkVersion(forkData, 9))
	}

	if ForkVersion(forkData, 11) != 3 {
		t.Errorf("Fork Version not equal to 3 %d", ForkVersion(forkData, 11))
	}
}

func TestDomainVersion(t *testing.T) {
	forkData := &pb.ForkData{
		ForkSlot:        10,
		PreForkVersion:  2,
		PostForkVersion: 3,
	}

	constant := uint64(math.Pow(2, 32))

	if DomainVersion(forkData, 9, 2) != 2*constant+2 {
		t.Errorf("Incorrect domain version %d", DomainVersion(forkData, 9, 2))
	}

	if DomainVersion(forkData, 11, 3) != 3*constant+3 {
		t.Errorf("Incorrect domain version %d", DomainVersion(forkData, 11, 3))
	}
}
