package p2p

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// Test `verifyConnectivity` function by trying to connect to google.com (successfully)
// and then by connecting to an unreachable IP and ensuring that a log is emitted
func TestVerifyConnectivity(t *testing.T) {
	cases := []struct {
		Address              string
		Port                 uint
		ExpectedConnectivity bool
	}{
		{"142.250.68.46", 80, true}, // google.com
		{"123.123.123.123", 19000, false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("Dialing %s:%d - Connectivy should be %v", tc.Address, tc.Port, tc.ExpectedConnectivity),
			func(t *testing.T) {
				var buf bytes.Buffer
				logrus.SetOutput(&buf)
				defer func() {
					logrus.SetOutput(os.Stderr)
				}()
				verifyConnectivity(tc.Address, tc.Port, "tcp")
				if tc.ExpectedConnectivity && buf.String() != "" {
					t.Fatal("Connectivity was supposed to be successful and not emit any warning log message")
				}
				if !tc.ExpectedConnectivity && !strings.Contains(buf.String(), "IP address is not accessible") {
					t.Fatal("Expected a warning log message alerting an unreachable ip address")
				}
			})
	}
}
