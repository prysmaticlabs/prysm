package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: false,
	})
}

func readTestsFromYaml(yamlDir string) ([]interface{}, error) {
	const forkChoiceTestsFolderName = "fork-choice-tests"
	const shuffleTestsFolderName = "shuffle-tests"
	const stateTestsFolderName = "state-tests"

	var tests []interface{}

	dirs, err := ioutil.ReadDir(yamlDir)
	if err != nil {
		return nil, fmt.Errorf("could not read YAML tests directory: %v", err)
	}
	for _, dir := range dirs {
		files, err := ioutil.ReadDir(path.Join(yamlDir, dir.Name()))
		if err != nil {
			return nil, fmt.Errorf("could not read YAML tests directory: %v", err)
		}
		for _, file := range files {
			filePath := path.Join(yamlDir, dir.Name(), file.Name())
			// #nosec G304
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("could not read YAML file: %v", err)
			}
			switch dir.Name() {
			case forkChoiceTestsFolderName:
				decoded := &backend.ForkChoiceTest{}
				if err := yaml.Unmarshal(data, decoded); err != nil {
					return nil, fmt.Errorf("could not unmarshal YAML file into test struct: %v", err)
				}
				tests = append(tests, decoded)
			case shuffleTestsFolderName:
				decoded := &backend.ShuffleTest{}
				if err := yaml.Unmarshal(data, decoded); err != nil {
					return nil, fmt.Errorf("could not unmarshal YAML file into test struct: %v", err)
				}
				tests = append(tests, decoded)
			case stateTestsFolderName:
				decoded := &backend.StateTest{}
				if err := yaml.Unmarshal(data, decoded); err != nil {
					return nil, fmt.Errorf("could not unmarshal YAML file into test struct: %v", err)
				}
				tests = append(tests, decoded)
			}
		}
	}
	return tests, nil
}

func runTests(tests []interface{}, sb *backend.SimulatedBackend) error {
	for _, tt := range tests {
		switch typedTest := tt.(type) {
		case *backend.ForkChoiceTest:
			log.Infof("Title: %v", typedTest.Title)
			log.Infof("Summary: %v", typedTest.Summary)
			log.Infof("Test Suite: %v", typedTest.TestSuite)
			for _, testCase := range typedTest.TestCases {
				if err := sb.RunForkChoiceTest(testCase); err != nil {
					return fmt.Errorf("chain test failed: %v", err)
				}
			}
			log.Info("Test PASSED")
		case *backend.ShuffleTest:
			log.Infof("Title: %v", typedTest.Title)
			log.Infof("Summary: %v", typedTest.Summary)
			log.Infof("Test Suite: %v", typedTest.TestSuite)
			log.Infof("Fork: %v", typedTest.Fork)
			log.Infof("Version: %v", typedTest.Version)
			for _, testCase := range typedTest.TestCases {
				if err := sb.RunShuffleTest(testCase); err != nil {
					return fmt.Errorf("chain test failed: %v", err)
				}
			}
			log.Info("Test PASSED")
		case *backend.StateTest:
			log.Infof("Title: %v", typedTest.Title)
			log.Infof("Summary: %v", typedTest.Summary)
			log.Infof("Test Suite: %v", typedTest.TestSuite)
			log.Infof("Fork: %v", typedTest.Fork)
			log.Infof("Version: %v", typedTest.Version)
			for _, testCase := range typedTest.TestCases {
				if err := sb.RunStateTransitionTest(testCase); err != nil {
					return fmt.Errorf("chain test failed: %v", err)
				}
			}
			log.Info("Test PASSED")
		default:
			return fmt.Errorf("receive unknown test type: %T", typedTest)
		}
		log.Info("-----------------------------")
	}
	return nil
}

func main() {
	var yamlDir = flag.String("tests-dir", "", "path to directory of yaml tests")
	flag.Parse()

	customFormatter := new(prefixed.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	tests, err := readTestsFromYaml(*yamlDir)
	if err != nil {
		log.Fatalf("Fail to load tests from yaml: %v", err)
	}

	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		log.Fatalf("Could not create backend: %v", err)
	}

	log.Info("----Running Tests----")
	startTime := time.Now()

	err = runTests(tests, sb)
	if err != nil {
		log.Fatalf("Test failed %v", err)
	}

	endTime := time.Now()
	log.Infof("Test Runs Finished In: %v", endTime.Sub(startTime))
}
