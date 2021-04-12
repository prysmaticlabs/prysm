package powchain

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// TestBeaconNode_extractAuthStringFromFlag tests extract auth string from flag
func TestHttpEndpoint(t *testing.T) {
	hook := logTest.NewGlobal()
	httpWeb3 := "http://infura"
	auth := "bearer xxxxxxxxx"
	separator := ","
	web3, au := HttpEndpoint(httpWeb3 + separator + auth)
	require.Equal(t, httpWeb3, web3)
	require.Equal(t, auth, au)
	web3, au = HttpEndpoint(httpWeb3 + separator)
	require.Equal(t, httpWeb3, web3)
	require.Equal(t, "", au)
	web3, au = HttpEndpoint(httpWeb3)
	require.Equal(t, httpWeb3, web3)
	require.Equal(t, "", au)
	web3, au = HttpEndpoint(httpWeb3 + separator + auth + separator)
	require.Equal(t, httpWeb3, web3)
	require.Equal(t, "", au)
	require.LogsContain(t, hook, "Web 3 provider string can contain one comma")
}
