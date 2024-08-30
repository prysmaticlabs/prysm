package p2p

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/wrapper"
	pb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

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

func TestMetaDataFromFile(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, metaDataPath)

	// Generate metadata V1
	seqNum := rand.Uint64()
	md := &pb.MetaDataV1{
		SeqNumber: seqNum,
		Attnets:   bitfield.NewBitvector64(),
		Syncnets:  bitfield.NewBitvector4(),
	}
	metaData := wrapper.WrappedMetadataV1(md)

	// Save to file
	err := saveMetaDataToFile(path, metaData.Copy())
	require.NoError(t, err)

	// Load file, and compare
	mdFromFile, err := metaDataFromFile(path)
	require.NoError(t, err)
	require.DeepEqual(t, metaData.Copy(), mdFromFile.Copy())
}

func TestMetaDataFromFile_V0(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, metaDataPath)

	// Generate metadata V0
	seqNum := rand.Uint64()
	md := &pb.MetaDataV0{
		SeqNumber: seqNum,
		Attnets:   bitfield.NewBitvector64(),
	}
	metaData := wrapper.WrappedMetadataV0(md)

	// Save to file
	err := saveMetaDataToFile(path, metaData.Copy())
	require.NoError(t, err)

	// Load file, and compare
	mdFromFile, err := metaDataFromFile(path)
	require.NoError(t, err)
	require.DeepEqual(t, metaData.Copy(), mdFromFile.Copy())
}
