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
		panic(err)
	}

	chainTests := &backend.ChainTest{}
	if err := yaml.Unmarshal(data, chainTests); err != nil {
		panic(err)
	}
	log.Info("----Running Chain Test----")
	log.Infof("Title: %v", chainTests.Title)
	log.Infof("Summary: %v", chainTests.Summary)
	log.Infof("Test Suite: %v", chainTests.TestSuite)
}
