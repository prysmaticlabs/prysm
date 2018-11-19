package main

import (
	"flag"
	"io/ioutil"
	"path"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func main() {
	var yamlDir = flag.String("tests-dir", "", "path to directory of yaml tests")
	flag.Parse()

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	var chainTests []*backend.ChainTest

	files, err := ioutil.ReadDir(*yamlDir)
	if err != nil {
		log.Fatalf("Could not read yaml tests directory: %v", err)
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(path.Join(*yamlDir, file.Name()))
		if err != nil {
			log.Fatalf("Could not read yaml file: %v", err)
		}
		decoded := &backend.ChainTest{}
		if err := yaml.Unmarshal(data, decoded); err != nil {
			log.Fatalf("Could not unmarshal YAML file into test struct: %v", err)
		}
		chainTests = append(chainTests, decoded)
	}

	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		log.Fatalf("Could not create backend: %v", err)
	}

	log.Info("----Running Chain Tests----")
	startTime := time.Now()

	for _, tt := range chainTests {
		log.Infof("Title: %v", tt.Title)
		log.Infof("Summary: %v", tt.Summary)
		log.Infof("Test Suite: %v", tt.TestSuite)
		for _, testCase := range tt.TestCases {
			if err := sb.RunChainTest(testCase); err != nil {
				log.Fatalf("Could not run chain test: %v", err)
			}
		}
	}

	endTime := time.Now()
	log.Infof("Test Runs Finished In: %v Seconds", endTime.Sub(startTime).Seconds())
}
