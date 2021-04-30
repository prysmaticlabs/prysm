package clientstats

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

type mockRT struct {
	body       string
	status     string
	statusCode int
}

func (rt *mockRT) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(rt.body)),
	}, nil
}

var _ http.RoundTripper = &mockRT{}

func TestBeaconNodeScraper(t *testing.T) {
	bnScraper := beaconNodeScraper{}
	bnScraper.tripper = &mockRT{body: prometheusTestBody}
	r, err := bnScraper.Scrape()
	assert.NoError(t, err, "Unexpected error calling beaconNodeScraper.Scrape")
	bs := &BeaconNodeStats{}
	err = json.NewDecoder(r).Decode(bs)
	assert.NoError(t, err, "Unexpected error decoding result of beaconNodeScraper.Scrape")
	// CommonStats
	assert.Equal(t, int64(225), bs.CPUProcessSecondsTotal)
	assert.Equal(t, int64(1166630912), bs.MemoryProcessBytes)
	assert.Equal(t, int64(1619586241), bs.ClientBuild)
	assert.Equal(t, "v1.3.8-hotfix+6c0942", bs.ClientVersion)
	assert.Equal(t, "prysm", bs.ClientName)

	// BeaconNodeStats
	assert.Equal(t, int64(256552), bs.SyncBeaconHeadSlot)
	assert.Equal(t, true, bs.SyncEth2Synced)
	assert.Equal(t, int64(7365341184), bs.DiskBeaconchainBytesTotal)
	assert.Equal(t, int64(37), bs.NetworkPeersConnected)
}

func TestFalseEth2Synced(t *testing.T) {
	bnScraper := beaconNodeScraper{}
	eth2NotSynced := strings.Replace(prometheusTestBody, "beacon_head_slot 256552", "beacon_head_slot 256559", 1)
	bnScraper.tripper = &mockRT{body: eth2NotSynced}
	r, err := bnScraper.Scrape()
	assert.NoError(t, err, "Unexpected error calling beaconNodeScraper.Scrape")

	bs := &BeaconNodeStats{}
	err = json.NewDecoder(r).Decode(bs)
	assert.NoError(t, err, "Unexpected error decoding result of beaconNodeScraper.Scrape")

	assert.Equal(t, false, bs.SyncEth2Synced)
}

func TestValidatorScraper(t *testing.T) {
	vScraper := validatorScraper{}
	vScraper.tripper = &mockRT{body: prometheusTestBody}
	r, err := vScraper.Scrape()
	assert.NoError(t, err, "Unexpected error calling validatorScraper.Scrape")
	vs := &ValidatorStats{}
	err = json.NewDecoder(r).Decode(vs)
	assert.NoError(t, err, "Unexpected error decoding result of validatorScraper.Scrape")
	// CommonStats
	assert.Equal(t, int64(225), vs.CPUProcessSecondsTotal)
	assert.Equal(t, int64(1166630912), vs.MemoryProcessBytes)
	assert.Equal(t, int64(1619586241), vs.ClientBuild)
	assert.Equal(t, "v1.3.8-hotfix+6c0942", vs.ClientVersion)
	assert.Equal(t, "prysm", vs.ClientName)
}

func mockNowFunc(fixedTime time.Time) func() time.Time {
	return func() time.Time {
		return fixedTime
	}
}

func TestValidatorAPIMessageDefaults(t *testing.T) {
	now = mockNowFunc(time.Unix(1619811114, 123456789))
	// 1+e6 ns per ms, so 123456789 ns rounded down should be 123 ms
	nowMillis := int64(1619811114123)
	vScraper := validatorScraper{}
	vScraper.tripper = &mockRT{body: prometheusTestBody}
	r, err := vScraper.Scrape()
	assert.NoError(t, err, "unexpected error from validatorScraper.Scrape()")

	vs := &ValidatorStats{}
	err = json.NewDecoder(r).Decode(vs)
	assert.NoError(t, err, "Unexpected error decoding result of validatorScraper.Scrape")

	// CommonStats
	assert.Equal(t, nowMillis, vs.Timestamp, "Unexpected 'timestamp' in client-stats APIMessage struct")
	assert.Equal(t, APIVersion, vs.APIVersion, "Unexpected 'version' in client-stats APIMessage struct")
	assert.Equal(t, ValidatorProcessName, vs.ProcessName, "Unexpected value for 'process' in client-stats APIMessage struct")
}

func TestBeaconNodeAPIMessageDefaults(t *testing.T) {
	now = mockNowFunc(time.Unix(1619811114, 123456789))
	// 1+e6 ns per ms, so 123456789 ns rounded down should be 123 ms
	nowMillis := int64(1619811114123)
	bScraper := beaconNodeScraper{}
	bScraper.tripper = &mockRT{body: prometheusTestBody}
	r, err := bScraper.Scrape()
	assert.NoError(t, err, "unexpected error from beaconNodeScraper.Scrape()")

	vs := &BeaconNodeStats{}
	err = json.NewDecoder(r).Decode(vs)
	assert.NoError(t, err, "Unexpected error decoding result of beaconNodeScraper.Scrape")

	// CommonStats
	assert.Equal(t, nowMillis, vs.Timestamp, "Unexpected 'timestamp' in client-stats APIMessage struct")
	assert.Equal(t, APIVersion, vs.APIVersion, "Unexpected 'version' in client-stats APIMessage struct")
	assert.Equal(t, BeaconNodeProcessName, vs.ProcessName, "Unexpected value for 'process' in client-stats APIMessage struct")
}

func TestBadInput(t *testing.T) {
	bnScraper := beaconNodeScraper{}
	bnScraper.tripper = &mockRT{body: ""}
	_, err := bnScraper.Scrape()
	assert.ErrorContains(t, "did not find metric family", err, "Expected errors for missing metric families on empty input.")
}

var prometheusTestBody = `
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 225.09
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 1.166630912e+09
# HELP prysm_version
# TYPE prysm_version gauge
prysm_version{buildDate="1619586241",commit="51eb1540fa838cdbe467bbeb0e36ee667d449377",version="v1.3.8-hotfix+6c0942"} 1
# HELP validator_count The total number of validators
# TYPE validator_count gauge
validator_count{state="Active"} 210301
validator_count{state="Exited"} 10
validator_count{state="Exiting"} 0
validator_count{state="Pending"} 0
validator_count{state="Slashed"} 0
validator_count{state="Slashing"} 0
# HELP beacon_head_slot Slot of the head block of the beacon chain
# TYPE beacon_head_slot gauge
beacon_head_slot 256552
# HELP beacon_clock_time_slot The current slot based on the genesis time and current clock
# TYPE beacon_clock_time_slot gauge
beacon_clock_time_slot 256552
# HELP bcnode_disk_beaconchain_bytes_total Total hard disk space used by the beaconchain database, in bytes. May include mmap.
# TYPE bcnode_disk_beaconchain_bytes_total gauge
bcnode_disk_beaconchain_bytes_total 7.365341184e+09
# HELP p2p_peer_count The number of peers in a given state.
# TYPE p2p_peer_count gauge
p2p_peer_count{state="Bad"} 1
p2p_peer_count{state="Connected"} 37
p2p_peer_count{state="Connecting"} 0
p2p_peer_count{state="Disconnected"} 62
p2p_peer_count{state="Disconnecting"} 0
`
