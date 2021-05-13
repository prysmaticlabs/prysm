package clientstats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	log "github.com/sirupsen/logrus"
)

type beaconNodeScraper struct {
	url     string
	tripper http.RoundTripper
}

func (bc *beaconNodeScraper) Scrape() (io.Reader, error) {
	log.Infof("Scraping beacon-node at %s", bc.url)
	pf, err := scrapeProm(bc.url, bc.tripper)
	if err != nil {
		return nil, nil
	}

	bs, err := populateBeaconNodeStats(pf)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(bs)
	return bytes.NewBuffer(b), err
}

// NewBeaconNodeScraper constructs a Scaper capable of scraping
// the prometheus endpoint of a beacon-node process and producing
// the json body for the beaconnode client-stats process type.
func NewBeaconNodeScraper(promExpoURL string) Scraper {
	return &beaconNodeScraper{
		url: promExpoURL,
	}
}

type validatorScraper struct {
	url     string
	tripper http.RoundTripper
}

func (vc *validatorScraper) Scrape() (io.Reader, error) {
	log.Infof("Scraping validator at %s", vc.url)
	pf, err := scrapeProm(vc.url, vc.tripper)
	if err != nil {
		return nil, nil
	}

	vs, err := populateValidatorStats(pf)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(vs)
	return bytes.NewBuffer(b), err
}

// NewValidatorScraper constructs a Scaper capable of scraping
// the prometheus endpoint of a validator process and producing
// the json body for the validator client-stats process type.
func NewValidatorScraper(promExpoURL string) Scraper {
	return &validatorScraper{
		url: promExpoURL,
	}
}

// note on tripper -- under the hood FetchMetricFamilies constructs an http.Client,
// which, if transport is nil, will just use the DefaultTransport, so we
// really only bother specifying the transport in tests, otherwise we let
// the zero-value (which is nil) flow through so that the default transport
// will be used.
func scrapeProm(url string, tripper http.RoundTripper) (map[string]*dto.MetricFamily, error) {
	mfChan := make(chan *dto.MetricFamily)
	errChan := make(chan error)
	go func() {
		// FetchMetricFamilies handles grpc flavored prometheus ez
		// but at the cost of the awkward channel select loop below
		err := prom2json.FetchMetricFamilies(url, mfChan, tripper)
		if err != nil {
			errChan <- err
		}
	}()
	result := make(map[string]*dto.MetricFamily)
	// channel select accumulates results from FetchMetricFamilies
	// unless there is an error.
	for {
		select {
		case fam, chanOpen := <-mfChan:
			// FetchMetricFamiles will close the channel when done
			// at which point we want to stop the goroutine
			if fam == nil && !chanOpen {
				return result, nil
			}
			ptr := fam
			result[fam.GetName()] = ptr
		case err := <-errChan:
			return result, err
		}
		if errChan == nil && mfChan == nil {
			return result, nil
		}
	}
}

type metricMap map[string]*dto.MetricFamily

func (mm metricMap) getFamily(name string) (*dto.MetricFamily, error) {
	f, ok := mm[name]
	if !ok {
		return nil, fmt.Errorf("scraper did not find metric family %s", name)
	}
	return f, nil
}

var now = time.Now // var hook for tests to overwrite
var nanosPerMilli = int64(time.Millisecond) / int64(time.Nanosecond)

func populateAPIMessage(processName string) APIMessage {
	return APIMessage{
		Timestamp:   now().UnixNano() / nanosPerMilli,
		APIVersion:  APIVersion,
		ProcessName: processName,
	}
}

func populateCommonStats(pf metricMap) (CommonStats, error) {
	cs := CommonStats{}
	cs.ClientName = ClientName
	var f *dto.MetricFamily
	var m *dto.Metric
	var err error

	f, err = pf.getFamily("process_cpu_seconds_total")
	if err != nil {
		return cs, err
	}
	m = f.Metric[0]
	// float64->int64: truncates fractional seconds
	cs.CPUProcessSecondsTotal = int64(m.Counter.GetValue())

	f, err = pf.getFamily("process_resident_memory_bytes")
	if err != nil {
		return cs, err
	}
	m = f.Metric[0]
	cs.MemoryProcessBytes = int64(m.Gauge.GetValue())

	f, err = pf.getFamily("prysm_version")
	if err != nil {
		return cs, err
	}
	m = f.Metric[0]
	for _, l := range m.GetLabel() {
		switch l.GetName() {
		case "version":
			cs.ClientVersion = l.GetValue()
		case "buildDate":
			buildDate, err := strconv.Atoi(l.GetValue())
			if err != nil {
				return cs, fmt.Errorf("error when retrieving buildDate label from the prysm_version metric: %s", err)
			}
			cs.ClientBuild = int64(buildDate)
		}
	}

	return cs, nil
}

func populateBeaconNodeStats(pf metricMap) (BeaconNodeStats, error) {
	var err error
	bs := BeaconNodeStats{}
	bs.CommonStats, err = populateCommonStats(pf)
	if err != nil {
		return bs, err
	}
	bs.APIMessage = populateAPIMessage(BeaconNodeProcessName)

	var f *dto.MetricFamily
	var m *dto.Metric

	f, err = pf.getFamily("beacon_head_slot")
	if err != nil {
		return bs, err
	}
	m = f.Metric[0]
	bs.SyncBeaconHeadSlot = int64(m.Gauge.GetValue())

	f, err = pf.getFamily("beacon_clock_time_slot")
	if err != nil {
		return bs, err
	}
	m = f.Metric[0]
	if int64(m.Gauge.GetValue()) == bs.SyncBeaconHeadSlot {
		bs.SyncEth2Synced = true
	}

	f, err = pf.getFamily("bcnode_disk_beaconchain_bytes_total")
	if err != nil {
		return bs, err
	}
	m = f.Metric[0]
	bs.DiskBeaconchainBytesTotal = int64(m.Gauge.GetValue())

	f, err = pf.getFamily("p2p_peer_count")
	if err != nil {
		return bs, err
	}
	for _, m := range f.Metric {
		for _, l := range m.GetLabel() {
			if l.GetName() == "state" {
				if l.GetValue() == "Connected" {
					bs.NetworkPeersConnected = int64(m.Gauge.GetValue())
				}
			}
		}
	}

	return bs, nil
}

func statusIsActive(statusCode int64) bool {
	s := eth.ValidatorStatus(statusCode)
	return s.String() == "ACTIVE"
}

func populateValidatorStats(pf metricMap) (ValidatorStats, error) {
	var err error
	vs := ValidatorStats{}
	vs.CommonStats, err = populateCommonStats(pf)
	if err != nil {
		return vs, err
	}
	vs.APIMessage = populateAPIMessage(ValidatorProcessName)

	f, err := pf.getFamily("validator_statuses")
	if err != nil {
		return vs, err
	}
	for _, m := range f.Metric {
		if statusIsActive(int64(m.Gauge.GetValue())) {
			vs.ValidatorActive += 1
		}
		vs.ValidatorTotal += 1
	}

	return vs, nil
}
