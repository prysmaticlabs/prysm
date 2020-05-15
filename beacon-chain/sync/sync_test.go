package sync

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	flags.Init(&flags.GlobalFlags{
		BlockBatchLimit:            64,
		BlockBatchLimitBurstFactor: 10,
	})

	os.Exit(m.Run())
}
