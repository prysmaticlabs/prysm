package spectest

import (
	"testing"
)

func TestSlashingsMainnet(t *testing.T) {
	t.Skip("We'll need to generate spec test for new hardfork configs")
	runSlashingsTests(t, "mainnet")
}
