package eth1

import (
	"time"

	e2etypes "github.com/prysmaticlabs/prysm/testing/endtoend/types"
)

const minerPort = 30303
const minerPasswordFile = "password.txt"
const minerFile = "UTC--2021-12-22T19-14-08.590377700Z--878705ba3f8bc32fcf7f4caa1a35e72af65cf766"
const timeGapPerTX = 100 * time.Millisecond
const NetworkId = 123456
const staticFilesPath = "/testing/endtoend/static-files/eth1"
const KeystorePassword = "password"

var _ e2etypes.ComponentRunner = (*NodeSet)(nil)
var _ e2etypes.ComponentRunner = (*Miner)(nil)
var _ e2etypes.ComponentRunner = (*Node)(nil)
