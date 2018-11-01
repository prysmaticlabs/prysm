package main

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
)

func main() {
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
	for _, tt := range chainTests.TestCases {
		sb.RunChainTest(tt)
	}
}
