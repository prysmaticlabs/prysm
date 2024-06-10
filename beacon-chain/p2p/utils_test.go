package p2p

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/peerdas"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	ecdsaprysm "github.com/prysmaticlabs/prysm/v5/crypto/ecdsa"
	"github.com/prysmaticlabs/prysm/v5/crypto/rand"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// generateRandomSubnets generates a set of `count` random subnets.
func generateRandomSubnets(requestedCount, totalSubnetsCount uint64) map[uint64]bool {
	// Populate all the subnets.
	subnets := make(map[uint64]bool, totalSubnetsCount)
	for i := uint64(0); i < totalSubnetsCount; i++ {
		subnets[i] = true
	}

	// Get a random generator.
	randGen := rand.NewGenerator()

	// Randomly delete subnets until we have the desired count.
	for uint64(len(subnets)) > requestedCount {
		// Get a random subnet.
		subnet := randGen.Uint64() % totalSubnetsCount

		// Delete the subnet.
		delete(subnets, subnet)
	}

	return subnets
}

func TestRandomPrivKeyWithConstraint(t *testing.T) {
	// Get the total number of subnets.
	totalSubnetsCount := params.BeaconConfig().DataColumnSidecarSubnetCount

	// We generate only tests for a low and high number of subnets to minimize computation, as explained here:
	// https://hackmd.io/@6-HLeMXARN2tdFLKKcqrxw/BJVSxU7VC
	testCases := []struct {
		name                 string
		expectedSubnetsCount uint64
		expectedError        bool
	}{
		{
			name:                 "0 subnet - n subnets",
			expectedSubnetsCount: 0,
		},
		{
			name:                 "1 subnet - n-1 subnets",
			expectedSubnetsCount: 1,
		},
		{
			name:                 "2 subnets - n-2 subnets",
			expectedSubnetsCount: 2,
		},
		{
			name:                 "3 subnets - n-3 subnets",
			expectedSubnetsCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expectedSubnetsList := []map[uint64]bool{
				generateRandomSubnets(tc.expectedSubnetsCount, totalSubnetsCount),
				generateRandomSubnets(totalSubnetsCount-tc.expectedSubnetsCount, totalSubnetsCount),
			}

			for _, expectedSubnets := range expectedSubnetsList {
				// Determine the number of expected subnets.
				expectedSubnetsCount := uint64(len(expectedSubnets))

				// Determine the private key that matches the expected subnets.
				privateKey, iterationsCount, _, err := randomPrivKeyWithSubnets(expectedSubnets)
				require.NoError(t, err)

				// Sanity check the number of iterations.
				assert.Equal(t, true, iterationsCount > 0)

				// Compute the node ID from the public key.
				ecdsaPrivKey, err := ecdsaprysm.ConvertFromInterfacePrivKey(privateKey)
				require.NoError(t, err)

				nodeID := enode.PubkeyToIDV4(&ecdsaPrivKey.PublicKey)

				// Retrieve the subnets from the node ID.
				actualSubnets, err := peerdas.CustodyColumnSubnets(nodeID, expectedSubnetsCount)
				require.NoError(t, err)

				// Determine the number of actual subnets.
				actualSubnetsCounts := uint64(len(actualSubnets))

				// Check the count of the actual subnets against the expected subnets.
				assert.Equal(t, expectedSubnetsCount, actualSubnetsCounts)

				// Check the actual subnets against the expected subnets.
				for _, subnet := range actualSubnets {
					assert.Equal(t, true, expectedSubnets[subnet])
				}
			}
		})
	}
}

// Test `verifyConnectivity` function by trying to connect to google.com (successfully)
// and then by connecting to an unreachable IP and ensuring that a log is emitted
func TestVerifyConnectivity(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	hook := logTest.NewGlobal()
	cases := []struct {
		address              string
		port                 uint
		expectedConnectivity bool
		name                 string
	}{
		{"142.250.68.46", 80, true, "Dialing a reachable IP: 142.250.68.46:80"}, // google.com
		{"123.123.123.123", 19000, false, "Dialing an unreachable IP: 123.123.123.123:19000"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf(tc.name),
			func(t *testing.T) {
				verifyConnectivity(tc.address, tc.port, "tcp")
				logMessage := "IP address is not accessible"
				if tc.expectedConnectivity {
					require.LogsDoNotContain(t, hook, logMessage)
				} else {
					require.LogsContain(t, hook, logMessage)
				}
			})
	}
}

func TestSerializeENR(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	t.Run("Ok", func(t *testing.T) {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		db, err := enode.OpenDB("")
		require.NoError(t, err)
		lNode := enode.NewLocalNode(db, key)
		record := lNode.Node().Record()
		s, err := SerializeENR(record)
		require.NoError(t, err)
		assert.NotEqual(t, "", s)
		s = "enr:" + s
		newRec, err := enode.Parse(enode.ValidSchemes, s)
		require.NoError(t, err)
		assert.Equal(t, s, newRec.String())
	})

	t.Run("Nil record", func(t *testing.T) {
		_, err := SerializeENR(nil)
		require.NotNil(t, err)
		assert.ErrorContains(t, "could not serialize nil record", err)
	})
}
