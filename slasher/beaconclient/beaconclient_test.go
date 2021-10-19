package beaconclient

import (
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
)

var (
	_ = Notifier(&Service{})
	_ = ChainFetcher(&Service{})
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)

	m.Run()
}
