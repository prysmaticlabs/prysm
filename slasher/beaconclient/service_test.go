package beaconclient

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

var (
	_ = Notifier(&Service{})
	_ = ChainFetcher(&Service{})
)

func TestMain(m *testing.M) {
	run := func() int {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetOutput(ioutil.Discard)

		return m.Run()
	}
	os.Exit(run())
}
