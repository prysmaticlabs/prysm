package clientstats

import (
	"bytes"
	"encoding/json"
	"fmt"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
)

func scrapeProm(url string) (map[string]*dto.MetricFamily, error) {
	mfChan := make(chan *dto.MetricFamily, 1024)
	err := prom2json.FetchMetricFamilies(url, mfChan, http.DefaultTransport)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*dto.MetricFamily)
	for fam := range mfChan {
		ptr := fam
		result[fam.GetName()] = ptr
	}
	return result, nil
}

type validatorScraper struct {
	url string
}

func (vc *validatorScraper) Scrape() (io.Reader, error) {
	log.Infof("Scraping validator at %s", vc.url)
	pf, err := scrapeProm(vc.url)
	if err != nil {
		return nil, nil
	}

	cs, err := populateCommonStats(pf)
	if err != nil {
		return nil, err
	}
	vs, err := populateValidatorStats(pf)
	if err != nil {
		return nil, err
	}
	vs.CommonStats = cs

	b, err := json.Marshal(vs)
	return bytes.NewBuffer(b), err
}

func NewValidatorScraper(promExpoURL string) Scraper {
	return &validatorScraper{
		url: promExpoURL,
	}
}

type beaconNodeScraper struct {
	url string
}

func populateCommonStats(pf map[string]*dto.MetricFamily) (CommonStats, error) {
	cs := CommonStats{}
	cs.ClientName = "prysm"
	var f *dto.MetricFamily
	var m *dto.Metric
	var ok bool

	f, ok = pf["process_cpu_seconds_total"]
	if ok {
		m = f.Metric[0]
		// float64->int64: truncates fractional seconds
		cs.CPUProcessSecondsTotal = int64(m.Counter.GetValue())
	}

	f, ok = pf["process_resident_memory_bytes"]
	if ok {
		m = f.Metric[0]
		cs.MemoryProcessBytes = int64(m.Gauge.GetValue())
	}

	f, ok = pf["prysm_version"]
	if ok {
		m = f.Metric[0]
		for _, l := range m.GetLabel(){
			switch l.GetName() {
			case "version":
				cs.ClientVersion = l.GetValue()
			case "buildDate":
				buildDate, err := strconv.Atoi(l.GetValue())
				if err != nil {
					return cs, fmt.Errorf("Error when retrieving buildDate label from the prysm_version metric: %s", err)
				}
				cs.ClientBuild = int64(buildDate)
			}
		}
	}

	return cs, nil
}

func populateValidatorStats(pf map[string]*dto.MetricFamily) (ValidatorStats, error) {
	vs := ValidatorStats{}

	//var f *dto.MetricFamily

	/*
	// woops this is the name of the key in the beacon node
	// TODO: determine if we can get this stat from validator clients
	f = pf["validator_count"]
	for _, m := range f.Metric {
		v := int64(m.Gauge.GetValue())
		for _, l := range m.GetLabel() {
			if l.GetName() == "state" {
				if l.GetValue() == "Active" {
					vs.ValidatorActive = v
				}
			}
		}
		vs.ValidatorTotal += v
	}
	 */

	return vs, nil
}

func populateBeaconNodeStats(pf map[string]*dto.MetricFamily) (BeaconNodeStats, error) {
	var err error
	bs := BeaconNodeStats{}
	bs.CommonStats, err = populateCommonStats(pf)
	if err != nil {
		return bs, err
	}

	var f *dto.MetricFamily
	var m *dto.Metric
	var ok bool

	f, ok = pf["beacon_head_slot"]
	if ok {
		m = f.Metric[0]
		bs.SyncBeaconHeadSlot = int64(m.Gauge.GetValue())
	}

	f, ok = pf["beacon_clock_time_slot"]
	if ok {
		m = f.Metric[0]
		if int64(m.Gauge.GetValue()) == bs.SyncBeaconHeadSlot {
			bs.SyncEth2Synced = true
		}
	}

	f, ok = pf["bcnode_disk_beaconchain_bytes_total"]
	if ok {
		m = f.Metric[0]
		bs.DiskBeaconchainBytesTotal = int64(m.Gauge.GetValue())
	}

	f, ok = pf["p2p_peer_count"]
	if ok {
		for _, m := range f.Metric {
			for _, l := range m.GetLabel() {
				if l.GetName() == "state" {
					if l.GetValue() == "Connected" {
						bs.NetworkPeersConnected = int64(m.Gauge.GetValue())
					}
				}
			}
		}
	}

	return bs, nil
}

func (vc *beaconNodeScraper) Scrape() (io.Reader, error) {
	log.Infof("Scraping beacon-node at %s", vc.url)
	pf, err := scrapeProm(vc.url)
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

func NewBeaconNodeScraper(promExpoURL string) Scraper {
	return &beaconNodeScraper{
		url: promExpoURL,
	}
}
