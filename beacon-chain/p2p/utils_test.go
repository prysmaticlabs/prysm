package p2p

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Test `verifyConnectivity` function by trying to connect to google.com (successfully)
// and then by connecting to an unreachable IP and ensuring that a log is emitted
func TestVerifyConnectivity(t *testing.T) {
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
					testutil.AssertLogsDoNotContain(t, hook, logMessage)
				} else {
					testutil.AssertLogsContain(t, hook, logMessage)
				}
			})
	}
}

func TestRoundtripSerialization(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	db, err := enode.OpenDB("")
	require.NoError(t, err, "could not open node's peer database")

	defer db.Close()
	lNode := enode.NewLocalNode(db, key)
	rec := lNode.Node().Record()

	raw, err := SerializeENR(rec)
	require.NoError(t, err)
	nRec, err := deserializeENR([]byte(raw))
	require.NoError(t, err)

	assert.DeepEqual(t, rec.Signature(), nRec.Signature())
}

func TestProcessENR(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	db, err := enode.OpenDB("")
	require.NoError(t, err, "could not open node's peer database")

	defer db.Close()
	lNode := enode.NewLocalNode(db, key)

	key2, err := crypto.GenerateKey()
	require.NoError(t, err)
	db2, err := enode.OpenDB("")
	require.NoError(t, err, "could not open node's peer database")

	defer db2.Close()
	lNode2 := enode.NewLocalNode(db2, key2)

	// Set a different sequence number
	lNode.Node().Record().SetSeq(2)
	err = processENR(lNode.Node().Record(), lNode2)
	require.NoError(t, err)

	assert.NotEqual(t, lNode.Seq(), lNode2.Seq(), "ENRs sequence number was updated")
}
