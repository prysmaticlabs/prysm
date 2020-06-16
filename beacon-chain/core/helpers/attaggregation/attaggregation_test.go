package attaggregation

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{
		AttestationAggregationStrategy: "naive",
	})
	defer resetCfg()
	os.Exit(m.Run())
}
