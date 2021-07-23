package v1

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/proto/prysm/v2"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func Test_handlePendingAttestation_OutOfRange(t *testing.T) {
	items := make([]*pb.PendingAttestation, 1)
	indices := []uint64{3}
	_, err := handlePendingAttestation(items, indices, false)
	assert.ErrorContains(t, "index 3 greater than number of pending attestations 1", err)
}

func Test_handleEth1DataSlice_OutOfRange(t *testing.T) {
	items := make([]*ethpb.Eth1Data, 1)
	indices := []uint64{3}
	_, err := HandleEth1DataSlice(items, indices, false)
	assert.ErrorContains(t, "index 3 greater than number of items in eth1 data slice 1", err)

}
