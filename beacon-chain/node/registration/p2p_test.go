package registration

import (
	"io/ioutil"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBootStrapNodeFile(t *testing.T) {
	file, err := ioutil.TempFile(t.TempDir(), "bootstrapFile")
	require.NoError(t, err)

	sampleNode0 := "- enr:-Ku4QMKVC_MowDsmEa20d5uGjrChI0h8_KsKXDmgVQbIbngZV0i" +
		"dV6_RL7fEtZGo-kTNZ5o7_EJI_vCPJ6scrhwX0Z4Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD" +
		"1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQJxCnE6v_x2ekgY_uo" +
		"E1rtwzvGy40mq9eD66XfHPBWgIIN1ZHCCD6A"
	sampleNode1 := "- enr:-TESTNODE2"
	sampleNode2 := "- enr:-TESTNODE3"
	err = ioutil.WriteFile(file.Name(), []byte(sampleNode0+"\n"+sampleNode1+"\n"+sampleNode2), 0644)
	require.NoError(t, err, "Error in WriteFile call")
	nodeList, err := readbootNodes(file.Name())
	require.NoError(t, err, "Error in readbootNodes call")
	assert.Equal(t, sampleNode0[2:], nodeList[0], "Unexpected nodes")
	assert.Equal(t, sampleNode1[2:], nodeList[1], "Unexpected nodes")
	assert.Equal(t, sampleNode2[2:], nodeList[2], "Unexpected nodes")
}
