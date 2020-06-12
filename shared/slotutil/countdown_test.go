package slotutil

import (
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestCountdownToGenesis(t *testing.T) {
	hook := logTest.NewGlobal()
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
	config.GenesisCountdownInterval = time.Second
	params.OverrideBeaconConfig(config)

	firstStringResult := "1 minute(s) until chain genesis"
	CountdownToGenesis(roughtime.Now().Add(2*time.Second), params.BeaconConfig().MinGenesisActiveValidatorCount)
	testutil.AssertLogsContain(t, hook, firstStringResult)
}

func Test_formatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "formats seconds properly",
			duration: time.Unix(20, 1000).Sub(time.Unix(0, 0)),
			want:     "20s",
		},
		{
			name:     "formats minutes properly",
			duration: time.Unix(80, 1000).Sub(time.Unix(0, 0)),
			want:     "1m20s",
		},
		{
			name:     "formats hours properly",
			duration: time.Unix(3680, 1000).Sub(time.Unix(0, 0)),
			want:     "1h1m20s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.duration); got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}
