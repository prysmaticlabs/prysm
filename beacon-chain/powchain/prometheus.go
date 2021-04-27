package powchain

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prysmaticlabs/prysm/shared/clientstats"
	"sync"
)

type powchainCollector struct {
	SyncEth1Connected *prometheus.Desc
	SyncEth1FallbackConnected *prometheus.Desc
	SyncEth1FallbackConfigured *prometheus.Desc // true if flag specified: --fallback-web3provider
	updateChan chan clientstats.BeaconNodeStats
	latestStats clientstats.BeaconNodeStats
	sync.Mutex
}

type BeaconnodeStatsUpdater interface {
	Update(stats clientstats.BeaconNodeStats)
}

func NewBeaconnodeStatsUpdater() (BeaconnodeStatsUpdater, error) {
	namespace := "powchain"
	updateChan := make(chan clientstats.BeaconNodeStats, 2)
	c := &powchainCollector{
		SyncEth1FallbackConfigured: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "sync_eth1_fallback_configured"),
			"Boolean recording whether a fallback eth1 endpoint was configured: 0=false, 1=true.",
			nil,
			nil,
		),
		SyncEth1FallbackConnected: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "sync_eth1_fallback_connected"),
			"Boolean indicating whether a fallback eth1 endpoint is currently connected: 0=false, 1=true.",
			nil,
			nil,
		),
		SyncEth1Connected: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "sync_eth1_connected"),
			"Boolean indicating whether a fallback eth1 endpoint is currently connected: 0=false, 1=true.",
			nil,
			nil,
		),
		updateChan: updateChan,
	}
	go c.latestStatsUpdateLoop()
	return c, prometheus.Register(c)
}

func (pc *powchainCollector) Describe(ch chan <-*prometheus.Desc) {
	ch <- pc.SyncEth1Connected
	ch <- pc.SyncEth1FallbackConfigured
	ch <- pc.SyncEth1FallbackConnected
}

func (pc *powchainCollector) Collect(ch chan <-prometheus.Metric) {
	bs := pc.getLatestStats()

	var syncEth1FallbackConfigured float64 = 0
	if bs.SyncEth1FallbackConfigured {
		syncEth1FallbackConfigured = 1
	}
	ch <- prometheus.MustNewConstMetric(
		pc.SyncEth1FallbackConfigured,
		prometheus.GaugeValue,
		syncEth1FallbackConfigured,
	)

	var syncEth1FallbackConnected float64 = 0
	if bs.SyncEth1FallbackConnected {
		syncEth1FallbackConnected = 1
	}
	ch <- prometheus.MustNewConstMetric(
		pc.SyncEth1FallbackConnected,
		prometheus.GaugeValue,
		syncEth1FallbackConnected,
	)

	var syncEth1Connected float64 = 0
	if bs.SyncEth1Connected {
		syncEth1Connected = 1
	}
	ch <- prometheus.MustNewConstMetric(
		pc.SyncEth1Connected,
		prometheus.GaugeValue,
		syncEth1Connected,
	)
}

func (pc *powchainCollector) getLatestStats() clientstats.BeaconNodeStats {
	pc.Lock()
	defer pc.Unlock()
	return pc.latestStats
}

func (pc *powchainCollector) setLatestStats(bs clientstats.BeaconNodeStats) {
	pc.Lock()
	pc.latestStats = bs
	pc.Unlock()
}

func (pc *powchainCollector) latestStatsUpdateLoop() {
	for bs := range pc.updateChan {
		pc.setLatestStats(bs)
	}
}

func (pc *powchainCollector) Update(update clientstats.BeaconNodeStats) {
	pc.updateChan <- update
}