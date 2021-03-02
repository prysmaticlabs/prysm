package sync

import (
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	resetFlags := flags.Get()
	flags.Init(&flags.GlobalFlags{
		BlockBatchLimit:            64,
		BlockBatchLimitBurstFactor: 10,
	})
	defer func() {
		flags.Init(resetFlags)
	}()
	m.Run()
}
