package p2p

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/peers/scorers"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestCustodyCountFromRemotePeer(t *testing.T) {
	const (
		expected uint64 = 7
		pid             = "test-id"
	)

	csc := CustodySubnetCount(expected)

	// Define a nil record
	var nilRecord *enr.Record = nil

	// Define an empty record (record with non `csc` entry)
	emptyRecord := &enr.Record{}

	// Define a nominal record
	nominalRecord := &enr.Record{}
	nominalRecord.Set(csc)

	testCases := []struct {
		name     string
		record   *enr.Record
		expected uint64
	}{
		{
			name:     "nominal",
			record:   nominalRecord,
			expected: expected,
		},
		{
			name:     "nil",
			record:   nilRecord,
			expected: params.BeaconConfig().CustodyRequirement,
		},
		{
			name:     "empty",
			record:   emptyRecord,
			expected: params.BeaconConfig().CustodyRequirement,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create peers status.
			peers := peers.NewStatus(context.Background(), &peers.StatusConfig{
				PeerLimit:    30,
				ScorerParams: &scorers.Config{},
			})

			// Add a new peer with the record.
			peers.Add(tc.record, pid, nil, network.DirOutbound)

			// Create a new service.
			service := &Service{
				peers: peers,
			}

			// Retrieve the custody count from the remote peer.
			actual := service.CustodyCountFromRemotePeer(pid)

			// Verify the result.
			require.Equal(t, tc.expected, actual)
		})
	}

}
