// Package flags contains all configuration runtime flags for
// the client-stats daemon.
package flags

import (
	"github.com/urfave/cli/v2"
)

var (
	// BeaconCertFlag defines a flag for the beacon api certificate.
	ValidatorMetricsURLFlag = &cli.StringFlag{
		Name:  "validator-metrics-url",
		Usage: "Full URL to the validator /metrics prometheus endpoint to scrape. eg http://localhost:8081/metrics",
	}
	// BeaconRPCProviderFlag defines a flag for the beacon host ip or address.
	BeaconnodeMetricsURLFlag = &cli.StringFlag{
		Name:  "beacon-node-metrics-url",
		Usage: "Full URL to the beacon-node /metrics prometheus endpoint to scrape. eg http://localhost:8080/metrics",
	}
	// CertFlag defines a flag for the node's TLS certificate.
	ClientStatsAPIURLFlag = &cli.StringFlag{
		Name:  "clientstats-api-url",
		Usage: "Full URL to the client stats endpoint where collected metrics should be sent.",
	}
	// CertFlag defines a flag for the node's TLS certificate.
	ScrapeIntervalFlag = &cli.DurationFlag{
		Name:  "scrape-interval",
		Usage: "Frequency of scraping expressed as a duration, eg 2m or 1m5s. Default is 60s.",
	}
)
