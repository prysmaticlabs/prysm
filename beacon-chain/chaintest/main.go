package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"reflect"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/prysmaticlabs/prysm/beacon-chain/chaintest/backend"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: false,
	})
}

func readTestsFromYaml(yamlDir string) (tests []interface{}, configs map[string]interface{}, err error) {
	const forkChoiceTestsFolderName = "fork-choice-tests"
	const shuffleTestsFolderName = "shuffle-tests"
	const stateTestsFolderName = "state-tests"
	const configFilesFolderName = "config"
	configs = make(map[string]interface{})
	dirs, err := ioutil.ReadDir(yamlDir)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read YAML tests directory: %v", err)
	}
	for _, dir := range dirs {
		files, err := ioutil.ReadDir(path.Join(yamlDir, dir.Name()))
		if err != nil {
			return nil, nil, fmt.Errorf("could not read YAML tests directory: %v", err)
		}
		for _, file := range files {
			filePath := path.Join(yamlDir, dir.Name(), file.Name())
			// #nosec G304
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				return nil, nil, fmt.Errorf("could not read YAML file: %v", err)
			}
			switch dir.Name() {
			case forkChoiceTestsFolderName:
				decoded := &backend.ForkChoiceTest{}
				if err := yaml.Unmarshal(data, decoded); err != nil {
					return nil, nil, fmt.Errorf("could not unmarshal YAML file into test struct: %v", err)
				}
				tests = append(tests, decoded)
			case shuffleTestsFolderName:
				decoded := &backend.ShuffleTest{}
				if err := yaml.Unmarshal(data, decoded); err != nil {
					return nil, nil, fmt.Errorf("could not unmarshal YAML file into test struct: %v", err)
				}
				tests = append(tests, decoded)
			case stateTestsFolderName:
				decoded := &backend.StateTest{}
				if err := yaml.Unmarshal(data, decoded); err != nil {
					return nil, nil, fmt.Errorf("could not unmarshal YAML file into test struct: %v", err)
				}
				tests = append(tests, decoded)

			case configFilesFolderName:
				decoded := &params.BeaconChainConfig{}
				if err := yaml.Unmarshal(data, decoded); err != nil {
					return nil, nil, fmt.Errorf("could not unmarshal YAML file into config struct: %v", err)
				}
				fileName := file.Name()
				var extension = filepath.Ext(fileName)
				var name = fileName[0 : len(fileName)-len(extension)]
				configs[name] = decoded
			}
		}
	}
	return tests, configs, nil
}

func runTests(tests []interface{}, configs map[string]interface{}, sb *backend.SimulatedBackend) error {
	for _, tt := range tests {
		switch typedTest := tt.(type) {
		case *backend.ForkChoiceTest:
			log.Infof("Title: %v", typedTest.Title)
			log.Infof("Summary: %v", typedTest.Summary)
			log.Infof("Test Suite: %v", typedTest.TestSuite)
			for _, testCase := range typedTest.TestCases {
				if err := sb.RunForkChoiceTest(testCase); err != nil {
					return fmt.Errorf("fork choice test failed: %v", err)
				}
			}
			log.Info("Test PASSED")
		case *backend.ShuffleTest:
			log.Infof("Title: %v", typedTest.Title)
			log.Infof("Summary: %v", typedTest.Summary)
			log.Infof("Fork: %v", typedTest.Forks)
			log.Infof("Config: %v", typedTest.Config)
			config, ok := configs[string(typedTest.Config)]
			if !ok {
				return errors.New("no config file found for test")
			}
			conf, ok := config.(*params.BeaconChainConfig)
			if !ok {
				return fmt.Errorf("config file is not of type *params.BeaconChainConfig found type: %v", reflect.TypeOf(config))
			}
			params.OverrideBeaconConfig(conf)
			for _, testCase := range typedTest.TestCases {
				if err := sb.RunShuffleTest(testCase); err != nil {
					return fmt.Errorf("shuffle test failed: %v", err)
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

	tests, configs, err := readTestsFromYaml(*yamlDir)
	if err != nil {
		log.Fatalf("Fail to load tests from yaml: %v", err)
	}

	sb, err := backend.NewSimulatedBackend()
	if err != nil {
		log.Fatalf("Could not create backend: %v", err)
	}

	log.Info("----Running Tests----")
	startTime := time.Now()

	err = runTests(tests, configs, sb)
	if err != nil {
		log.Fatalf("Test failed %v", err)
	}

	endTime := time.Now()
	log.Infof("Test Runs Finished In: %v", endTime.Sub(startTime))
}
