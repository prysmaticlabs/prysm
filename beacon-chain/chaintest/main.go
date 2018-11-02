package main

import (
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
)

func main() {
	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	data, err := ioutil.ReadFile("./sampletests/basic_fork_choice.yaml")
	if err != nil {
		log.Fatalf("Could not read file: %v", err)
	}
	chainTests := &backend.ChainTest{}
	if err := yaml.Unmarshal(data, chainTests); err != nil {
		log.Fatalf("Could not unmarshal YAML file into test struct: %v", err)
	}
	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		log.Fatalf("Could not create backend: %v", err)
	}
	log.Info("----Running Chain Tests----")
	log.Infof("Title: %v", chainTests.Title)
	log.Infof("Summary: %v", chainTests.Summary)
	log.Infof("Test Suite: %v", chainTests.TestSuite)
	startTime := time.Now()
	for _, tt := range chainTests.TestCases {
		if err := sb.RunChainTest(tt); err != nil {
			log.Fatalf("Could not run chain test: %v", err)
		}
	}
	endTime := time.Now()
	log.Infof("Test Runs Finished In: %v Seconds", endTime.Sub(startTime).Seconds())
}
