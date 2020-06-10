package p2p

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Test `verifyConnectivity` function by trying to connect to google.com (successfully)
// and then by connecting to an unreachable IP and ensuring that a log is emitted
func TestVerifyConnectivity(t *testing.T) {
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
				var buf bytes.Buffer
				logrus.SetOutput(&buf)
				defer func() {
					logrus.SetOutput(os.Stderr)
				}()
				verifyConnectivity(tc.address, tc.port, "tcp")
				logMessage := "IP address is not accessible"
				if tc.expectedConnectivity {
					testutil.AssertLogsDoNotContain(t, hook, logMessage)
				} else {
					testutil.AssertLogsContain(t, hook, logMessage)
				}
			})
	}
}
