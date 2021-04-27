package node

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"os"
)

type bcnodeCollector struct {
	DiskBeaconchainBytesTotal *prometheus.Desc
	dbPath string
}

func startBeaconNodePromCollector(dbPath string) error {
	namespace := "bcnode"
	c := &bcnodeCollector{
		DiskBeaconchainBytesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "disk_beaconchain_bytes_total"),
			"Total hard disk space used by the beaconchain database, in bytes.",
			nil,
			nil,
		),
		dbPath: dbPath,
	}
	_, err := c.getCurrentDbBytes()
	if err != nil {
		return err
	}
	return prometheus.Register(c)
}

func (bc *bcnodeCollector) Describe(ch chan <-*prometheus.Desc) {
	ch <- bc.DiskBeaconchainBytesTotal
}

func (bc *bcnodeCollector) Collect(ch chan <-prometheus.Metric) {
	dbBytes, err := bc.getCurrentDbBytes()
	if err != nil {
		log.Warn(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		bc.DiskBeaconchainBytesTotal,
		prometheus.GaugeValue,
		dbBytes,
	)
}

func (bc *bcnodeCollector) getCurrentDbBytes() (float64, error) {
	fs, err := os.Stat(bc.dbPath)
	if err != nil {
		return 0, fmt.Errorf("Could not collect database file size for prometheus, path=%s, err=%s", bc.dbPath, err)
	}
	return float64(fs.Size()), nil
}