package sync

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	allowedBlocksPerSecond = 64
	allowedBlocksBurst = int64(10 * allowedBlocksPerSecond)

	os.Exit(m.Run())
}
